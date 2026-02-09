package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <filehub://key>",
	Short: "下载文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("please provide filehub:// key")
		}
		fileID, err := parseFilehubURL(args[0])
		if err != nil {
			return err
		}
		output, _ := cmd.Flags().GetString("output")
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		client := NewClient(cfg)
		path, err := client.DownloadFile(fileID, output, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Saved to %s\n", path)
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringP("output", "o", "", "Output directory or file path")
}

func parseFilehubURL(value string) (string, error) {
	if strings.HasPrefix(value, "filehub://") {
		return strings.TrimPrefix(value, "filehub://"), nil
	}
	return "", errors.New("invalid filehub URL")
}
