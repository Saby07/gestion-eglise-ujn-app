package upload

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const Dir = "./uploads"

const MaxFileSize = 10 << 20 // 10 MiB

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

func SaveFormFile(c *gin.Context, field, prefix string) (string, error) {
	file, err := c.FormFile(field)
	if err != nil {
		return "", nil
	}
	if file.Size > MaxFileSize {
		return "", fmt.Errorf("fichier trop volumineux (max %d Mo)", MaxFileSize/(1<<20))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return "", fmt.Errorf("type de fichier non autorisé")
	}

	if err := os.MkdirAll(Dir, 0o750); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)
	dest := filepath.Join(Dir, name)
	if err := c.SaveUploadedFile(file, dest); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}

func SaveBase64Image(dataURL, prefix string) (string, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return "", nil
	}
	payload := dataURL
	ext := ".png"
	if idx := strings.Index(dataURL, ","); idx >= 0 {
		header := dataURL[:idx]
		payload = dataURL[idx+1:]
		if strings.Contains(header, "jpeg") || strings.Contains(header, "jpg") {
			ext = ".jpg"
		}
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("signature invalide")
	}
	if len(raw) > MaxFileSize {
		return "", fmt.Errorf("fichier trop volumineux (max %d Mo)", MaxFileSize/(1<<20))
	}
	if err := os.MkdirAll(Dir, 0o750); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)
	dest := filepath.Join(Dir, name)
	if err := os.WriteFile(dest, raw, 0o640); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}

func ContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
