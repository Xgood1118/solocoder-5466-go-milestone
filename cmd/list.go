package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"milestone-tracker/internal/models"
	"milestone-tracker/internal/progress"
	"milestone-tracker/internal/project"
	"milestone-tracker/internal/report"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有项目概览",
	Long:  `以表格形式列出所有项目的进度、延期数、预警等级和健康评分。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := project.NewStore(dataPath)
		if err != nil {
			return fmt.Errorf("打开数据存储失败: %w", err)
		}

		projects := store.List()
		if len(projects) == 0 {
			fmt.Println("暂无项目数据。")
			return nil
		}

		now := time.Now()
		progresses := []models.ProjectProgress{}
		for _, p := range projects {
			progresses = append(progresses, progress.ComputeProjectProgress(p, now))
		}

		bold := color.New(color.Bold).SprintFunc()
		fmt.Printf("\n%s\n", bold(fmt.Sprintf("项目概览 (共 %d 个项目, 更新于 %s)", len(projects), now.Format("2006-01-02 15:04"))))
		report.PrintProjectSummaryTable(projects, progresses, os.Stdout)

		activeCount := 0
		delayedCount := 0
		alertCount := map[models.AlertLevel]int{}
		for _, pp := range progresses {
			if pp.DelayedCount > 0 {
				delayedCount++
			}
			alertCount[pp.AlertLevel]++
		}
		for _, p := range projects {
			if p.Status == models.ProjectStatusActive {
				activeCount++
			}
		}

		fmt.Printf("\n统计: 在进行 %d | 有延期 %d | ", activeCount, delayedCount)
		colors := map[models.AlertLevel]func(a ...interface{}) string{
			models.AlertLevelPurple: color.New(color.FgMagenta, color.Bold).SprintFunc(),
			models.AlertLevelRed:    color.New(color.FgRed, color.Bold).SprintFunc(),
			models.AlertLevelYellow: color.New(color.FgYellow, color.Bold).SprintFunc(),
			models.AlertLevelNone:   color.New(color.FgGreen).SprintFunc(),
		}
		labels := map[models.AlertLevel]string{
			models.AlertLevelPurple: "紫灯",
			models.AlertLevelRed:    "红灯",
			models.AlertLevelYellow: "黄灯",
			models.AlertLevelNone:   "正常",
		}
		for level, c := range []models.AlertLevel{models.AlertLevelPurple, models.AlertLevelRed, models.AlertLevelYellow, models.AlertLevelNone} {
			if level > 0 {
				fmt.Printf(" | ")
			}
			fmt.Printf("%s: %s", labels[c], colors[c](fmt.Sprintf("%d", alertCount[c])))
		}
		fmt.Println()

		return nil
	},
}

func init() {
	_ = tablewriter.NewWriter
	rootCmd.AddCommand(listCmd)
}
