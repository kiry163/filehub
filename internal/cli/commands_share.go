package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share <filehub://key>",
	Short: "获取分享链接",
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
		url, err := client.ShareFile(fileID)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	},
}
