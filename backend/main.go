// backend/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	if err := LoadConfig("config.json"); err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 初始化目录
	if err := os.MkdirAll("tempshare-files", os.ModePerm); err != nil {
		log.Fatalf("无法创建文件目录: %v", err)
	}

	// 连接数据库
	db, err := ConnectDatabase(AppConfig.DatabasePath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化 Clamd 扫描器
	clamdScanner, err := NewScanner(AppConfig.ClamdSocket)
	if err != nil {
		log.Printf("‼️ 警告: Clamd 扫描器初始化失败: %v。文件扫描功能将不可用，但服务器将继续运行。", err)
	}

	// 启动后台清理任务
	go CleanupExpiredFilesTask(db)

	// 设置 Gin 路由
	router := gin.Default()

	// 为了在代理后也能正确获取IP，信任X-Forwarded-For等头
	router.SetTrustedProxies(nil)

	corsConfig := cors.Config{
		AllowOrigins:     []string{"https://localhost:5173", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-File-Name", "X-File-Original-Size", "X-File-Encrypted", "X-File-Salt", "X-File-Expires-In", "X-File-Download-Once", "X-Requested-With", "X-File-Verification-Hash"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// 创建 Handler 实例
	fileHandler := &FileHandler{
		DB:      db,
		Scanner: clamdScanner,
	}

	// 注册路由
	apiV1 := router.Group("/api/v1")
	{
		if AppConfig.RateLimit.Enabled {
			limiter := NewIPRateLimiter(
				AppConfig.RateLimit.Requests,
				time.Duration(AppConfig.RateLimit.DurationMinutes)*time.Minute,
			)
			uploadAndReportGroup := apiV1.Group("/")
			uploadAndReportGroup.Use(limiter.RateLimitMiddleware())
			{
				uploadAndReportGroup.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
				uploadAndReportGroup.POST("/report", fileHandler.HandleReport)
			}
			log.Printf("🛡️ 已启用上传/举报速率限制: 每 %d 分钟 %d 次请求", AppConfig.RateLimit.DurationMinutes, AppConfig.RateLimit.Requests)

			apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
			apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)
			apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
			// ✨✨✨ 核心修改：添加新的预览数据接口路由 ✨✨✨
			apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)

		} else {
			apiV1.POST("/uploads/stream-complete", fileHandler.HandleStreamUpload)
			apiV1.GET("/files/meta/:code", fileHandler.HandleGetFileMeta)
			apiV1.GET("/files/public", fileHandler.HandleGetPublicFiles)
			apiV1.POST("/report", fileHandler.HandleReport)
			apiV1.GET("/preview/:code", fileHandler.HandlePreviewFile)
			// ✨✨✨ 核心修改：添加新的预览数据接口路由 ✨✨✨
			apiV1.GET("/preview/data-uri/:code", fileHandler.HandlePreviewDataURI)
		}
	}

	router.GET("/data/:code", fileHandler.HandleDownloadFile)
	router.POST("/data/:code", fileHandler.HandleDownloadFile)

	// 启动服务器
	serverAddr := ":" + AppConfig.ServerPort
	log.Printf("🚀 后端服务已启动，监听 HTTPS 端口: %s", AppConfig.ServerPort)
	if err := http.ListenAndServeTLS(serverAddr, "cert.pem", "key.pem", router); err != nil {
		log.Fatalf("无法启动 HTTPS 服务器: %v", err)
	}
}
