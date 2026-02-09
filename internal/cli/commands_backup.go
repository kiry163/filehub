package cli

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "打包数据目录",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("dir")
		output, _ := cmd.Flags().GetString("output")
		if dataDir == "" {
			dataDir = defaultDataDir()
		}
		if output == "" {
			output = defaultBackupName()
		}
		path, err := createBackup(dataDir, output)
		if err != nil {
			return err
		}
		fmt.Printf("Backup created: %s\n", path)
		return nil
	},
}

func init() {
	backupCmd.Flags().String("dir", "", "Data directory (default: ~/.filehub/data)")
	backupCmd.Flags().String("output", "", "Output archive path (default: ./filehub-backup-YYYY-MM-DD.tar.gz)")
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./.filehub/data"
	}
	return filepath.Join(home, ".filehub", "data")
}

func defaultBackupName() string {
	return fmt.Sprintf("filehub-backup-%s.tar.gz", time.Now().Format("2006-01-02"))
}

func createBackup(dataDir, output string) (string, error) {
	dataDir = filepath.Clean(dataDir)
	info, err := os.Stat(dataDir)
	if err != nil {
		return "", fmt.Errorf("data dir not found: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("data dir is not a directory")
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}

	file, err := os.Create(output)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	baseName := filepath.Base(dataDir)

	err = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if shouldSkipBackupPath(path, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		name := filepath.Join(baseName, rel)
		if rel == "." {
			name = baseName
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(name)

		if d.IsDir() {
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
			return tw.WriteHeader(header)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = target
			header.Typeflag = tar.TypeSymlink
			return tw.WriteHeader(header)
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return "", err
	}

	return output, nil
}

func shouldSkipBackupPath(path string, d fs.DirEntry) bool {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for _, segment := range segments {
		if segment == ".minio.sys" {
			return true
		}
	}
	return false
}
