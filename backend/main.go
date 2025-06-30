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
	"gorm.io/gorm"

	"tempshare/storage" // 引入 storage
)

func main() {
	// 1. 初始化日志
	InitLogger()

	// 2. 加载配置
	if err := LoadConfig("config.json"); err != nil {
		slog.Error("无法加载配置", "error", err)
		os.Exit(1)
	}

	// --- 检查是否需要初始化 ---
	if !AppConfig.IsInitialized() {
		slog.Warn("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		slog.Warn("!! 系统配置不完整，正在以 [初始化模式] 启动。")
		slog.Warn("!! 请访问前端页面完成设置。")
		slog.Warn("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		runInitServer()
		return
	}

	// 3. 初始化存储提供者
	storageProvider, err := initStorageProvider(AppConfig.Storage)
	if err != nil {
		slog.Error("存储提供者初始化失败", "error", err)
		os.Exit(1)
	}

	// 4. 连接数据库
	db, err := ConnectDatabase(AppConfig.Database)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	// 5. 初始化 Clamd 扫描器
	clamdScanner, err := NewScanner(AppConfig.ClamdSocket)
	if err != nil {
		slog.Warn("Clamd 扫描器初始化失败，文件扫描功能将不可用。", "error", err)
	}

	// 6. 启动后台清理任务
	go CleanupExpiredFilesTask(db, storageProvider)

	// 7. 设置 Gin 路由
	router := setupRouter(db, clamdScanner, storageProvider)

	// 8. 启动服务器
	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("后端服务已启动", "address", "http://localhost"+serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		slog.Error("无法启动 HTTP 服务器", "error", err)
		os.Exit(1)
	}
}

func initStorageProvider(cfg StorageConfig) (storage.StorageProvider, error) {
	switch cfg.Type {
	case "local":
		slog.Info("使用 [Local] 存储提供者", "path", cfg.Local.Path)
		return storage.NewLocalStorage(cfg.Local.Path)
	case "s3":
		slog.Error("[S3] 存储提供者暂未实现")
		return nil, fmt.Errorf("[S3] 存储提供者暂未实现")
	case "webdav":
		slog.Error("[WebDAV] 存储提供者暂未实现")
		return nil, fmt.Errorf("[WebDAV] 存储提供者暂未实现")
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.Type)
	}
}

func setupRouter(db *gorm.DB, scanner *ClamdScanner, sp storage.StorageProvider) *gin.Engine {
	router := gin.Default()
	router.SetTrustedProxies(nil)

	// CORS 配置 (与之前基本相同)
	corsConfig := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost", "http://127.0.0.1"},
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
		Scanner: scanner,
		Storage: sp,
	}

	apiV1 := router.Group("/api/v1")
	{
		// 初始化状态检查端点，即使在完整模式下也保留，以便前端检查
		apiV1.GET("/init/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"needsInit": false})
		})

		// 速率限制
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

		// 其他 GET 端点
		apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
		apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)
		apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
		apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)
	}

	// 下载路由
	downloadGroup := router.Group("/data/:code")
	{
		downloadGroup.GET("", fileHandler.HandleDownloadFile)
		downloadGroup.POST("", fileHandler.HandleDownloadFile)
	}

	return router
}

// runInitServer 启动一个简化的服务器，仅用于处理初始化
func runInitServer() {
	router := gin.Default()
	router.Use(cors.Default()) // 允许所有跨域请求以便于设置

	initHandler := &InitHandler{}

	apiV1 := router.Group("/api/v1/init")
	{
		apiV1.GET("/status", initHandler.GetStatus)
		apiV1.POST("/validate", initHandler.ValidateConfig)
	}

	// 捕获所有其他路由，提示需要初始化
	router.NoRoute(func(c *gin.Context) {
		// 如果是API请求，返回JSON
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"message": "服务正在等待初始化配置",
				"code":    "NEEDS_INITIALIZATION",
			})
			return
		}
		// 其他情况可以返回一个简单的HTML页面或重定向，但前端会处理好
		c.JSON(http.StatusOK, gin.H{"needsInit": true})
	})

	serverAddr := ":" + AppConfig.ServerPort
	slog.Info("初始化服务器已启动", "address", "http://localhost"+serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		slog.Error("无法启动初始化服务器", "error", err)
		os.Exit(1)
	}
}
