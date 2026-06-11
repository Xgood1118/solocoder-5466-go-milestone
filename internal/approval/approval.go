package approval

import (
	"fmt"
	"time"

	"milestone-tracker/internal/delay"
	"milestone-tracker/internal/milestone"
	"milestone-tracker/internal/models"
)

type ApprovalResult struct {
	Approved bool
	Reasons  []string
}

func RequestApproval(
	project *models.Project,
	ms *models.Milestone,
	approver string,
	role string,
	comment string,
	allProjects []models.Project,
) (ApprovalResult, error) {
	result := ApprovalResult{Approved: false, Reasons: []string{}}

	if ms.Category == models.CategoryCloseout {
		if delay.HasUnsettledDelayBills(*project) {
			unsettledCount := delay.CountUnsettledDelayBills(*project)
			result.Reasons = append(result.Reasons,
				fmt.Sprintf("项目存在 %d 条未结算的延期账单，财务结算完成前无法推进 closeout 审批", unsettledCount))
			return result, nil
		}
	}

	required := milestone.GetRequiredApprovers(ms.Category)
	roleValid := false
	for _, r := range required {
		if r == role {
			roleValid = true
			break
		}
	}
	if !roleValid {
		return result, fmt.Errorf("角色 %s 无权审批该 %s 类里程碑，需要: %v", role, ms.Category, required)
	}

	alreadyApproved := false
	for i := range ms.Approvals {
		if ms.Approvals[i].Role == role {
			ms.Approvals[i].Approved = true
			ms.Approvals[i].Approver = approver
			ms.Approvals[i].ApprovedAt = time.Now()
			ms.Approvals[i].Comment = comment
			alreadyApproved = true
			break
		}
	}
	if !alreadyApproved {
		ms.Approvals = append(ms.Approvals, models.ApprovalRecord{
			Approver:   approver,
			Role:       role,
			Approved:   true,
			ApprovedAt: time.Now(),
			Comment:    comment,
		})
	}

	if milestone.IsFullyApproved(*ms) {
		result.Approved = true
	}

	return result, nil
}

func RejectApproval(
	ms *models.Milestone,
	approver string,
	role string,
	reason string,
) error {
	required := milestone.GetRequiredApprovers(ms.Category)
	roleValid := false
	for _, r := range required {
		if r == role {
			roleValid = true
			break
		}
	}
	if !roleValid {
		return fmt.Errorf("角色 %s 无权审批该 %s 类里程碑", role, ms.Category)
	}

	for i := range ms.Approvals {
		if ms.Approvals[i].Role == role {
			ms.Approvals[i].Approved = false
			ms.Approvals[i].Approver = approver
			ms.Approvals[i].ApprovedAt = time.Now()
			ms.Approvals[i].Comment = reason
			return nil
		}
	}

	ms.Approvals = append(ms.Approvals, models.ApprovalRecord{
		Approver:   approver,
		Role:       role,
		Approved:   false,
		ApprovedAt: time.Now(),
		Comment:    reason,
	})
	return nil
}

func GetApprovalStatus(ms models.Milestone) map[string]bool {
	status := make(map[string]bool)
	required := milestone.GetRequiredApprovers(ms.Category)
	for _, role := range required {
		status[role] = false
		for _, a := range ms.Approvals {
			if a.Role == role && a.Approved {
				status[role] = true
				break
			}
		}
	}
	return status
}
