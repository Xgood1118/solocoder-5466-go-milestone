package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"

	"milestone-tracker/internal/delay"
	"milestone-tracker/internal/milestone"
	"milestone-tracker/internal/models"
	"milestone-tracker/internal/progress"
)

type ReportGranularity string

const (
	GranularityWeekly ReportGranularity = "weekly"
	GranularityMonthly ReportGranularity = "monthly"
)

func GenerateReport(projects []models.Project, now time.Time, granularity ReportGranularity) models.WeeklyReport {
	projectProgresses := []models.ProjectProgress{}
	for _, p := range projects {
		projectProgresses = append(projectProgresses, progress.ComputeProjectProgress(p, now))
	}

	delayReasonStats := delay.AggregateDelayReasons(projects)
	managerRanking := computeManagerRanking(projects, now)
	suggestedActions := generateSuggestedActions(projects, projectProgresses, now)

	monteCarloResults := []models.MonteCarloResult{}
	simParams := delay.DefaultSimulationParams()
	for _, p := range projects {
		if p.Status == models.ProjectStatusActive || p.Status == models.ProjectStatusOnHold {
			monteCarloResults = append(monteCarloResults,
				delay.MonteCarloSimulation(p, projects, now, 10000, simParams))
		}
	}

	return models.WeeklyReport{
		GeneratedAt:       now,
		Projects:          projectProgresses,
		DelayReasonStats:  delayReasonStats,
		ManagerRanking:    managerRanking,
		SuggestedActions:  suggestedActions,
		MonteCarloResults: monteCarloResults,
	}
}

func computeManagerRanking(projects []models.Project, now time.Time) []models.ManagerPerformance {
	managerMap := make(map[string]*models.ManagerPerformance)

	for _, p := range projects {
		pp := progress.ComputeProjectProgress(p, now)
		m, ok := managerMap[p.PM]
		if !ok {
			m = &models.ManagerPerformance{Name: p.PM}
			managerMap[p.PM] = m
		}
		m.TotalProjects++
		if pp.DelayedCount > 0 {
			m.DelayedCount++
		}
		m.AvgHealthScore += pp.Health.Overall
	}

	result := []models.ManagerPerformance{}
	for _, m := range managerMap {
		if m.TotalProjects > 0 {
			m.DelayRate = float64(m.DelayedCount) / float64(m.TotalProjects) * 100
			m.AvgHealthScore = m.AvgHealthScore / float64(m.TotalProjects)
		}
		result = append(result, *m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DelayRate < result[j].DelayRate
	})

	return result
}

func generateSuggestedActions(projects []models.Project, progresses []models.ProjectProgress, now time.Time) []models.SuggestedAction {
	actions := []models.SuggestedAction{}
	progressMap := make(map[string]models.ProjectProgress)
	for _, pp := range progresses {
		progressMap[pp.ProjectID] = pp
	}

	for _, p := range projects {
		pp, ok := progressMap[p.ID]
		if !ok {
			continue
		}
		switch pp.AlertLevel {
		case models.AlertLevelPurple:
			actions = append(actions, models.SuggestedAction{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Action:      fmt.Sprintf("项目严重延期超14天，建议增加2-3名高级工程师并启动专项攻关"),
				Priority:    "紧急",
			})
		case models.AlertLevelRed:
			if pp.DelayedCount >= 3 {
				actions = append(actions, models.SuggestedAction{
					ProjectID:   p.ID,
					ProjectName: p.Name,
					Action:      fmt.Sprintf("同时%d个里程碑延期，建议项目组合经理介入并重新排期", pp.DelayedCount),
					Priority:    "高",
				})
			} else {
				actions = append(actions, models.SuggestedAction{
					ProjectID:   p.ID,
					ProjectName: p.Name,
					Action:      fmt.Sprintf("延期超7天，建议PM与客户沟通并提交延期申请"),
					Priority:    "高",
				})
			}
		case models.AlertLevelYellow:
			actions = append(actions, models.SuggestedAction{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Action:      fmt.Sprintf("延期超3天，建议关注并检查资源配置"),
				Priority:    "中",
			})
		}

		if pp.Health.ProgressScore < 50 {
			actions = append(actions, models.SuggestedAction{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Action:      fmt.Sprintf("进度达成率仅%.1f%%，建议每日站会跟踪关键里程碑", pp.Health.ProgressScore),
				Priority:    "高",
			})
		}

		resourceDelays := 0
		techDelays := 0
		for _, m := range p.Milestones {
			for _, dr := range m.DelayRecords {
				if dr.Reason == models.DelayReasonResource {
					resourceDelays++
				}
				if dr.Reason == models.DelayReasonTechnical {
					techDelays++
				}
			}
		}
		if resourceDelays >= 2 {
			actions = append(actions, models.SuggestedAction{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Action:      fmt.Sprintf("多次出现资源不足问题，建议人力资源部协调增加人力"),
				Priority:    "高",
			})
		}
		if techDelays >= 2 {
			actions = append(actions, models.SuggestedAction{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Action:      fmt.Sprintf("多次遇到技术难点，建议安排架构师或技术专家支持"),
				Priority:    "中",
			})
		}
	}

	sort.Slice(actions, func(i, j int) bool {
		priorityOrder := map[string]int{"紧急": 0, "高": 1, "中": 2, "低": 3}
		return priorityOrder[actions[i].Priority] < priorityOrder[actions[j].Priority]
	})

	return actions
}

