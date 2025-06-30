// backend/main.go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	InitLogger()

	// LoadConfig 现在会优雅地处理文件不存在的情况，
	// 并且在真正遇到解析错误时才会返回 err。
	if err := LoadConfig("config.json"); err != nil {
		slog.Error("加载配置时发生严重错误，程序无法启动", "error", err)
		os.Exit(1)
	}

	// 检查配置是否已初始化，这是启动服务的先决条件。
	if !AppConfig.Initialized {
		runInitializationGuide()
		os.Exit(1)
	}

	// 初始化存储后端
	storage, err := NewFileStorage(AppConfig.Storage)
	if err != nil {
		slog.Error("存储后端初始化失败", "error", err)
		os.Exit(1)
	}

	// 初始化数据库
	db, err := ConnectDatabase(AppConfig.Database)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	// ✨ 修改点: 直接从 AppConfig 读取 ClamdSocket 配置
	clamdScanner, err := NewScanner(AppConfig.ClamdSocket)
	if err != nil {
		// NewScanner 内部已经记录了详细日志，这里只记录一个警告即可
		slog.Warn("Clamd 扫描器初始化失败，文件扫描功能将不可用。", "error", err)
	}

	// 启动后台清理任务
	go CleanupExpiredFilesTask(db, storage)

	// 初始化 Gin 路由器
	router := gin.Default()
	router.SetTrustedProxies(nil) // 根据您的部署环境决定是否信任代理

	// 配置 CORS (跨域资源共享)
	// ✨ 修改点: 直接从 AppConfig 读取 CORS 配置
	var allowedOrigins []string
	allowedOriginsEnv := AppConfig.CORSAllowedOrigins
	if allowedOriginsEnv != "" {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	}
	slog.Info("CORS Allowed Origins", "origins", allowedOrigins)

	corsConfig := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-File-Name", "X-File-Original-Size", "X-File-Encrypted", "X-File-Salt", "X-File-Expires-In", "X-File-Download-Once", "X-Requested-With", "X-File-Verification-Hash"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// 初始化 API 处理器
	fileHandler := &FileHandler{
		DB:      db,
		Scanner: clamdScanner,
		Storage: storage,
	}

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 路由组
	apiV1 := router.Group("/api/v1")
	{
		// 配置速率限制
		if AppConfig.RateLimit.Enabled {
			limiter := NewIPRateLimiter(AppConfig.RateLimit.Requests, AppConfig.GetRateLimitDuration())
			// 对上传和举报接口应用速率限制
			uploadAndReportGroup := apiV1.Group("/")
			uploadAndReportGroup.Use(limiter.RateLimitMiddleware())
			{
				uploadAndReportGroup.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
				uploadAndReportGroup.POST("/report", fileHandler.HandleReport)
			}
			slog.Info("已启用上传/举报速率限制", "requests", AppConfig.RateLimit.Requests, "durationMinutes", AppConfig.RateLimit.DurationMinutes)
		} else {
			slog.Warn("速率限制已禁用")
			apiV1.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
			apiV1.POST("/report", fileHandler.HandleReport)
		}

		// 公共文件 API
		apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
		apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)

		// 文件预览 API
		apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
		apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)
	}

	// 文件下载路由组 (支持加密文件的POST验证)
	dataGroup := router.Group("/data/:code")
	{
		dataGroup.GET("", fileHandler.HandleDownloadFile)
		dataGroup.POST("", fileHandler.HandleDownloadFile)
	}

	// 启动服务器
	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("后端服务准备启动...", "address", "http://localhost"+serverAddr, "storage", AppConfig.Storage.Type, "database", AppConfig.Database.Type)

	if err := router.Run(serverAddr); err != nil {
		slog.Error("无法启动 HTTP 服务器", "error", err)
		os.Exit(1)
	}
}

// runInitializationGuide 在配置未初始化时显示引导信息
func runInitializationGuide() {
	fmt.Println("--- 闪传驿站 | TempShare 未初始化 ---")
	fmt.Println("检测到这是首次运行或配置尚未完成。")
	fmt.Println("\n请通过环境变量进行配置。创建一个 `.env` 文件并设置以下变量：")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("# 基础设置")
	fmt.Println("TEMPSHARE_INITIALIZED=true                 # 完成配置后，设置为 true 来启动服务")
	fmt.Println("TEMPSHARE_SERVERPORT=8080                    # 应用监听的端口")
	fmt.Println("TEMPSHARE_CORS_ALLOWED_ORIGINS=https://your-frontend.com # 允许的前端域名, 多个用逗号隔开")

	fmt.Println("\n# 数据库配置 (选择一种)")
	fmt.Println("## SQLite (默认)")
	fmt.Println("TEMPSHARE_DATABASE_TYPE=sqlite")
	fmt.Println("TEMPSHARE_DATABASE_DSN=data/tempshare.db   # 推荐放在持久化卷中")

	fmt.Println("\n# 存储配置 (选择一种)")
	fmt.Println("## 本地存储 (默认)")
	fmt.Println("TEMPSHARE_STORAGE_TYPE=local")
	fmt.Println("TEMPSHARE_STORAGE_LOCALPATH=data/files     # 推荐放在持久化卷中")

	fmt.Println("\n# (可选) ... 其他配置项 ...")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("\n配置完成后，请确保 TEMPSHARE_INITIALIZED=true，然后重新启动服务。")
}
