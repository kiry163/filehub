package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var statCmd = &cobra.Command{
	Use:   "stat <filehub://key>",
	Short: "查看文件详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileID, err := parseFilehubURL(args[0])
		if err != nil {
			return err
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		file, err := client.GetFile(fileID)
		if err != nil {
			return err
		}
		fmt.Printf("File ID:       %s\n", file.FileID)
		fmt.Printf("Original Name: %s\n", file.OriginalName)
		fmt.Printf("Size:          %d bytes\n", file.Size)
		fmt.Printf("MIME Type:     %s\n", file.MimeType)
		fmt.Printf("Created At:     %s\n", file.CreatedAt)
		fmt.Printf("FileHub URL:    filehub://%s\n", file.FileID)
		fmt.Printf("Download URL:   %s\n", file.DownloadURL)
		return nil
	},
}

func init() {
}

var statFolderCmd = &cobra.Command{
	Use:   "stat-folder <folder_id>",
	Short: "查看文件夹详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		folder, err := client.GetFolder(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Folder ID: %s\n", folder.FolderID)
		fmt.Printf("Name:      %s\n", folder.Name)
		if folder.ParentID != nil && *folder.ParentID != "" {
			fmt.Printf("Parent ID: %s\n", *folder.ParentID)
		} else {
			fmt.Printf("Parent ID: (root)\n")
		}
		fmt.Printf("Created:   %s\n", folder.CreatedAt)
		return nil
	},
}

func init() {
}

var urlFileCmd = &cobra.Command{
	Use:   "url-file <filehub://key>",
	Short: "获取文件访问页面链接（需登录）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileID, err := parseFilehubURL(args[0])
		if err != nil {
			return err
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		file, err := client.GetFile(fileID)
		if err != nil {
			return err
		}
		viewURL := strings.Replace(file.DownloadURL, "/download", "", 1)
		fmt.Println(viewURL)
		return nil
	},
}

func init() {
}

var urlFolderCmd = &cobra.Command{
	Use:   "url-folder <folder_id>",
	Short: "获取文件夹访问页面链接",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		folder, err := client.GetFolder(args[0])
		if err != nil {
			return err
		}
		viewURL := strings.TrimRight(cfg.Endpoint, "/") + "/app/folders/" + folder.FolderID
		fmt.Println(viewURL)
		return nil
	},
}

func init() {
}

var findCmd = &cobra.Command{
	Use:   "find <pattern>",
	Short: "搜索文件（支持通配符）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		folderIDStr, _ := cmd.Flags().GetString("folder")
		var folderIDPtr *string
		if folderIDStr != "" {
			folderIDPtr = &folderIDStr
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		files, total, err := client.ListFiles(folderIDPtr, 100, 0, "desc", args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Found %d files matching '%s':\n", total, args[0])
		for _, file := range files {
			fmt.Printf("%s\t%s\t%s\n", file.FileID, file.OriginalName, file.CreatedAt)
		}
		return nil
	},
}

func init() {
	findCmd.Flags().StringP("folder", "f", "", "搜索文件夹（默认根目录）")
}
