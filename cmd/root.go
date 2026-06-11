package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	dataPath string
	port     string
)

var rootCmd = &cobra.Command{
	Use:   "milestone-tracker",
	Short: "项目里程碑跟踪命令行工具",
	Long:  `一个用于跟踪项目里程碑达成情况的命令行工具，支持彩色进度报告、HTTP API服务、内网数据同步等功能。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dataPath, "data", "d", "data", "数据存储目录路径")
	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "", "HTTP服务端口 (也可通过MILESTONE_PORT或PORT环境变量设置)")
}
