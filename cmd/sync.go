package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"milestone-tracker/internal/project"
	"milestone-tracker/internal/scraper"
)

var (
	syncBaseURL       string
	syncProjectsPath  string
	syncCookie        string
	syncCookieFile    string
	syncUsername      string
	syncPassword      string
	syncLoginURL      string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "从内网项目管理页面同步数据",
	Long:  `使用goquery爬取内网项目管理页面数据，支持cookie字符串、cookie文件、用户名密码三种登录方式。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if syncBaseURL == "" {
			return fmt.Errorf("请通过 --url 指定内网项目管理系统的基础URL")
		}

		cfg := scraper.ScraperConfig{
			BaseURL:       syncBaseURL,
			SessionCookie: syncCookie,
			CookieFile:    syncCookieFile,
			Username:      syncUsername,
			Password:      syncPassword,
			LoginURL:      syncLoginURL,
		}

		s, err := scraper.NewScraper(cfg)
		if err != nil {
			return fmt.Errorf("初始化爬虫失败: %w", err)
		}

		fmt.Println("正在同步项目数据...")
		projects, err := s.FetchProjects(syncProjectsPath)
		if err != nil {
			return fmt.Errorf("获取项目列表失败: %w", err)
		}

		store, err := project.NewStore(dataPath)
		if err != nil {
			return fmt.Errorf("打开数据存储失败: %w", err)
		}

		updated := 0
		added := 0
		for _, p := range projects {
			_, exists := store.Get(p.ID)
			if exists {
				if err := store.Update(p); err == nil {
					updated++
				}
			} else {
				if err := store.Add(p); err == nil {
					added++
				}
			}
		}

		if err := store.Save(); err != nil {
			return fmt.Errorf("保存数据失败: %w", err)
		}
		store.SetLastSync(time.Now())
		_ = store.Save()

		if syncCookieFile != "" {
			if err := s.SaveCookiesToFile(syncCookieFile); err != nil {
				fmt.Printf("警告: 保存cookie失败: %v\n", err)
			}
		}

		fmt.Printf("同步完成: 新增 %d 个项目, 更新 %d 个项目, 总计 %d 个项目\n", added, updated, len(projects))
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVar(&syncBaseURL, "url", "", "内网项目管理系统基础URL (必填)")
	syncCmd.Flags().StringVar(&syncProjectsPath, "path", "/projects", "项目列表页面路径")
	syncCmd.Flags().StringVar(&syncCookie, "cookie", "", "登录Cookie字符串，格式: 'sessionid=xxx; token=yyy'")
	syncCmd.Flags().StringVar(&syncCookieFile, "cookie-file", "", "Cookie文件路径（用于持久化登录态）")
	syncCmd.Flags().StringVar(&syncUsername, "username", "", "登录用户名")
	syncCmd.Flags().StringVar(&syncPassword, "password", "", "登录密码")
	syncCmd.Flags().StringVar(&syncLoginURL, "login-url", "/login", "登录接口路径")
	rootCmd.AddCommand(syncCmd)
}
