package upload

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
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

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	header := make([]byte, 16)
	n, _ := io.ReadFull(src, header)
	header = header[:n]
	if !matchesMagic(ext, header) {
		return "", fmt.Errorf("contenu de fichier non autorisé")
	}

	if err := os.MkdirAll(Dir, 0o750); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)
	dest := filepath.Join(Dir, name)

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := out.Write(header); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	if _, err := io.Copy(out, io.LimitReader(src, MaxFileSize+1)); err != nil {
		_ = os.Remove(dest)
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
		} else if strings.Contains(header, "webp") {
			ext = ".webp"
		}
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("signature invalide")
	}
	if len(raw) > MaxFileSize {
		return "", fmt.Errorf("fichier trop volumineux (max %d Mo)", MaxFileSize/(1<<20))
	}
	if !matchesMagic(ext, raw) {
		return "", fmt.Errorf("contenu de fichier non autorisé")
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

func matchesMagic(ext string, data []byte) bool {
	switch ext {
	case ".pdf":
		return bytes.HasPrefix(data, []byte("%PDF"))
	case ".png":
		return bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	case ".jpg", ".jpeg":
		return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
	case ".webp":
		return len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
	default:
		return false
	}
}

func ContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