func statusColorFunc(s models.MilestoneStatus) func(a ...interface{}) string {
	switch s {
	case models.StatusCompleted:
		return color.New(color.FgGreen).SprintFunc()
	case models.StatusInProgress:
		return color.New(color.FgYellow).SprintFunc()
	case models.StatusDelayed:
		return color.New(color.FgRed).SprintFunc()
	case models.StatusCanceled:
		return color.New(color.FgHiBlack).SprintFunc()
	default:
		return color.New(color.FgWhite).SprintFunc()
	}
}

func alertLevelColorFunc(level models.AlertLevel) func(a ...interface{}) string {
	switch level {
	case models.AlertLevelPurple:
		return color.New(color.FgMagenta, color.Bold).SprintFunc()
	case models.AlertLevelRed:
		return color.New(color.FgRed, color.Bold).SprintFunc()
	case models.AlertLevelYellow:
		return color.New(color.FgYellow, color.Bold).SprintFunc()
	default:
		return color.New(color.FgGreen).SprintFunc()
	}
}

func alertLevelLabel(level models.AlertLevel) string {
	switch level {
	case models.AlertLevelPurple:
		return "紫灯"
	case models.AlertLevelRed:
		return "红灯"
	case models.AlertLevelYellow:
		return "黄灯"
	default:
		return "正常"
	}
}

func PrintProjectSummaryTable(projects []models.Project, progresses []models.ProjectProgress, w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"项目ID", "项目名称", "PM", "进度%", "已完成/应完成", "延期数", "预警", "健康分"})
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	progressMap := make(map[string]models.ProjectProgress)
	for _, pp := range progresses {
		progressMap[pp.ProjectID] = pp
	}

	for _, p := range projects {
		pp, ok := progressMap[p.ID]
		if !ok {
			continue
		}
		alertColor := alertLevelColorFunc(pp.AlertLevel)
		healthColor := color.New(color.FgGreen).SprintFunc()
		if pp.Health.Overall < 60 {
			healthColor = color.New(color.FgRed).SprintFunc()
		} else if pp.Health.Overall < 80 {
			healthColor = color.New(color.FgYellow).SprintFunc()
		}

		row := []string{
			p.ID,
			p.Name,
			p.PM,
			fmt.Sprintf("%.1f%%", pp.CompletionRate),
			fmt.Sprintf("%d/%d", pp.CompletedCount, pp.ExpectedCount),
			fmt.Sprintf("%d", pp.DelayedCount),
			alertColor(alertLevelLabel(pp.AlertLevel)),
			healthColor(fmt.Sprintf("%.1f", pp.Health.Overall)),
		}
		table.Append(row)
	}
	table.Render()
}

