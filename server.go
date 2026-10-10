package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/YanxinTang/clipboard-online/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-toast/toast"
	"github.com/lxn/walk"
	"github.com/sirupsen/logrus"
	"golang.org/x/image/bmp"
)

const (
	apiVersion = "1"
)

func setupRoute(engin *gin.Engine) {
	engin.Use(clientName(), logger(), gin.Recovery())

	// 白名单路由：不要求 API 版本校验与 authkey 认证，任何设备可直接调用
	engin.POST("/message", messageHandler)

	// 原有剪切板接口保留版本校验与认证
	engin.GET("/", apiVersionChecker(), auth(), getHandler)
	engin.POST("/", apiVersionChecker(), auth(), setHandler)
	engin.NoRoute(notFoundHandler)
}

func clientName() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlEncodedClientName := c.GetHeader("X-Client-Name")
		clientName, err := url.PathUnescape(urlEncodedClientName)
		if err != nil || clientName == "" {
			clientName = "匿名设备"
		}
		c.Set("clientName", clientName)
		c.Next()
	}
}

func apiVersionChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.GetHeader("X-API-Version")
		if version == apiVersion {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "接口版本不匹配，请升级您的捷径",
		})
	}
}

func auth() gin.HandlerFunc {
	return func(c *gin.Context) {

		if app.config.Authkey == "" {
			c.Next()
			return
		}

		reqAuth := c.GetHeader("X-Auth")

		timestamp := time.Now().Unix()
		timeKey := timestamp / app.config.AuthkeyExpiredTimeout

		authCodeRaw := app.config.Authkey + "." + strconv.FormatInt(timeKey, 10)
		authCodeHash := md5.Sum([]byte(authCodeRaw))
		authCodeString := hex.EncodeToString(authCodeHash[:])

		if authCodeString == reqAuth {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "操作被拒绝：Authkey 验证失败",
		})
	}
}

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		clientIP := c.ClientIP()
		statusCode := c.Writer.Status()
		clientName := c.GetString("clientName")
		requestLogger := log.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"statusCode": statusCode,
			"clientIP":   clientIP,
			"path":       path,
			"duration":   duration,
			"clientName": clientName,
		})

		if statusCode >= http.StatusInternalServerError {
			requestLogger.Error()
		} else if statusCode >= http.StatusBadRequest {
			requestLogger.Warn()
		} else {
			requestLogger.Info()
		}
	}
}

type ResponseFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ResponseFiles []ResponseFile

func getHandler(c *gin.Context) {
	contentType, err := utils.Clipboard().ContentType()
	if err != nil {
		log.WithError(err).Info("failed to get content type of clipboard")
		c.Status(http.StatusBadRequest)
		return
	}

	if contentType == utils.TypeText {
		str, err := walk.Clipboard().Text()
		if err != nil {
			c.Status(http.StatusBadRequest)
			log.WithError(err).Warn("failed to get clipboard")
			return
		}
		log.Info("get clipboard text")
		c.JSON(http.StatusOK, gin.H{
			"type": "text",
			"data": str,
		})
		defer sendCopyNotification(log, c.GetString("clientName"), str)
		return
	}

	if contentType == utils.TypeBitmap {
		bmpBytes, err := utils.Clipboard().Bitmap()
		if err != nil {
			log.WithError(err).Warn("failed to get bmp bytes from clipboard")
		}

		bmpBytesReader := bytes.NewReader(bmpBytes)
		bmpImage, err := bmp.Decode(bmpBytesReader)
		if err != nil {
			log.WithError(err).Warn("failed to decode bmp")
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取剪切板内容"})
			return
		}
		pngBytesBuffer := new(bytes.Buffer)
		if err = png.Encode(pngBytesBuffer, bmpImage); err != nil {
			log.WithError(err).Warn("failed to encode bmp as png")
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取剪切板内容"})
			return
		}

		responseFiles := make([]ResponseFile, 0, 1)
		responseFiles = append(responseFiles, ResponseFile{
			"clipboard.png",
			base64.StdEncoding.EncodeToString(pngBytesBuffer.Bytes()),
		})

		c.JSON(http.StatusOK, gin.H{
			"type": "file",
			"data": responseFiles,
		})
		defer sendCopyNotification(log, c.GetString("clientName"), "[图片媒体] 被复制")
		return
	}

	if contentType == utils.TypeFile {
		// get path of files from clipboard
		filenames, err := utils.Clipboard().Files()
		if err != nil {
			log.WithError(err).Warn("failed to get path of files from clipboard")
			c.Status(http.StatusBadRequest)
			return
		}

		responseFiles := make([]ResponseFile, 0, len(filenames))
		for _, path := range filenames {
			base64, err := readBase64FromFile(path)
			if err != nil {
				log.WithError(err).WithField("filepath", path).Warning("read base64 from file failed")
				continue
			}
			responseFiles = append(responseFiles, ResponseFile{filepath.Base(path), base64})
		}
		log.Info("get clipboard files")

		c.JSON(http.StatusOK, gin.H{
			"type": "file",
			"data": responseFiles,
		})
		defer sendCopyNotification(log, c.GetString("clientName"), "[文件] 被复制")
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "无法识别剪切板内容"})
}

