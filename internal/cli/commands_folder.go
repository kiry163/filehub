package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var mkdirCmd = &cobra.Command{
	Use:   "mkdir <name>",
	Short: "创建文件夹",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parentID, _ := cmd.Flags().GetString("parent")
		var parentPtr *string
		if parentID != "" {
			parentPtr = &parentID
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		folder, err := client.CreateFolder(args[0], parentPtr)
		if err != nil {
			return err
		}
		fmt.Printf("folder_id: %s\n", folder.FolderID)
		fmt.Printf("name: %s\n", folder.Name)
		fmt.Printf("parent_id: %v\n", folder.ParentID)
		return nil
	},
}

func init() {
	mkdirCmd.Flags().StringP("parent", "p", "", "父文件夹ID (默认根目录)")
}

var lsFoldersCmd = &cobra.Command{
	Use:   "ls-folders [folder_id]",
	Short: "列出文件夹",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var parentID *string
		if len(args) > 0 && args[0] != "" {
			parentID = &args[0]
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		folders, err := client.ListFolders(parentID)
		if err != nil {
			return err
		}
		if len(folders) == 0 {
			fmt.Println("No folders found")
			return nil
		}
		fmt.Printf("%-14s %-30s %-10s %s\n", "FOLDER_ID", "NAME", "ITEMS", "CREATED_AT")
		for _, f := range folders {
			fmt.Printf("%-14s %-30s %-10d %s\n", f.FolderID, f.Name, f.ItemCount, f.CreatedAt)
		}
		return nil
	},
}

func init() {
}

var lsCmd = &cobra.Command{
	Use:   "ls [folder_id]",
	Short: "列出文件夹内容（文件+子文件夹）",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		folderID := ""
		if len(args) > 0 {
			folderID = args[0]
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		contents, err := client.GetFolderContents(folderID)
		if err != nil {
			return err
		}
		fmt.Printf("Folder: %s (%s)\n", contents.Name, contents.FolderID)
		fmt.Printf("Stats: %d folders, %d files\n\n", contents.Stats.FolderCount, contents.Stats.FileCount)

		if len(contents.Folders) > 0 {
			fmt.Println("Folders:")
			fmt.Printf("%-14s %-30s %-10s %s\n", "FOLDER_ID", "NAME", "ITEMS", "CREATED_AT")
			for _, f := range contents.Folders {
				fmt.Printf("%-14s %-30s %-10d %s\n", f.FolderID, f.Name, f.ItemCount, f.CreatedAt)
			}
			fmt.Println()
		}

		if len(contents.Files) > 0 {
			fmt.Println("Files:")
			fmt.Printf("%-14s %-30s %15s %s\n", "FILE_ID", "NAME", "SIZE", "CREATED_AT")
			for _, f := range contents.Files {
				fmt.Printf("%-14s %-30s %15d %s\n", f.FileID, f.OriginalName, f.Size, f.CreatedAt)
			}
		}
		return nil
	},
}

func init() {
}

var renameFolderCmd = &cobra.Command{
	Use:   "rename-folder <folder_id> <new_name>",
	Short: "重命名文件夹",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		if err := client.RenameFolder(args[0], args[1]); err != nil {
			return err
		}
		fmt.Println("renamed")
		return nil
	},
}

func init() {
}

var mvFolderCmd = &cobra.Command{
	Use:   "mv-folder <folder_id>",
	Short: "移动文件夹",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		if to == "" {
			return errors.New("missing --to parameter")
		}
		var parentPtr *string
		if to != "" {
			parentPtr = &to
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		if err := client.MoveFolder(args[0], parentPtr); err != nil {
			return err
		}
		fmt.Println("moved")
		return nil
	},
}

func init() {
	mvFolderCmd.Flags().StringP("to", "t", "", "目标父文件夹ID")
}

var rmdirCmd = &cobra.Command{
	Use:   "rmdir <folder_id>",
	Short: "删除空文件夹",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !force {
			fmt.Printf("Delete folder %s? (y/N): ", args[0])
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		if err := client.DeleteFolder(args[0]); err != nil {
			return err
		}
		fmt.Println("deleted")
		return nil
	},
}

var force bool

func init() {
	rmdirCmd.Flags().BoolVar(&force, "force", false, "Force delete without confirmation")
}