func PrintMilestoneTable(p models.Project, w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"编号", "里程碑", "分类", "计划日期", "实际日期", "负责人", "状态", "交付物", "验收人"})
	table.SetAutoWrapText(false)

	for _, m := range milestone.SortedByNumber(p.Milestones) {
		statusColor := statusColorFunc(m.Status)
		plannedStr := m.PlannedDate.Format("2006-01-02")
		actualStr := "-"
		if m.ActualDate != nil && !m.ActualDate.IsZero() {
			actualStr = m.ActualDate.Format("2006-01-02")
		}
		row := []string{
			fmt.Sprintf("M%03d", m.Number),
			m.Name,
			m.Category.String(),
			plannedStr,
			actualStr,
			m.Owner,
			statusColor(m.Status.String()),
			m.Deliverable,
			m.Acceptor,
		}
		table.Append(row)
	}
	table.Render()
}

func PrintDelayReasonPieChart(stats map[string]int, w io.Writer) {
	total := 0
	for _, v := range stats {
		total += v
	}
	if total == 0 {
		fmt.Fprintln(w, "\n延期原因统计: 暂无延期记录")
		return
	}

	reasonNames := map[string]string{
		"client":              "客户原因",
		"requirement_change":  "需求变更",
		"resource_shortage":   "资源不足",
		"technical_difficulty": "技术难点",
		"external_dependency": "外部依赖",
	}

	type reasonStat struct {
		name  string
		count int
		pct   float64
	}
	sorted := []reasonStat{}
	for k, v := range stats {
		name, ok := reasonNames[k]
		if !ok {
			name = k
		}
		sorted = append(sorted, reasonStat{name: name, count: v, pct: float64(v) / float64(total) * 100})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

	fmt.Fprintln(w, "\n=== 延期原因分类统计 ===")
	for _, rs := range sorted {
		barLen := int(rs.pct / 2)
		bar := strings.Repeat("█", barLen)
		fmt.Fprintf(w, "%-10s [%3d次] %5.1f%% %s\n", rs.name, rs.count, rs.pct, bar)
	}
}

func PrintManagerRanking(ranking []models.ManagerPerformance, w io.Writer) {
	fmt.Fprintln(w, "\n=== 负责人绩效排名（按延期率升序） ===")
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"排名", "负责人", "项目数", "延期项目数", "延期率%", "平均健康分"})
	table.SetAutoWrapText(false)

	for i, m := range ranking {
		delayColor := color.New(color.FgGreen).SprintFunc()
		if m.DelayRate > 30 {
			delayColor = color.New(color.FgRed).SprintFunc()
		} else if m.DelayRate > 10 {
			delayColor = color.New(color.FgYellow).SprintFunc()
		}
		row := []string{
			fmt.Sprintf("%d", i+1),
			m.Name,
			fmt.Sprintf("%d", m.TotalProjects),
			fmt.Sprintf("%d", m.DelayedCount),
			delayColor(fmt.Sprintf("%.1f%%", m.DelayRate)),
			fmt.Sprintf("%.1f", m.AvgHealthScore),
		}
		table.Append(row)
	}
	table.Render()
}

func PrintSuggestedActions(actions []models.SuggestedAction, w io.Writer) {
	fmt.Fprintln(w, "\n=== 建议措施 ===")
	if len(actions) == 0 {
		fmt.Fprintln(w, "所有项目进展正常，无需额外措施。")
		return
	}
	for i, a := range actions {
		priorityColor := color.New(color.FgGreen).SprintFunc()
		switch a.Priority {
		case "紧急":
			priorityColor = color.New(color.FgMagenta, color.Bold).SprintFunc()
		case "高":
			priorityColor = color.New(color.FgRed, color.Bold).SprintFunc()
		case "中":
			priorityColor = color.New(color.FgYellow, color.Bold).SprintFunc()
		}
		fmt.Fprintf(w, "%d. [%s] %s - %s\n", i+1, priorityColor(a.Priority), a.ProjectName, a.Action)
	}
}

