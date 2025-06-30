// backend/main.go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// --- 初始化和配置加载 ---
	InitLogger()

	if err := LoadConfig("config.json"); err != nil {
		slog.Error("无法加载配置", "error", err)
		os.Exit(1)
	}

	// --- 初始化检查 ---
	if !AppConfig.Initialized {
		runInitializationGuide()
		os.Exit(1) // 退出，强制用户进行配置
	}

	// --- 依赖初始化 ---
	storage, err := NewFileStorage(AppConfig.Storage)
	if err != nil {
		slog.Error("存储后端初始化失败", "error", err)
		os.Exit(1)
	}

	db, err := ConnectDatabase(AppConfig.Database)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	// 从环境变量或配置文件读取 Clamd 地址
	clamdSocket := os.Getenv("TEMPSHARE_CLAMDSOCKET")
	if clamdSocket == "" && AppConfig.ClamdSocket != "" {
		clamdSocket = AppConfig.ClamdSocket
	}

	clamdScanner, err := NewScanner(clamdSocket)
	if err != nil {
		slog.Warn("Clamd 扫描器初始化失败，文件扫描功能将不可用。", "error", err)
	}

	// --- 启动后台任务 ---
	go CleanupExpiredFilesTask(db, storage)

	// --- 设置 Gin 路由 ---
	router := gin.Default()
	router.SetTrustedProxies(nil) // 信任所有代理，以便在Nginx后获取真实IP

	// CORS 设置需要允许你的前端域名
	// 在生产环境中，你应该将 AllowOrigins 设置为你的实际前端域名
	// 例如: []string{"https://yourdomain.com"}
	// 你也可以从环境变量中读取
	allowedOrigins := os.Getenv("TEMPSHARE_CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "https://localhost:5173,http://localhost:5173" // 默认开发环境
	}

	corsConfig := cors.Config{
		AllowOrigins:     []string{allowedOrigins},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-File-Name", "X-File-Original-Size", "X-File-Encrypted", "X-File-Salt", "X-File-Expires-In", "X-File-Download-Once", "X-Requested-With", "X-File-Verification-Hash"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// 创建 Handler 实例，并注入依赖
	fileHandler := &FileHandler{
		DB:      db,
		Scanner: clamdScanner,
		Storage: storage,
	}

	// --- 健康检查路由 ---
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- 注册API路由 ---
	apiV1 := router.Group("/api/v1")
	{
		if AppConfig.RateLimit.Enabled {
			limiter := NewIPRateLimiter(
				AppConfig.RateLimit.Requests,
				AppConfig.GetRateLimitDuration(),
			)
			uploadAndReportGroup := apiV1.Group("/")
			uploadAndReportGroup.Use(limiter.RateLimitMiddleware())
			{
				uploadAndReportGroup.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
				uploadAndReportGroup.POST("/report", fileHandler.HandleReport)
			}
			slog.Info("已启用上传/举报速率限制",
				"requests", AppConfig.RateLimit.Requests,
				"durationMinutes", AppConfig.RateLimit.DurationMinutes)
		} else {
			slog.Warn("速率限制已禁用")
			apiV1.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
			apiV1.POST("/report", fileHandler.HandleReport)
		}

		apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
		apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)
		apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
		apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)
	}

	dataGroup := router.Group("/data/:code")
	{
		dataGroup.GET("", fileHandler.HandleDownloadFile)
		dataGroup.POST("", fileHandler.HandleDownloadFile)
	}

	// --- 启动服务器 ---
	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("后端服务准备启动...", "address", "http://localhost"+serverAddr, "storage", AppConfig.Storage.Type, "database", AppConfig.Database.Type)

	// 在Docker环境中，TLS终止通常由反向代理（如Nginx）处理，
	// 所以应用本身运行在HTTP模式下更简单、更通用。
	// 我们移除 ListenAndServeTLS，改用 ListenAndServe。
	// 您的NPM会处理HTTPS。
	if err := router.Run(serverAddr); err != nil {
		slog.Error("无法启动 HTTP 服务器", "error", err)
		os.Exit(1)
	}
}

// runInitializationGuide 打印设置指南
func runInitializationGuide() {
	fmt.Println("--- 闪传驿站 | TempShare 未初始化 ---")
	fmt.Println("检测到这是首次运行或配置尚未完成。")
	fmt.Println("\n请通过环境变量进行配置。创建一个 `.env` 文件并设置以下变量：")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("# 基础设置")
	fmt.Println("TEMPSHARE_INITIALIZED=true                 # 完成配置后，设置为 true 来启动服务")
	fmt.Println("TEMPSHARE_SERVERPORT=8080                    # 应用监听的端口")
	fmt.Println("TEMPSHARE_CORS_ALLOWED_ORIGINS=https://your-frontend.com # 允许的前端域名")

	fmt.Println("\n# 数据库配置 (选择一种)")
	fmt.Println("## SQLite (默认)")
	fmt.Println("TEMPSHARE_DATABASE_TYPE=sqlite")
	fmt.Println("TEMPSHARE_DATABASE_DSN=data/tempshare.db   # 推荐放在持久化卷中")
	fmt.Println("\n## MySQL")
	fmt.Println("# TEMPSHARE_DATABASE_TYPE=mysql")
	fmt.Println("# TEMPSHARE_DATABASE_DSN='user:pass@tcp(mysql_host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local'")
	fmt.Println("\n## PostgreSQL")
	fmt.Println("# TEMPSHARE_DATABASE_TYPE=postgres")
	fmt.Println("# TEMPSHARE_DATABASE_DSN='host=postgres_host user=user password=pass dbname=tempshare port=5432 sslmode=disable'")

	fmt.Println("\n# 存储配置 (选择一种)")
	fmt.Println("## 本地存储 (默认)")
	fmt.Println("TEMPSHARE_STORAGE_TYPE=local")
	fmt.Println("TEMPSHARE_STORAGE_LOCALPATH=data/files     # 推荐放在持久化卷中")
	fmt.Println("\n## S3/MinIO 对象存储")
	fmt.Println("# TEMPSHARE_STORAGE_TYPE=s3")
	fmt.Println("# TEMPSHARE_STORAGE_S3_ENDPOINT=http://minio:9000")
	fmt.Println("# TEMPSHARE_STORAGE_S3_REGION=us-east-1")
	fmt.Println("# TEMPSHARE_STORAGE_S3_BUCKET=tempshare-bucket")
	fmt.Println("# TEMPSHARE_STORAGE_S3_ACCESSKEYID=your_access_key")
	fmt.Println("# TEMPSHARE_STORAGE_S3_SECRETACCESSKEY=your_secret_key")
	fmt.Println("# TEMPSHARE_STORAGE_S3_USEPATHSTYLE=true")

	fmt.Println("\n# (可选) ClamAV 病毒扫描")
	fmt.Println("# TEMPSHARE_CLAMDSOCKET=tcp://clamav_host:3310")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("\n配置完成后，请确保 TEMPSHARE_INITIALIZED=true，然后重新启动服务。")
}
