package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"milestone-tracker/internal/approval"
	"milestone-tracker/internal/project"
)

var (
	approveProjectID   string
	approveMilestoneID string
	approveApprover    string
	approveRole        string
	approveComment     string
	approveReject      bool
)

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: "审批里程碑",
	Long: `审批指定项目的里程碑，支持多级审批流程。
里程碑分类与所需审批角色:
  kickoff(启动):   部门主管
  mid_stage(中期): 部门主管 + PMO
  delivery(交付):  客户代表 + PMO
  closeout(结束):  部门主管 + PMO + 财务 + 客户代表

注意: closeout类里程碑会自动检查项目是否存在未结算的延期账单，如有则阻止审批。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if approveProjectID == "" || approveMilestoneID == "" || approveApprover == "" || approveRole == "" {
			return fmt.Errorf("--project, --milestone, --approver, --role 参数均为必填")
		}

		store, err := project.NewStore(dataPath)
		if err != nil {
			return fmt.Errorf("打开数据存储失败: %w", err)
		}

		p, ok := store.Get(approveProjectID)
		if !ok {
			return fmt.Errorf("项目 %s 不存在", approveProjectID)
		}

		var msIdx = -1
		for i := range p.Milestones {
			if p.Milestones[i].ID == approveMilestoneID {
				msIdx = i
				break
			}
		}
		if msIdx == -1 {
			return fmt.Errorf("里程碑 %s 不存在于项目 %s", approveMilestoneID, approveProjectID)
		}

		allProjects := store.List()

		if approveReject {
			if err := approval.RejectApproval(&p.Milestones[msIdx], approveApprover, approveRole, approveComment); err != nil {
				return fmt.Errorf("驳回失败: %w", err)
			}
			fmt.Printf("已驳回: %s - %s (%s)\n", p.Name, p.Milestones[msIdx].Name, approveRole)
		} else {
			result, err := approval.RequestApproval(p, &p.Milestones[msIdx], approveApprover, approveRole, approveComment, allProjects)
			if err != nil {
				return fmt.Errorf("审批失败: %w", err)
			}
			if len(result.Reasons) > 0 {
				fmt.Println("审批被阻止:")
				for _, r := range result.Reasons {
					fmt.Printf("  - %s\n", r)
				}
				return nil
			}
			if result.Approved {
				fmt.Printf("✅ 审批通过: %s - %s 已完成全部审批\n", p.Name, p.Milestones[msIdx].Name)
			} else {
				fmt.Printf("⏳ 部分审批完成: %s - %s 等待其他角色审批\n", p.Name, p.Milestones[msIdx].Name)
			}
		}

		if err := store.Update(*p); err != nil {
			return fmt.Errorf("更新项目失败: %w", err)
		}
		return store.Save()
	},
}

func init() {
	approveCmd.Flags().StringVarP(&approveProjectID, "project", "i", "", "项目ID (必填)")
	approveCmd.Flags().StringVarP(&approveMilestoneID, "milestone", "m", "", "里程碑ID (必填)")
	approveCmd.Flags().StringVarP(&approveApprover, "approver", "a", "", "审批人姓名 (必填)")
	approveCmd.Flags().StringVarP(&approveRole, "role", "r", "", "审批角色: 部门主管/PMO/客户代表/财务 (必填)")
	approveCmd.Flags().StringVarP(&approveComment, "comment", "c", "", "审批意见")
	approveCmd.Flags().BoolVar(&approveReject, "reject", false, "驳回而非通过")
	rootCmd.AddCommand(approveCmd)
}
