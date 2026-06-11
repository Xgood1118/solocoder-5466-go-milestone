package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"milestone-tracker/internal/api"
	"milestone-tracker/internal/project"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动HTTP API服务",
	Long:  `启动HTTP API服务，提供REST接口供Grafana或其他系统对接。端口通过-p参数或MILESTONE_PORT/PORT环境变量设置，默认9090。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := project.NewStore(dataPath)
		if err != nil {
			return fmt.Errorf("打开数据存储失败: %w", err)
		}

		serverPort := port
		if serverPort == "" {
			serverPort = api.GetPortFromEnv("9090")
		}

		server := api.NewServer(store, serverPort)
		return server.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
