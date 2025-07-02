// backend/handlers.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 临时的本地文件目录，仅用于病毒扫描
var (
	tempScanDir = filepath.Join(os.TempDir(), "tempshare-scans")
)

type VerificationPayload struct {
	VerificationHash string `json:"verificationHash" binding:"required"`
}

type FileHandler struct {
	DB      *gorm.DB
	Scanner *ClamdScanner
	Storage FileStorage // 使用抽象接口
}

func (h *FileHandler) HandleStreamUpload(c *gin.Context) {
	// --- 应用上传大小限制 ---
	maxUploadBytes := AppConfig.MaxUploadSizeMB * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	// --- 读取 Headers (逻辑不变) ---
	fileName, err := url.QueryUnescape(c.GetHeader("X-File-Name"))
	if err != nil || fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效或缺失的文件名 (X-File-Name)"})
		return
	}
	originalSize, err := strconv.ParseInt(c.GetHeader("X-File-Original-Size"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效或缺失的原始文件大小 (X-File-Original-Size)"})
		return
	}
	isEncrypted, _ := strconv.ParseBool(c.GetHeader("X-File-Encrypted"))
	salt := c.GetHeader("X-File-Salt")
	verificationHash := c.GetHeader("X-File-Verification-Hash")
	expiresInSeconds, _ := strconv.ParseInt(c.GetHeader("X-File-Expires-In"), 10, 64)
	downloadOnce, _ := strconv.ParseBool(c.GetHeader("X-File-Download-Once"))

	var expiresAt time.Time
	if expiresInSeconds > 0 {
		expiresAt = time.Now().Add(time.Duration(expiresInSeconds) * time.Second)
	} else {
		expiresAt = time.Now().Add(7 * 24 * time.Hour) // 默认值
	}

	// --- 文件存储与扫描逻辑 (核心修改) ---
	storageKey := uuid.NewString()
	var writtenBytes int64
	var scanStatus, scanResult string

	// 设计决策: 为保证扫描功能在任何存储后端下都可用，
	// 我们先将文件流式传输到本地临时文件进行扫描，然后再上传到最终存储。
	if !isEncrypted && h.Scanner != nil {
		if err := os.MkdirAll(tempScanDir, os.ModePerm); err != nil {
			slog.Error("无法创建临时扫描目录", "path", tempScanDir, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
			return
		}
		tempFilePath := filepath.Join(tempScanDir, storageKey)
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			slog.Error("无法创建临时文件", "path", tempFilePath, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
			return
		}

		// 流式写入临时文件
		writtenBytes, err = io.Copy(tempFile, c.Request.Body)
		tempFile.Close() // 关闭文件以备扫描和读取
		if err != nil {
			os.Remove(tempFilePath)
			// ... (处理 MaxBytesError 的逻辑不变)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "文件上传中断"})
			return
		}

		// 扫描临时文件
		scanStatus, scanResult = h.Scanner.ScanFile(tempFilePath)

		// 从临时文件重新打开并上传到最终存储
		fileReader, err := os.Open(tempFilePath)
		if err != nil {
			os.Remove(tempFilePath)
			slog.Error("无法重新打开临时文件以上传", "path", tempFilePath, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
			return
		}
		defer fileReader.Close()
		defer os.Remove(tempFilePath) // 确保临时文件最终被删除

		_, err = h.Storage.Save(storageKey, fileReader)
		if err != nil {
			slog.Error("无法保存文件到最终存储", "storageType", AppConfig.Storage.Type, "key", storageKey, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存文件"})
			return
		}

	} else {
		// 如果是加密文件或扫描器不可用，直接流式传输到最终存储
		var err error
		writtenBytes, err = h.Storage.Save(storageKey, c.Request.Body)
		if err != nil {
			h.Storage.Delete(storageKey) // 尝试清理
			// ... (处理 MaxBytesError 的逻辑)
			slog.Error("无法保存文件到最终存储", "storageType", AppConfig.Storage.Type, "key", storageKey, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存文件"})
			return
		}
		// 根据情况设置扫描状态
		if isEncrypted {
			scanStatus, scanResult = ScanStatusClean, "端到端加密文件，服务器未扫描"
		} else {
			scanStatus, scanResult = ScanStatusSkipped, "扫描器不可用，已跳过"
		}
	}

	// --- 数据库记录 (逻辑微调) ---
	accessCode, err := h.generateUniqueAccessCode(6)
	if err != nil {
		h.Storage.Delete(storageKey) // 清理已上传的文件
		slog.Error("无法生成分享码", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法生成分享码"})
		return
	}

	newFile := File{
		ID:                uuid.NewString(), // 使用独立的UUID作为主键
		AccessCode:        accessCode,
		Filename:          fileName,
		SizeBytes:         writtenBytes,
		OriginalSizeBytes: originalSize,
		IsEncrypted:       isEncrypted,
		EncryptionSalt:    salt,
		VerificationHash:  verificationHash,
		StorageKey:        storageKey, // 使用 storageKey
		DownloadOnce:      downloadOnce,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		ScanStatus:        scanStatus,
		ScanResult:        scanResult,
	}

	if err := h.DB.Create(&newFile).Error; err != nil {
		h.Storage.Delete(storageKey) // 清理已上传的文件
		slog.Error("无法保存文件记录到数据库", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存文件记录"})
		return
	}
	slog.Info("上传成功", "clientIP", c.ClientIP(), "accessCode", accessCode, "key", storageKey, "scanStatus", scanStatus)
	c.JSON(http.StatusCreated, gin.H{"accessCode": accessCode, "urlPath": fmt.Sprintf("/download/%s", accessCode)})
}

func (h *FileHandler) HandleDownloadFile(c *gin.Context) {
	code := c.Param("code")
	var file File
	if err := h.DB.Where("access_code = ?", code).First(&file).Error; err != nil {
		// ... (错误处理逻辑不变)
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}

	// 检查过期 (在查询后再次检查，更保险)
	if time.Now().After(file.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件已过期"})
		return
	}

	// 加密文件密码验证
	if file.IsEncrypted {
		if c.Request.Method != "POST" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "下载加密文件需要使用 POST 方法"})
			return
		}
		var payload VerificationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "无效的验证请求"})
			return
		}
		if payload.VerificationHash != file.VerificationHash {
			slog.Warn("密码验证失败", "clientIP", c.ClientIP(), "accessCode", file.AccessCode)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "密码错误"})
			return
		}
		slog.Info("密码验证成功，开始下载", "clientIP", c.ClientIP(), "accessCode", file.AccessCode)
	}

	// --- 从存储后端获取文件流并发送 (核心修改) ---
	reader, err := h.Storage.Retrieve(file.StorageKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "物理文件丢失"})
		} else {
			slog.Error("下载失败: 无法从存储后端获取文件", "key", file.StorageKey, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "无法获取文件"})
		}
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(file.SizeBytes, 10))

	_, err = io.Copy(c.Writer, reader)
	if err != nil {
		slog.Error("流式传输文件到客户端时出错", "key", file.StorageKey, "clientIP", c.ClientIP(), "error", err)
	}

	h.handleDownloadOnce(c, file)
}

