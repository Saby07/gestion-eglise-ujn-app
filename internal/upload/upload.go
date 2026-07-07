package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const Dir = "./uploads"

func SaveFormFile(c *gin.Context, field, prefix string) (string, error) {
	file, err := c.FormFile(field)
	if err != nil {
		return "", nil // optionnel
	}
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return "", err
	}
	ext := filepath.Ext(file.Filename)
	name := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)
	dest := filepath.Join(Dir, name)
	if err := c.SaveUploadedFile(file, dest); err != nil {
		return "", err
	}
	return "/" + strings.TrimPrefix(dest, "./"), nil
}