func PrintMonteCarloSummary(results []models.MonteCarloResult, projects []models.Project, w io.Writer) {
	fmt.Fprintln(w, "\n=== 蒙特卡洛风险模拟（10000次抽样） ===")
	projectMap := make(map[string]string)
	for _, p := range projects {
		projectMap[p.ID] = p.Name
	}

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"项目", "延期超2周概率", "延期超1月概率", "平均额外天数", "P95额外天数"})
	table.SetAutoWrapText(false)

	for _, r := range results {
		name, ok := projectMap[r.ProjectID]
		if !ok {
			name = r.ProjectID
		}
		p14Color := color.New(color.FgGreen).SprintFunc()
		if r.ProbDelayOver2Weeks > 50 {
			p14Color = color.New(color.FgRed, color.Bold).SprintFunc()
		} else if r.ProbDelayOver2Weeks > 20 {
			p14Color = color.New(color.FgYellow).SprintFunc()
		}
		p30Color := color.New(color.FgGreen).SprintFunc()
		if r.ProbDelayOver1Month > 30 {
			p30Color = color.New(color.FgRed, color.Bold).SprintFunc()
		} else if r.ProbDelayOver1Month > 10 {
			p30Color = color.New(color.FgYellow).SprintFunc()
		}

		row := []string{
			name,
			p14Color(fmt.Sprintf("%.1f%%", r.ProbDelayOver2Weeks)),
			p30Color(fmt.Sprintf("%.1f%%", r.ProbDelayOver1Month)),
			fmt.Sprintf("%.1f", r.AvgExtraDays),
			fmt.Sprintf("%.1f", r.P95ExtraDays),
		}
		table.Append(row)
	}
	table.Render()

	for _, r := range results {
		if len(r.DependencyImpacts) > 0 {
			name, _ := projectMap[r.ProjectID]
			fmt.Fprintf(w, "\n项目【%s】的依赖影响：\n", name)
			for _, di := range r.DependencyImpacts {
				fmt.Fprintf(w, "  → %s 预计延期 %.1f 天\n", di.ProjectName, di.ExpectedDelay)
			}
		}
	}
}

func PrintFullReport(rep models.WeeklyReport, projects []models.Project, w io.Writer, granularity ReportGranularity) {
	title := "周报"
	if granularity == GranularityMonthly {
		title = "月报"
	}
	bold := color.New(color.Bold).SprintFunc()
	fmt.Fprintf(w, "\n%s\n", bold(fmt.Sprintf("========== 项目里程碑达成%s [%s] ==========", title, rep.GeneratedAt.Format("2006-01-02"))))

	PrintProjectSummaryTable(projects, rep.Projects, w)
	PrintDelayReasonPieChart(rep.DelayReasonStats, w)
	PrintManagerRanking(rep.ManagerRanking, w)
	PrintMonteCarloSummary(rep.MonteCarloResults, projects, w)
	PrintSuggestedActions(rep.SuggestedActions, w)
}

func PrintProjectDetail(p models.Project, pp models.ProjectProgress, w io.Writer) {
	bold := color.New(color.Bold).SprintFunc()
	fmt.Fprintf(w, "\n%s\n", bold(fmt.Sprintf("项目详情: %s (%s)", p.Name, p.ID)))
	fmt.Fprintf(w, "客户: %s | 合同金额: %.2f万 | PM: %s | 状态: %s\n",
		p.Client, p.ContractAmount, p.PM, p.Status)
	fmt.Fprintf(w, "进度: %.1f%% | 延期里程碑: %d个 | 总延期: %d天 | 预警: %s\n",
		pp.CompletionRate, pp.DelayedCount, pp.TotalDelayDays, alertLevelLabel(pp.AlertLevel))
	fmt.Fprintf(w, "健康评分: 总体%.1f (延期%.1f / 进度%.1f / 验收%.1f)\n",
		pp.Health.Overall, pp.Health.DelayScore, pp.Health.ProgressScore, pp.Health.AcceptanceScore)
	PrintMilestoneTable(p, w)
}
