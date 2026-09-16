package services

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/upload"

	"gorm.io/gorm"
)

type BackupService struct {
	db *gorm.DB
}

func NewBackupService(db *gorm.DB) *BackupService {
	return &BackupService{db: db}
}

func (s *BackupService) RunBackup(ctx context.Context) (string, error) {
	_ = ctx
	cfg := config.LoadDB()
	if cfg.Driver != "sqlite" {
		return "", fmt.Errorf("sauvegarde automatique disponible uniquement pour SQLite")
	}

	ts := time.Now().Format("20060102_150405")
	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return "", fmt.Errorf("impossible de créer le dossier backups: %w", err)
	}

	destZip := filepath.Join(backupDir, fmt.Sprintf("backup_%s.zip", ts))
	if err := s.createBackupZip(cfg.Path, destZip); err != nil {
		return "", err
	}
	return destZip, nil
}

func (s *BackupService) createBackupZip(dbPath, destZip string) error {
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	if err := addFileToZip(zw, dbPath, "database/"+filepath.Base(dbPath)); err != nil {
		return fmt.Errorf("copie base de données: %w", err)
	}
	if err := addDirToZip(zw, upload.Dir, "uploads"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("archivage uploads: %w", err)
	}
	return nil
}

func addFileToZip(zw *zip.Writer, srcPath, entryName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = entryName
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

func addDirToZip(zw *zip.Writer, srcDir, prefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		entry := filepath.Join(prefix, rel)
		return addFileToZip(zw, path, entry)
	})
}
