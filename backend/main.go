// backend/main.go
package main

import (
	"errors" // ✨ 导入 errors 包
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper" // ✨ 导入 viper 包
)

func main() {
	InitLogger()

	// ✨✨✨ 最终修复点: 对 LoadConfig 的错误进行精确判断 ✨✨✨
	if err := LoadConfig("config.json"); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		// 使用 errors.As 判断错误的具体类型
		// 如果错误确实是 "配置文件未找到"，我们就忽略它，因为这是 Docker 环境的预期行为。
		if errors.As(err, &configFileNotFoundError) {
			slog.Info("config.json 未找到，将仅使用环境变量和默认值进行配置。")
			// 即使文件不存在，也需要执行一次 Unmarshal 来应用环境变量和默认值。
			// 这里我们创建一个临时的空的 Viper 实例来做这件事。
			v := viper.New()
			v.SetEnvPrefix("TEMPSHARE")
			v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
			v.AutomaticEnv()
			if err := v.Unmarshal(&AppConfig); err != nil {
				slog.Error("从环境变量解析配置失败", "error", err)
				os.Exit(1)
			}
		} else {
			// 如果是其他错误 (比如 JSON 格式错误)，这就是一个严重问题，必须退出。
			slog.Error("加载 config.json 时发生严重错误，程序无法启动", "error", err)
			os.Exit(1)
		}
	}

	// 检查 AppConfig 是否已初始化
	if AppConfig == nil || !AppConfig.Initialized {
		// 如果 AppConfig 仍然是 nil，说明 LoadConfig 和后续的 Unmarshal 都没成功
		if AppConfig == nil {
			AppConfig = &Config{} // 创建一个空实例以防 panic
		}
		runInitializationGuide()
		os.Exit(1)
	}

	// --- 后续代码保持不变 ---

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

	clamdScanner, err := NewScanner(AppConfig.ClamdSocket)
	if err != nil {
		slog.Warn("Clamd 扫描器初始化失败，文件扫描功能将不可用。", "error", err)
	}

	go CleanupExpiredFilesTask(db, storage)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	var allowedOrigins []string
	if AppConfig.CORSAllowedOrigins != "" {
		allowedOrigins = strings.Split(AppConfig.CORSAllowedOrigins, ",")
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

	fileHandler := &FileHandler{
		DB:      db,
		Scanner: clamdScanner,
		Storage: storage,
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	apiV1 := router.Group("/api/v1")
	{
		if AppConfig.RateLimit.Enabled {
			limiter := NewIPRateLimiter(AppConfig.RateLimit.Requests, AppConfig.GetRateLimitDuration())
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

	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("后端服务准备启动...", "address", "https://localhost"+serverAddr, "storage", AppConfig.Storage.Type, "database", AppConfig.Database.Type)

	// ✨✨✨ 修复本地开发 HTTPS 问题的关键 ✨✨✨
	// 在本地开发环境中，我们启动一个 TLS 服务器。
	// 这解决了 net::ERR_H2_OR_QUIC_REQUIRED 问题。
	if err := router.RunTLS(serverAddr, "cert.pem", "key.pem"); err != nil {
		slog.Error("无法启动 HTTPS 服务器", "error", err)
		slog.Warn("请确保 backend 目录下存在 cert.pem 和 key.pem 文件。")
		slog.Warn("可以通过运行 'mkcert -install && mkcert localhost' 来生成它们。")
		os.Exit(1)
	}
}

// ... runInitializationGuide 函数保持不变 ...
func runInitializationGuide() {
	fmt.Println("--- 闪传驿站 | TempShare 未初始化 ---")
	fmt.Println("检测到这是首次运行或配置尚未完成。")
	fmt.Println("\n请通过环境变量进行配置。创建一个 `.env` 文件并设置以下变量：")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("# 基础设置")
	fmt.Println("TEMPSHARE_INITIALIZED=true                 # 完成配置后，设置为 true 来启动服务")
	fmt.Println("TEMPSHARE_SERVERPORT=8080                    # 应用监听的端口")
	fmt.Println("TEMPSHARE_CORS_ALLOWED_ORIGINS=https://localhost:5173 # 允许的前端域名, 多个用逗号隔开")

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