func readBase64FromFile(path string) (string, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(fileBytes), nil
}

// Set clipboard handler

// TextBody is a struct of request body when iOS send files to windows
type TextBody struct {
	Text string `json:"data"`
}

func setHandler(c *gin.Context) {
	if !app.config.ReserveHistory {
		cleanTempFiles()
	}

	contentType := c.GetHeader("X-Content-Type")
	if contentType == utils.TypeText {
		setTextHandler(c)
		return
	}

	setFileHandler(c)
}

func setTextHandler(c *gin.Context) {
	var body TextBody
	if err := c.ShouldBindJSON(&body); err != nil {
		log.WithError(err).Warn("failed to bind text body")
		c.Status(http.StatusBadRequest)
		return
	}

	if err := utils.Clipboard().SetText(body.Text); err != nil {
		log.WithError(err).Warn("failed to set clipboard")
		c.Status(http.StatusBadRequest)
		return
	}

	var notify string = "粘贴内容为空"
	if body.Text != "" {
		notify = body.Text
	}
	defer sendPasteNotification(log, c.GetString("clientName"), notify)
	log.WithField("text", body.Text).Info("set clipboard text")
	c.Status(http.StatusOK)
}

// FileBody is a struct of request body when iOS send files to windows
type FileBody struct {
	Files []File `json:"data"`
}

// File is a struct represtents request file
type File struct {
	Name   string `json:"name"` // filename
	Base64 string `json:"base64"`
	_bytes []byte `json:"-"` // don't use this directly. use *File.Bytes() to get bytes
}

// Bytes returns byte slice of file
func (f *File) Bytes() ([]byte, error) {
	if len(f._bytes) > 0 {
		return f._bytes, nil
	}
	fileBytes, err := base64.StdEncoding.DecodeString(f.Base64)
	if err != nil {
		return []byte{}, nil
	}
	f._bytes = fileBytes
	return fileBytes, nil
}

