package milestone

import (
	"fmt"
	"sort"
	"time"

	"milestone-tracker/internal/models"
)

func NewID(projectID string, number int) string {
	return fmt.Sprintf("%s-M%03d", projectID, number)
}

func CalculateDelayDays(m models.Milestone, now time.Time) int {
	if m.Status == models.StatusCompleted || m.Status == models.StatusCanceled {
		return 0
	}
	diff := now.Sub(m.PlannedDate)
	if diff > 0 {
		return int(diff.Hours() / 24)
	}
	return 0
}

func DetermineAlertLevel(m models.Milestone, now time.Time) models.AlertLevel {
	delayDays := CalculateDelayDays(m, now)
	if delayDays >= 14 {
		return models.AlertLevelPurple
	}
	if delayDays >= 7 {
		return models.AlertLevelRed
	}
	if delayDays >= 3 {
		return models.AlertLevelYellow
	}
	return models.AlertLevelNone
}

func AutoUpdateStatus(m *models.Milestone, now time.Time) {
	if m.Status == models.StatusCompleted || m.Status == models.StatusCanceled {
		return
	}
	delayDays := CalculateDelayDays(*m, now)
	if delayDays > 0 && m.Status != models.StatusDelayed {
		m.Status = models.StatusDelayed
	}
	if m.ActualDate != nil && !m.ActualDate.IsZero() {
		m.Status = models.StatusCompleted
	}
}

func GetRequiredApprovers(category models.MilestoneCategory) []string {
	switch category {
	case models.CategoryKickoff:
		return []string{"部门主管"}
	case models.CategoryMidStage:
		return []string{"部门主管", "PMO"}
	case models.CategoryDelivery:
		return []string{"客户代表", "PMO"}
	case models.CategoryCloseout:
		return []string{"部门主管", "PMO", "财务", "客户代表"}
	default:
		return []string{"部门主管"}
	}
}

func IsFullyApproved(m models.Milestone) bool {
	required := GetRequiredApprovers(m.Category)
	approved := make(map[string]bool)
	for _, a := range m.Approvals {
		if a.Approved {
			approved[a.Role] = true
		}
	}
	for _, role := range required {
		if !approved[role] {
			return false
		}
	}
	return true
}

func SortedByNumber(milestones []models.Milestone) []models.Milestone {
	result := make([]models.Milestone, len(milestones))
	copy(result, milestones)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Number < result[j].Number
	})
	return result
}

func FilterByCategory(milestones []models.Milestone, category models.MilestoneCategory) []models.Milestone {
	result := []models.Milestone{}
	for _, m := range milestones {
		if m.Category == category {
			result = append(result, m)
		}
	}
	return result
}

func FilterByStatus(milestones []models.Milestone, status models.MilestoneStatus) []models.Milestone {
	result := []models.Milestone{}
	for _, m := range milestones {
		if m.Status == status {
			result = append(result, m)
		}
	}
	return result
}

func CountByStatus(milestones []models.Milestone) map[models.MilestoneStatus]int {
	result := make(map[models.MilestoneStatus]int)
	for _, m := range milestones {
		result[m.Status]++
	}
	return result
}

func AddDelayRecord(m *models.Milestone, reason models.DelayReason, description string, days int, reportedAt time.Time) {
	m.DelayRecords = append(m.DelayRecords, models.DelayRecord{
		Reason:      reason,
		Description: description,
		Days:        days,
		ReportedAt:  reportedAt,
		Settled:     false,
	})
	if m.Status != models.StatusCompleted && m.Status != models.StatusCanceled {
		m.Status = models.StatusDelayed
	}
}
