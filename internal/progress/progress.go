package progress

import (
	"time"

	"milestone-tracker/internal/milestone"
	"milestone-tracker/internal/models"
)

func CalculateCompletionRate(p models.Project, now time.Time) float64 {
	expectedCount := 0
	completedCount := 0
	for _, m := range p.Milestones {
		if m.Status == models.StatusCanceled {
			continue
		}
		if !m.PlannedDate.After(now) {
			expectedCount++
		}
		if m.Status == models.StatusCompleted {
			completedCount++
		}
	}
	if expectedCount == 0 {
		return 100.0
	}
	return float64(completedCount) / float64(expectedCount) * 100
}

func CountDelayedMilestones(p models.Project, now time.Time) int {
	count := 0
	for _, m := range p.Milestones {
		if m.Status == models.StatusDelayed {
			count++
			continue
		}
		if m.Status != models.StatusCompleted && m.Status != models.StatusCanceled {
			if milestone.CalculateDelayDays(m, now) > 0 {
				count++
			}
		}
	}
	return count
}

func CountCompletedMilestones(p models.Project) int {
	count := 0
	for _, m := range p.Milestones {
		if m.Status == models.StatusCompleted {
			count++
		}
	}
	return count
}

func CountExpectedMilestones(p models.Project, now time.Time) int {
	count := 0
	for _, m := range p.Milestones {
		if m.Status == models.StatusCanceled {
			continue
		}
		if !m.PlannedDate.After(now) {
			count++
		}
	}
	return count
}

func TotalDelayDays(p models.Project, now time.Time) int {
	total := 0
	for _, m := range p.Milestones {
		if m.Status != models.StatusCompleted && m.Status != models.StatusCanceled {
			total += milestone.CalculateDelayDays(m, now)
		}
	}
	return total
}

func DetermineProjectAlertLevel(p models.Project, now time.Time) models.AlertLevel {
	maxLevel := models.AlertLevelNone
	delayedCount := 0
	for _, m := range p.Milestones {
		level := milestone.DetermineAlertLevel(m, now)
		switch level {
		case models.AlertLevelPurple:
			if maxLevel != models.AlertLevelPurple {
				maxLevel = models.AlertLevelPurple
			}
		case models.AlertLevelRed:
			if maxLevel == models.AlertLevelYellow || maxLevel == models.AlertLevelNone {
				maxLevel = models.AlertLevelRed
			}
		case models.AlertLevelYellow:
			if maxLevel == models.AlertLevelNone {
				maxLevel = models.AlertLevelYellow
			}
		}
		if m.Status == models.StatusDelayed || milestone.CalculateDelayDays(m, now) > 0 {
			delayedCount++
		}
	}
	if delayedCount >= 3 && maxLevel != models.AlertLevelPurple {
		maxLevel = models.AlertLevelRed
	}
	return maxLevel
}

func CalculateHealthScore(p models.Project, now time.Time) models.HealthScore {
	delayScore := 100.0
	totalDelay := TotalDelayDays(p, now)
	delayedCount := CountDelayedMilestones(p, now)
	delayDeduction := float64(totalDelay)*2.0 + float64(delayedCount)*5.0
	delayScore -= delayDeduction
	if delayScore < 0 {
		delayScore = 0
	}

	progressScore := CalculateCompletionRate(p, now)

	acceptanceScore := 100.0
	totalApprovalsNeeded := 0
	approvedCount := 0
	for _, m := range p.Milestones {
		if m.Status == models.StatusCanceled {
			continue
		}
		required := milestone.GetRequiredApprovers(m.Category)
		totalApprovalsNeeded += len(required)
		for _, role := range required {
			for _, a := range m.Approvals {
				if a.Role == role && a.Approved {
					approvedCount++
					break
				}
			}
		}
	}
	if totalApprovalsNeeded > 0 {
		acceptanceScore = float64(approvedCount) / float64(totalApprovalsNeeded) * 100
	}

	overall := delayScore*0.4 + progressScore*0.4 + acceptanceScore*0.2

	return models.HealthScore{
		Overall:         overall,
		DelayScore:      delayScore,
		ProgressScore:   progressScore,
		AcceptanceScore: acceptanceScore,
	}
}

func ComputeProjectProgress(p models.Project, now time.Time) models.ProjectProgress {
	expectedCount := CountExpectedMilestones(p, now)
	completedCount := CountCompletedMilestones(p)
	rate := 0.0
	if expectedCount > 0 {
		rate = float64(completedCount) / float64(expectedCount) * 100
	}
	return models.ProjectProgress{
		ProjectID:      p.ID,
		CompletionRate: rate,
		ExpectedCount:  expectedCount,
		CompletedCount: completedCount,
		DelayedCount:   CountDelayedMilestones(p, now),
		AlertLevel:     DetermineProjectAlertLevel(p, now),
		Health:         CalculateHealthScore(p, now),
		TotalDelayDays: TotalDelayDays(p, now),
	}
}
