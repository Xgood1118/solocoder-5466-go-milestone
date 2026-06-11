package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"milestone-tracker/internal/models"
	"milestone-tracker/internal/progress"
	"milestone-tracker/internal/project"
	"milestone-tracker/internal/report"
)

var (
	reportGranularity string
	reportProjectID   string
	reportMonthly     bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "生成里程碑达成情况报告",
	Long:  `生成周报或月报，包含项目健康度、延期统计、蒙特卡洛风险模拟、负责人绩效排名及建议措施。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := project.NewStore(dataPath)
		if err != nil {
			return fmt.Errorf("打开数据存储失败: %w", err)
		}

		projects := store.List()
		if len(projects) == 0 {
			fmt.Println("暂无项目数据。请先使用 sync 命令同步数据或手动编辑 data/projects.json")
			return nil
		}

		granularity := report.GranularityWeekly
		if reportMonthly {
			granularity = report.GranularityMonthly
		}

		if reportProjectID != "" {
			p, ok := store.Get(reportProjectID)
			if !ok {
				return fmt.Errorf("项目 %s 不存在", reportProjectID)
			}
			pp := progress.ComputeProjectProgress(*p, time.Now())
			report.PrintProjectDetail(*p, pp, os.Stdout)
			return nil
		}

		rep := report.GenerateReport(projects, time.Now(), granularity)
		report.PrintFullReport(rep, projects, os.Stdout, granularity)

		return nil
	},
}

func init() {
	reportCmd.Flags().BoolVarP(&reportMonthly, "monthly", "m", false, "生成月报（默认周报）")
	reportCmd.Flags().StringVarP(&reportProjectID, "project", "i", "", "查看指定项目ID的详细报告")
	_ = reportGranularity
	_ = models.StatusCompleted
	rootCmd.AddCommand(reportCmd)
}
