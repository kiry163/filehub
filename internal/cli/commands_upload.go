package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <files...>",
	Short: "上传文件或目录",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("please provide file paths")
		}
		recursive, _ := cmd.Flags().GetBool("recursive")
		folderIDStr, _ := cmd.Flags().GetString("folder")
		var folderIDPtr *string
		if folderIDStr != "" {
			folderIDPtr = &folderIDStr
		}
		files, err := collectFiles(args, recursive)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return errors.New("no files found")
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		for _, file := range files {
			fmt.Printf("Uploading %s...\n", file)
			item, err := client.UploadFile(file, folderIDPtr, nil)
			if err != nil {
				fmt.Printf("Upload failed: %s\n", err)
				continue
			}
			fmt.Printf("filehub://%s\n", item.FileID)
			fmt.Println(item.DownloadURL)
		}
		return nil
	},
}

func init() {
	uploadCmd.Flags().Bool("recursive", false, "Upload directories recursively")
	uploadCmd.Flags().StringP("folder", "f", "", "目标文件夹ID")
}

func collectFiles(args []string, recursive bool) ([]string, error) {
	var files []string
	for _, arg := range args {
		if strings.ContainsAny(arg, "*?[") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, matches...)
			continue
		}
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			if !recursive {
				return nil, fmt.Errorf("%s is a directory (use --recursive)", arg)
			}
			err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				files = append(files, path)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		files = append(files, arg)
	}
	return files, nil
}
