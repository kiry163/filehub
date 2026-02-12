package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		order, _ := cmd.Flags().GetString("order")
		keyword, _ := cmd.Flags().GetString("keyword")
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
		files, total, err := client.ListFiles(folderIDPtr, limit, offset, order, keyword)
		if err != nil {
			return err
		}
		fmt.Printf("Total: %d\n", total)
		for _, file := range files {
			fmt.Printf("%s\t%s\t%s\n", file.FileID, file.OriginalName, file.CreatedAt)
			fmt.Printf("%s\n", file.DownloadURL)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().Int("limit", 20, "Limit")
	listCmd.Flags().Int("offset", 0, "Offset")
	listCmd.Flags().String("order", "desc", "Order (asc/desc)")
	listCmd.Flags().String("keyword", "", "Search keyword")
	listCmd.Flags().StringP("folder", "f", "", "文件夹ID")
}
