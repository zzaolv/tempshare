// backend/handlers.go
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"tempshare/storage" // 引入新的 storage 包
)

type VerificationPayload struct {
	VerificationHash string `json:"verificationHash" binding:"required"`
}

// FileHandler 现在包含一个 StorageProvider
type FileHandler struct {
	DB      *gorm.DB
	Scanner *ClamdScanner
	Storage storage.StorageProvider
}

func (h *FileHandler) HandleStreamUpload(c *gin.Context) {
	// --- 应用上传大小限制 ---
	maxUploadBytes := AppConfig.MaxUploadSizeMB * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	// --- 读取 Headers ---
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
		expiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	// --- 文件存储逻辑 (使用 StorageProvider) ---
	finalFileId := uuid.NewString()
	writtenBytes, err := h.Storage.Save(c.Request.Context(), finalFileId, c.Request.Body)
	if err != nil {
		// 删除可能已创建的不完整文件
		_ = h.Storage.Delete(c.Request.Context(), finalFileId)

		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			slog.Warn("上传文件过大",
				"clientIP", c.ClientIP(),
				"limitBytes", maxBytesError.Limit,
				"maxConfigMB", AppConfig.MaxUploadSizeMB)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": fmt.Sprintf("文件大小不能超过 %d MB", AppConfig.MaxUploadSizeMB)})
			return
		}
		slog.Error("文件存储失败", "key", finalFileId, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "文件上传中断或存储失败"})
		return
	}

	// --- 病毒扫描逻辑 ---
	var scanStatus, scanResult string
	physicalPath := h.Storage.GetFullPath(finalFileId)

	const twentyFourHoursInSeconds = 24 * 60 * 60
	if isEncrypted {
		scanStatus, scanResult = ScanStatusClean, "端到端加密文件，服务器未扫描"
	} else if expiresInSeconds > 0 && expiresInSeconds < twentyFourHoursInSeconds {
		scanStatus, scanResult = ScanStatusSkipped, "短期文件，已跳过病毒扫描"
		slog.Info("短期文件，跳过扫描", "filename", fileName, "fileID", finalFileId)
	} else if physicalPath != "" { // 仅当是本地存储时才扫描
		scanStatus, scanResult = h.Scanner.ScanFile(physicalPath)
	} else {
		scanStatus, scanResult = ScanStatusSkipped, "非本地存储，已跳过病毒扫描"
	}

	// --- 数据库记录 ---
	accessCode, err := h.generateUniqueAccessCode(6)
	if err != nil {
		_ = h.Storage.Delete(c.Request.Context(), finalFileId)
		slog.Error("无法生成分享码", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法生成分享码"})
		return
	}

	newFile := File{
		ID:                finalFileId,
		AccessCode:        accessCode,
		Filename:          fileName,
		SizeBytes:         writtenBytes,
		OriginalSizeBytes: originalSize,
		IsEncrypted:       isEncrypted,
		EncryptionSalt:    salt,
		VerificationHash:  verificationHash,
		StorageKey:        finalFileId, // 现在只存ID，而不是完整路径
		DownloadOnce:      downloadOnce,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		ScanStatus:        scanStatus,
		ScanResult:        scanResult,
	}

	if err := h.DB.Create(&newFile).Error; err != nil {
		_ = h.Storage.Delete(c.Request.Context(), finalFileId)
		slog.Error("无法保存文件记录到数据库", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存文件记录"})
		return
	}

	slog.Info("流式上传成功",
		"clientIP", c.ClientIP(),
		"accessCode", accessCode,
		"fileID", finalFileId,
		"scanStatus", scanStatus)
	c.JSON(http.StatusCreated, gin.H{"accessCode": accessCode, "urlPath": fmt.Sprintf("/download/%s", accessCode)})
}

func (h *FileHandler) HandleGetFileMeta(c *gin.Context) {
	code := c.Param("code")
	var file File
	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}
	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) HandleDownloadFile(c *gin.Context) {
	code := c.Param("code")
	var file File

	if err := h.DB.Where("access_code = ?", code).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
		} else {
			slog.Error("查询文件时发生数据库错误", "accessCode", code, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "查询文件时发生错误"})
		}
		return
	}

	if time.Now().After(file.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件已过期"})
		return
	}

	// 验证逻辑
	if file.IsEncrypted {
		if c.Request.Method != "POST" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "下载加密文件需要使用 POST 方法并提供验证信息"})
			return
		}
		var payload VerificationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "无效的验证请求"})
			return
		}
		if payload.VerificationHash != file.VerificationHash {
			slog.Warn("密码验证失败", "clientIP", c.ClientIP(), "accessCode", file.AccessCode)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "密码错误或文件已损坏"})
			return
		}
		slog.Info("密码验证成功，开始下载", "clientIP", c.ClientIP(), "accessCode", file.AccessCode)
	}

	// 流式下载
	reader, err := h.Storage.Open(c.Request.Context(), file.StorageKey)
	if err != nil {
		slog.Error("无法打开文件进行下载", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件"})
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.Header("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	c.Header("Content-Type", "application/octet-stream")

	// 使用 Stream 方法进行流式响应
	_, err = io.Copy(c.Writer, reader)
	if err != nil {
		slog.Error("下载期间发生流错误", "storageKey", file.StorageKey, "clientIP", c.ClientIP(), "error", err)
		// 此时可能已经发送了部分响应头，不能再写入JSON错误，只能中断连接
	} else {
		// 只有在成功写入后才处理“阅后即焚”
		h.handleDownloadOnce(c, file)
	}
}

func (h *FileHandler) handleDownloadOnce(c *gin.Context, file File) {
	if file.DownloadOnce {
		go func(db *gorm.DB, st storage.StorageProvider, f File) {
			time.Sleep(2 * time.Second)
			slog.Info("阅后即焚: 文件已被下载，即将销毁", "filename", f.Filename, "id", f.ID)
			if err := st.Delete(context.Background(), f.StorageKey); err != nil {
				slog.Error("阅后即焚错误: 删除存储文件失败", "id", f.ID, "storageKey", f.StorageKey, "error", err)
			}
			if err := db.Delete(&File{}, "id = ?", f.ID).Error; err != nil {
				slog.Error("阅后即焚错误: 删除数据库记录失败", "id", f.ID, "error", err)
			}
		}(h.DB, h.Storage, file)
	}
}

// ... 其他 handler 保持类似逻辑，如果需要操作文件，就通过 h.Storage ...
// HandlePreviewFile 和 HandlePreviewDataURI 尤其需要修改，从 h.Storage.Open 读取数据

func (h *FileHandler) HandlePreviewFile(c *gin.Context) {
	code := c.Param("code")
	var file File

	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}

	if file.IsEncrypted {
		c.JSON(http.StatusForbidden, gin.H{"message": "加密文件无法在服务器端预览"})
		return
	}
	if file.ScanStatus == ScanStatusInfected {
		c.JSON(http.StatusForbidden, gin.H{"message": "检测到威胁，已禁止预览此文件"})
		return
	}

	reader, err := h.Storage.Open(c.Request.Context(), file.StorageKey)
	if err != nil {
		slog.Error("预览错误: 无法打开文件", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}
	defer reader.Close()

	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		slog.Error("预览错误: 无法读取文件内容到内存", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}

	contentType := http.DetectContentType(fileBytes)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, fileBytes)
}

func (h *FileHandler) HandlePreviewDataURI(c *gin.Context) {
	code := c.Param("code")
	var file File

	if err := h.DB.Where("access_code = ? AND expires_at > ?", code, time.Now()).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在或已过期"})
		return
	}
	if file.IsEncrypted {
		c.JSON(http.StatusForbidden, gin.H{"message": "加密文件无法在服务器端预览"})
		return
	}
	if file.ScanStatus == ScanStatusInfected {
		c.JSON(http.StatusForbidden, gin.H{"message": "检测到威胁，已禁止预览此文件"})
		return
	}

	reader, err := h.Storage.Open(c.Request.Context(), file.StorageKey)
	if err != nil {
		slog.Error("Data URI 预览错误: 无法打开文件", "storageKey", file.StorageKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}
	defer reader.Close()

	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		slog.Error("Data URI 预览错误: 无法读取文件内容", "storageKey", file.StorageKey, "error", err)
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

// generateUniqueAccessCode, HandleGetPublicFiles, HandleReport 保持不变
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
		h.DB.Model(&File{}).Where("access_code = ? AND expires_at > ?", code, time.Now()).Count(&count)
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("无法在20次尝试内生成唯一的便捷码")
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
