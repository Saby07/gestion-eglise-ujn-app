package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"eglise_ujn/internal/upload"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) Serve(c *gin.Context) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if rel == "" || strings.Contains(rel, "..") {
		c.Status(http.StatusNotFound)
		return
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		c.Status(http.StatusNotFound)
		return
	}

	absDir, err := filepath.Abs(upload.Dir)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	absFile, err := filepath.Abs(filepath.Join(upload.Dir, clean))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if absFile != absDir && !strings.HasPrefix(absFile, absDir+string(filepath.Separator)) {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "attachment")
	c.File(absFile)
}
