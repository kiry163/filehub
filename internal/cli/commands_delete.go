package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <filehub://key>",
	Short: "删除文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("please provide filehub:// key")
		}
		fileID, err := parseFilehubURL(args[0])
		if err != nil {
			return err
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		if err := client.DeleteFile(fileID); err != nil {
			return err
		}
		fmt.Println("deleted")
		return nil
	},
}