func setFileHandler(c *gin.Context) {
	contentType := c.GetHeader("X-Content-Type")

	var body FileBody
	if err := c.ShouldBindJSON(&body); err != nil {
		log.WithError(err).Warn("failed to bind file body")
		c.Status(http.StatusBadRequest)
		return
	}

	paths := make([]string, 0, len(body.Files))
	for _, file := range body.Files {
		if file.Name == "-" && file.Base64 == "-" {
			continue
		}
		path := utils.LatestFilename(app.GetTempFilePath(file.Name))
		fileBytes, err := file.Bytes()
		if err != nil {
			log.WithField("filename", file.Name).Warn("failed to read file bytes")
			continue
		}
		if err := newFile(path, fileBytes); err != nil {
			log.WithError(err).WithField("path", path).Warn("failed to create file")
			continue
		}
		paths = append(paths, path)
	}

	if app.config.ReserveHistory {
		// clean paths in _filename.txt
		setLastFilenames(nil)
	} else {
		// write paths to file
		setLastFilenames(paths)
	}

	if err := utils.Clipboard().SetFiles(paths); err != nil {
		log.WithError(err).Warn("failed to set clipboard")
		c.Status(http.StatusBadRequest)
		return
	}

	var notify string
	if contentType == utils.TypeMedia {
		notify = "[图片媒体] 已复制到剪贴板"
	} else {
		notify = "[文件] 已复制到剪贴板"
	}

	defer sendPasteNotification(log, c.GetString("clientName"), notify)
	log.WithField("paths", paths).Info("set clipboard file")
	c.Status(http.StatusOK)
}

// MessageBody 是 /message 接口的请求体
type MessageBody struct {
	Text string `json:"text"`
}

// verificationCodeKeywords 命中任一关键词即视为文本疑似与验证码相关
var verificationCodeKeywords = []string{
	"验证码", "动态码", "校验码", "安全码", "验证代码", "短信验证码",
	"verification code", "verify code", "captcha",
}

// 验证码常见形态：4~8 位纯数字，或 4~8 位字母数字组合
var (
	verificationCodeDigitsRe = regexp.MustCompile(`\d{4,8}`)
	verificationCodeAlnumRe  = regexp.MustCompile(`[A-Za-z0-9]{4,8}`)
)

// extractVerificationCode 判断文本是否疑似验证码相关，若是则返回提取出的验证码
func extractVerificationCode(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}

	// 情形一：文本本身就是 4~8 位纯数字串（部分推送直接把验证码作为消息内容）
	if len(trimmed) >= 4 && len(trimmed) <= 8 && verificationCodeDigitsRe.FindString(trimmed) == trimmed {
		return trimmed, true
	}

	// 情形二：文本包含验证码关键词，从关键词附近提取验证码
	lowerText := strings.ToLower(trimmed)
	for _, kw := range verificationCodeKeywords {
		lowerKw := strings.ToLower(kw)
		idx := strings.Index(lowerText, lowerKw)
		if idx < 0 {
			continue
		}

		// 优先在关键词之后的一段内容中提取（如"您的验证码是 123456，请勿泄露"）
		after := trimmed[idx+len(kw):]
		if len(after) > 30 {
			after = after[:30]
		}
		if m := verificationCodeDigitsRe.FindString(after); m != "" {
			return m, true
		}
		if m := verificationCodeAlnumRe.FindString(after); m != "" {
			return m, true
		}

		// 关键词后无匹配时回退到全文提取
		if m := verificationCodeDigitsRe.FindString(trimmed); m != "" {
			return m, true
		}
	}
	return "", false
}

