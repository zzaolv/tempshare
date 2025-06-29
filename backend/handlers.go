// backend/handlers.go
package main

import (
	"crypto/rand"
	"encoding/base64" // ✨✨✨ 核心修改：导入 base64 包 ✨✨✨
	"errors"
	"fmt"
	"io"
	"log"
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

var (
	finalFileDir = filepath.Join("tempshare-files")
)

type VerificationPayload struct {
	VerificationHash string `json:"verificationHash" binding:"required"`
}

type FileHandler struct {
	DB      *gorm.DB
	Scanner *ClamdScanner
}

// ... HandleStreamUpload, HandleGetFileMeta, HandleDownloadFile, handleDownloadOnce, HandleGetPublicFiles, HandleReport ...
// (这些函数保持不变，为节省篇幅此处省略，请保留您文件中的这些函数)
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

	// --- 文件存储逻辑 ---
	finalFileId := uuid.NewString()
	finalFilePath := filepath.Join(finalFileDir, finalFileId)
	outFile, err := os.Create(finalFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法创建文件存储"})
		return
	}
	defer outFile.Close()

	writtenBytes, err := io.Copy(outFile, c.Request.Body)
	if err != nil {
		outFile.Close()
		os.Remove(finalFilePath)
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			log.Printf("🚫 上传文件过大! IP: %s, 限制: %d MB", c.ClientIP(), AppConfig.MaxUploadSizeMB)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": fmt.Sprintf("文件大小不能超过 %d MB", AppConfig.MaxUploadSizeMB)})
			return
		}
		log.Printf("⚠️ 文件上传中断! IP: %s, Error: %v", c.ClientIP(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "文件上传中断"})
		return
	}

	// --- 病毒扫描逻辑 ---
	var scanStatus, scanResult string
	const twentyFourHoursInSeconds = 24 * 60 * 60
	if isEncrypted {
		scanStatus, scanResult = ScanStatusClean, "端到端加密文件，服务器未扫描"
	} else if expiresInSeconds > 0 && expiresInSeconds < twentyFourHoursInSeconds {
		scanStatus, scanResult = ScanStatusSkipped, "短期文件，已跳过病毒扫描"
		log.Printf("⏩ 文件 %s (ID: %s) 为短期文件，跳过扫描。", fileName, finalFileId)
	} else {
		scanStatus, scanResult = h.Scanner.ScanFile(finalFilePath)
	}

	// --- 数据库记录 ---
	accessCode, err := h.generateUniqueAccessCode(6)
	if err != nil {
		os.Remove(finalFilePath)
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
		StorageKey:        finalFilePath,
		DownloadOnce:      downloadOnce,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		ScanStatus:        scanStatus,
		ScanResult:        scanResult,
	}

	if err := h.DB.Create(&newFile).Error; err != nil {
		os.Remove(finalFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存文件记录"})
		return
	}

	log.Printf("🎉 流式上传成功! IP: %s, AccessCode: %s, FileID: %s, ScanStatus: %s", c.ClientIP(), accessCode, finalFileId, scanStatus)
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

	// 1. 获取文件元数据
	if err := h.DB.Where("access_code = ?", code).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "查询文件时发生错误"})
		}
		return
	}

	// 2. 检查文件是否已过期
	if time.Now().After(file.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"message": "文件已过期"})
		return
	}

	// 3. 根据是否加密执行不同逻辑
	if !file.IsEncrypted {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
		c.File(file.StorageKey)
		h.handleDownloadOnce(c, file)
		return
	}

	// --- 加密文件逻辑 (必须是 POST 请求) ---
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "下载加密文件需要使用 POST 方法并提供验证信息"})
		return
	}

	var payload VerificationPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的验证请求"})
		return
	}

	// 4. 验证密码哈希
	if payload.VerificationHash != file.VerificationHash {
		log.Printf("🚫 密码错误! IP: %s, AccessCode: %s", c.ClientIP(), file.AccessCode)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "密码错误或文件已损坏"})
		return
	}

	// 5. 验证成功，提供文件
	log.Printf("✅ 密码验证成功, 开始下载. IP: %s, AccessCode: %s", c.ClientIP(), file.AccessCode)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.File(file.StorageKey)
	h.handleDownloadOnce(c, file)
}

func (h *FileHandler) handleDownloadOnce(c *gin.Context, file File) {
	if file.DownloadOnce && c.Writer.Status() == http.StatusOK {
		go func(db *gorm.DB, f File) {
			time.Sleep(2 * time.Second)
			log.Printf("🔥 阅后即焚: 文件 %s (ID: %s) 已被下载，即将销毁。", f.Filename, f.ID)
			if err := os.Remove(f.StorageKey); err != nil && !os.IsNotExist(err) {
				log.Printf("! 阅后即焚错误: 删除物理文件失败: %v", err)
			}
			if err := db.Delete(&File{}, "id = ?", f.ID).Error; err != nil {
				log.Printf("! 阅后即焚错误: 删除数据库记录失败: %v", err)
			}
		}(h.DB, file)
	}
}

func (h *FileHandler) HandleGetPublicFiles(c *gin.Context) {
	var files []File
	result := h.DB.Select("access_code", "filename", "size_bytes", "expires_at", "is_encrypted").
		Where("expires_at > ? AND is_encrypted = false AND download_once = false", time.Now()).
		Order("created_at desc").Limit(20).Find(&files)
	if result.Error != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法提交举报，请稍后再试"})
		return
	}
	log.Printf("🚩 收到举报! IP: %s, AccessCode: %s, Reason: %s", c.ClientIP(), report.AccessCode, report.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "您的举报已收到，感谢您的帮助！我们将会尽快处理。"})
}

// HandlePreviewFile 保持不变，用于图片/视频等可以直接链接的类型
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

	fileBytes, err := os.ReadFile(file.StorageKey)
	if err != nil {
		log.Printf("! 预览错误: 无法读取文件 %s: %v", file.StorageKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}

	contentType := http.DetectContentType(fileBytes)

	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename*=UTF-8''%s`, url.PathEscape(file.Filename)))
	c.Header("X-Content-Type-Options", "nosniff")

	c.Data(http.StatusOK, contentType, fileBytes)
}

// ✨✨✨ 核心修改：添加新的 Handler 用于服务器端生成 Data URI ✨✨✨
func (h *FileHandler) HandlePreviewDataURI(c *gin.Context) {
	code := c.Param("code")
	var file File

	// 1. 查找文件并检查权限（与 HandlePreviewFile 相同）
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

	// 2. 读取文件内容
	fileBytes, err := os.ReadFile(file.StorageKey)
	if err != nil {
		log.Printf("! Data URI 预览错误: 无法读取文件 %s: %v", file.StorageKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}

	// 3. 将文件内容编码为 Base64
	base64Data := base64.StdEncoding.EncodeToString(fileBytes)

	// 4. 确定 Content-Type
	contentType := http.DetectContentType(fileBytes)

	// 5. 构造完整的 Data URI 字符串
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)

	// 6. 以 JSON 格式返回 Data URI
	c.JSON(http.StatusOK, gin.H{
		"dataUri": dataURI,
	})
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
		h.DB.Model(&File{}).Where("access_code = ? AND expires_at > ?", code, time.Now()).Count(&count)
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("无法在20次尝试内生成唯一的便捷码")
}