// 修改为 Handler 的方法，以便访问 h.Storage
func (h *FileHandler) handleDownloadOnce(c *gin.Context, file File) {
	if file.DownloadOnce && c.Writer.Status() == http.StatusOK {
		// 使用 goroutine 异步执行，不阻塞下载响应
		go func(f File) {
			time.Sleep(2 * time.Second) // 等待一会确保连接关闭
			slog.Info("阅后即焚: 文件已被下载，即将销毁", "filename", f.Filename, "key", f.StorageKey)
			if err := h.Storage.Delete(f.StorageKey); err != nil {
				slog.Error("阅后即焚错误: 删除存储对象失败", "key", f.StorageKey, "error", err)
			}
			if err := h.DB.Delete(&File{}, "id = ?", f.ID).Error; err != nil {
				slog.Error("阅后即焚错误: 删除数据库记录失败", "id", f.ID, "error", err)
			}
		}(file)
	}
}

func (h *FileHandler) HandlePreviewFile(c *gin.Context) {
	code := c.Param("code")
	var file File
	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}
	// ... (权限检查逻辑不变)
	if file.IsEncrypted || file.ScanStatus == ScanStatusInfected {
		c.JSON(http.StatusForbidden, gin.H{"message": "文件无法预览"})
		return
	}

	reader, err := h.Storage.Retrieve(file.StorageKey)
	if err != nil {
		slog.Error("预览错误: 无法读取文件", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}
	defer reader.Close()

	// 需要读取一部分来判断 Content-Type
	buffer := make([]byte, 512)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "读取文件时出错"})
		return
	}

	ext := filepath.Ext(file.Filename)
	var contentType string

	// Map of Office extensions to their MIME types
	officeMimeTypes := map[string]string{
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	// Check if the file is an Office document
	if mime, isOffice := officeMimeTypes[ext]; isOffice {
		contentType = mime
		// For Office documents, we do not set Content-Disposition
	} else {
		// For other files, detect content type and set Content-Disposition to inline
		contentType = http.DetectContentType(buffer[:n])
		c.Header("Content-Disposition", fmt.Sprintf(`inline; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	}

	c.Header("Content-Type", contentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Length", strconv.FormatInt(file.SizeBytes, 10))

	// 先把已读的 buffer 写回去，再把剩下的流拷贝过去
	c.Writer.Write(buffer[:n])
	io.Copy(c.Writer, reader)
}

// 其他 Handler (HandleGetFileMeta, HandleGetPublicFiles, HandleReport, HandlePreviewDataURI, generateUniqueAccessCode) 基本不变
// HandlePreviewDataURI 也需要修改为从 h.Storage 读取
func (h *FileHandler) HandlePreviewDataURI(c *gin.Context) {
	code := c.Param("code")
	var file File

	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}
	if file.IsEncrypted || file.ScanStatus == ScanStatusInfected {
		c.JSON(http.StatusForbidden, gin.H{"message": "文件无法预览"})
		return
	}

	reader, err := h.Storage.Retrieve(file.StorageKey)
	if err != nil {
		slog.Error("Data URI 预览错误: 无法读取文件", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}
	defer reader.Close()

	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		slog.Error("Data URI 预览错误: 读取流失败", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(fileBytes)
	contentType := http.DetectContentType(fileBytes)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)

	c.JSON(http.StatusOK, gin.H{
		"dataUri": dataURI,
	})
}

// --- 不变的 Handler 函数 ---
func (h *FileHandler) HandleGetFileMeta(c *gin.Context) {
	code := c.Param("code")
	var file File
	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}
	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) HandleGetPublicFiles(c *gin.Context) {
	var files []File
	result := h.DB.Select("access_code", "filename", "size_bytes", "expires_at", "is_encrypted").
		Where("expires_at > ? AND is_encrypted = false AND download_once = false", time.Now()).
		Order("created_at desc").Limit(20).Find(&files)
	if result.Error != nil {
		slog.Error("查询公开文件列表失败", "error", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询公开文件列表失败"})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (h *FileHandler) HandleReport(c *gin.Context) {
	var reportData struct {
		AccessCode string `json:"accessCode" binding:"required"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&reportData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的举报请求"})
		return
	}
	report := Report{AccessCode: reportData.AccessCode, Reason: reportData.Reason, ReporterIP: c.ClientIP()}
	if err := h.DB.Create(&report).Error; err != nil {
		slog.Error("无法提交举报到数据库", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法提交举报，请稍后再试"})
		return
	}
	slog.Info("收到举报", "clientIP", c.ClientIP(), "accessCode", report.AccessCode, "reason", report.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "您的举报已收到，感谢您的帮助！我们将会尽快处理。"})
}

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (h *FileHandler) generateUniqueAccessCode(length int) (string, error) {
	for i := 0; i < 20; i++ {
		buffer := make([]byte, length)
		if _, err := rand.Read(buffer); err != nil {
			return "", err
		}
		for i := 0; i < length; i++ {
			buffer[i] = codeChars[int(buffer[i])%len(codeChars)]
		}
		code := string(buffer)
		var count int64
		h.DB.Model(&File{}).Where("access_code = ?", code).Count(&count)
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("无法在20次尝试内生成唯一的便捷码")
}

// App Info Handler
func HandleGetAppInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"publicHost": AppConfig.PublicHost,
	})
}
