package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint, _ := cmd.Flags().GetString("endpoint")
		localKey, _ := cmd.Flags().GetString("local-key")
		path, err := InitConfig(endpoint, localKey)
		if err != nil {
			return err
		}
		fmt.Printf("配置已写入: %s\n", path)
		return nil
	},
}

func init() {
	configInitCmd.Flags().String("endpoint", "", "API endpoint")
	configInitCmd.Flags().String("local-key", "", "Local key")
	configCmd.AddCommand(configInitCmd)
}