// messageHandler 处理白名单路由 /message：
// 文本疑似验证码相关时，将提取出的验证码复制到剪切板并通知；
// 其他内容仅通知，不复制到剪切板。
func messageHandler(c *gin.Context) {
	var body MessageBody
	if err := c.ShouldBindJSON(&body); err != nil {
		log.WithError(err).Warn("failed to bind message body")
		c.Status(http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(body.Text)
	if text == "" {
		c.Status(http.StatusOK)
		return
	}

	code, isVerificationCode := extractVerificationCode(text)
	if isVerificationCode {
		if err := utils.Clipboard().SetText(code); err != nil {
			log.WithError(err).Warn("failed to set clipboard")
			c.Status(http.StatusBadRequest)
			return
		}
		log.WithField("code", code).Info("verification code copied to clipboard")
		// 验证码：默认自动复制，通知正常显示消息原文，不发送"复制成功"提示
		defer sendMessageNotification(log, text)
		c.Status(http.StatusOK)
		return
	}

	// 非验证码内容：正常通知，不复制
	defer sendMessageNotification(log, text)
	log.WithField("text", text).Info("message notified without copying")
	c.Status(http.StatusOK)
}

// sendMessageNotification 显示 /message 的通知，不显示"消息自 X"标题
func sendMessageNotification(logger *logrus.Logger, notify string) {
	if notify == "" {
		notify = "消息内容为空"
	}
	showNotify(logger, "", notify)
}

func notFoundHandler(c *gin.Context) {
	requestLogger := log.WithFields(logrus.Fields{"user_ip": c.Request.RemoteAddr})
	requestLogger.Info("404 not found")
	c.Status(http.StatusNotFound)
}

func sendCopyNotification(logger *logrus.Logger, client, notify string) {
	if app.config.Notify.Copy {
		sendNotification(logger, "复制", client, notify)
	}
}

func sendPasteNotification(logger *logrus.Logger, client, notify string) {
	if app.config.Notify.Paste {
		sendNotification(logger, "粘贴", client, notify)
	}
}

func sendNotification(logger *logrus.Logger, action, client, notify string) {
	if notify == "" {
		notify = action + "内容为空"
	}
	title := fmt.Sprintf("%s自 %s", action, client)
	showNotify(logger, title, notify)
}

// showNotify 统一发送通知：优先使用 Windows Toast（进入通知中心且持久），
// Toast 失败时回退到传统托盘气泡。
func showNotify(logger *logrus.Logger, title, notify string) {
	if err := pushToast(title, notify); err != nil {
		logger.WithError(err).WithField("notify", notify).Warn("failed to send toast notification, fallback to balloon")
		sendBalloonNotify(logger, title, notify)
	}
}

// pushToast 通过 Windows Toast API 发送通知，支持自定义左侧图标（config.notify.icon，建议 PNG），
// 并在通知下方显示"标记已读"按钮（protocol 类型、空参数：点击无副作用，激活后通知自动从通知中心消失）
func pushToast(title, message string) error {
	n := toast.Notification{
		AppID:   appUserModelID,
		Title:   title,
		Message: message,
		Icon:    notifyIconPath(),
		Actions: []toast.Action{
			{Type: "protocol", Label: "标记已读", Arguments: ""},
		},
	}
	return n.Push()
}

// notifyIconPath 返回 config.notify.icon 的绝对路径（正斜杠，toast XML 不识别反斜杠转义）；
// 未配置时返回空字符串（使用默认图标）
func notifyIconPath() string {
	iconPath := config.Notify.Icon
	if iconPath == "" {
		return ""
	}
	if !filepath.IsAbs(iconPath) {
		iconPath = filepath.Join(execPath, iconPath)
	}
	return filepath.ToSlash(iconPath)
}

// sendBalloonNotify 传统托盘气泡通知（Toast 的降级方案）
func sendBalloonNotify(logger *logrus.Logger, title, notify string) {
	if notifyIconImage != nil {
		if err := app.ni.ShowCustom(title, notify, notifyIconImage); err != nil {
			logger.WithError(err).WithField("notify", notify).Warn("failed to send notification")
		}
		return
	}
	if err := app.ni.ShowInfo(title, notify); err != nil {
		logger.WithError(err).WithField("notify", notify).Warn("failed to send notification")
	}
}

func setLastFilenames(filenames []string) {
	path := app.GetTempFilePath("_filename.txt")
	allFilenames := strings.Join(filenames, "\n")
	_ = ioutil.WriteFile(path, []byte(allFilenames), os.ModePerm)
}

func newFile(path string, bytes []byte) error {
	return ioutil.WriteFile(path, bytes, 0644)
}

func cleanTempFiles() {
	path := app.GetTempFilePath("_filename.txt")
	if utils.IsExistFile(path) {
		file, err := os.Open(path)
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("failed to open temp file")
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			delPath := scanner.Text()
			if err = os.Remove(delPath); err != nil {
				log.WithError(err).WithField("delPath", delPath).Warn("failed to delete specify path")
			}
		}
	}
}
