package models

import (
	"time"
)

type MilestoneStatus string

const (
	StatusNotStarted MilestoneStatus = "not_started"
	StatusInProgress MilestoneStatus = "in_progress"
	StatusCompleted  MilestoneStatus = "completed"
	StatusDelayed    MilestoneStatus = "delayed"
	StatusCanceled   MilestoneStatus = "canceled"
)

func (s MilestoneStatus) String() string {
	switch s {
	case StatusNotStarted:
		return "未开始"
	case StatusInProgress:
		return "进行中"
	case StatusCompleted:
		return "已完成"
	case StatusDelayed:
		return "已延期"
	case StatusCanceled:
		return "已取消"
	default:
		return string(s)
	}
}

type MilestoneCategory string

const (
	CategoryKickoff   MilestoneCategory = "kickoff"
	CategoryMidStage  MilestoneCategory = "mid_stage"
	CategoryDelivery  MilestoneCategory = "delivery"
	CategoryCloseout  MilestoneCategory = "closeout"
)

func (c MilestoneCategory) String() string {
	switch c {
	case CategoryKickoff:
		return "启动"
	case CategoryMidStage:
		return "中期"
	case CategoryDelivery:
		return "交付"
	case CategoryCloseout:
		return "结束"
	default:
		return string(c)
	}
}

type DelayReason string

const (
	DelayReasonClient        DelayReason = "client"
	DelayReasonRequirement   DelayReason = "requirement_change"
	DelayReasonResource      DelayReason = "resource_shortage"
	DelayReasonTechnical     DelayReason = "technical_difficulty"
	DelayReasonExternal      DelayReason = "external_dependency"
)

func (r DelayReason) String() string {
	switch r {
	case DelayReasonClient:
		return "客户原因"
	case DelayReasonRequirement:
		return "需求变更"
	case DelayReasonResource:
		return "资源不足"
	case DelayReasonTechnical:
		return "技术难点"
	case DelayReasonExternal:
		return "外部依赖"
	default:
		return string(r)
	}
}

type AlertLevel string

const (
	AlertLevelNone   AlertLevel = "none"
	AlertLevelYellow AlertLevel = "yellow"
	AlertLevelRed    AlertLevel = "red"
	AlertLevelPurple AlertLevel = "purple"
)

type DelayRecord struct {
	Reason      DelayReason `json:"reason"`
	Description string      `json:"description"`
	Days        int         `json:"days"`
	ReportedAt  time.Time   `json:"reported_at"`
	Settled     bool        `json:"settled"`
	SettledAt   *time.Time  `json:"settled_at,omitempty"`
}

type ApprovalRecord struct {
	Approver    string    `json:"approver"`
	Role        string    `json:"role"`
	Approved    bool      `json:"approved"`
	ApprovedAt  time.Time `json:"approved_at"`
	Comment     string    `json:"comment,omitempty"`
}

type Milestone struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"project_id"`
	Number        int               `json:"number"`
	Name          string            `json:"name"`
	Category      MilestoneCategory `json:"category"`
	PlannedDate   time.Time         `json:"planned_date"`
	ActualDate    *time.Time        `json:"actual_date,omitempty"`
	Owner         string            `json:"owner"`
	Status        MilestoneStatus   `json:"status"`
	Deliverable   string            `json:"deliverable"`
	Acceptor      string            `json:"acceptor"`
	DelayRecords  []DelayRecord     `json:"delay_records,omitempty"`
	Approvals     []ApprovalRecord  `json:"approvals,omitempty"`
	Dependencies  []string          `json:"dependencies,omitempty"`
}

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusOnHold   ProjectStatus = "on_hold"
	ProjectStatusClosed   ProjectStatus = "closed"
	ProjectStatusCanceled ProjectStatus = "canceled"
)

type Project struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Client        string            `json:"client"`
	ContractAmount float64          `json:"contract_amount"`
	PM            string            `json:"pm"`
	Status        ProjectStatus     `json:"status"`
	StartDate     time.Time         `json:"start_date"`
	PlannedEndDate time.Time        `json:"planned_end_date"`
	ActualEndDate *time.Time        `json:"actual_end_date,omitempty"`
	Dependencies  []string          `json:"dependencies,omitempty"`
	Milestones    []Milestone       `json:"milestones"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type HealthScore struct {
	Overall         float64 `json:"overall"`
	DelayScore      float64 `json:"delay_score"`
	ProgressScore   float64 `json:"progress_score"`
	AcceptanceScore float64 `json:"acceptance_score"`
}

type ProjectProgress struct {
	ProjectID       string      `json:"project_id"`
	CompletionRate  float64     `json:"completion_rate"`
	ExpectedCount   int         `json:"expected_count"`
	CompletedCount  int         `json:"completed_count"`
	DelayedCount    int         `json:"delayed_count"`
	AlertLevel      AlertLevel  `json:"alert_level"`
	Health          HealthScore `json:"health"`
	TotalDelayDays  int         `json:"total_delay_days"`
}

type MonteCarloResult struct {
	ProjectID              string    `json:"project_id"`
	Simulations            int       `json:"simulations"`
	ProbDelayOver2Weeks    float64   `json:"prob_delay_over_2_weeks"`
	ProbDelayOver1Month    float64   `json:"prob_delay_over_1_month"`
	AvgExtraDays           float64   `json:"avg_extra_days"`
	P95ExtraDays           float64   `json:"p95_extra_days"`
	DependencyImpacts      []DependencyImpact `json:"dependency_impacts"`
}

type DependencyImpact struct {
	ProjectID      string  `json:"project_id"`
	ProjectName    string  `json:"project_name"`
	ExpectedDelay  float64 `json:"expected_delay_days"`
}

type ManagerPerformance struct {
	Name           string  `json:"name"`
	TotalProjects  int     `json:"total_projects"`
	DelayedCount   int     `json:"delayed_count"`
	DelayRate      float64 `json:"delay_rate"`
	AvgHealthScore float64 `json:"avg_health_score"`
}

type SuggestedAction struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	Action      string `json:"action"`
	Priority    string `json:"priority"`
}

type WeeklyReport struct {
	GeneratedAt        time.Time            `json:"generated_at"`
	Projects           []ProjectProgress    `json:"projects"`
	DelayReasonStats   map[string]int       `json:"delay_reason_stats"`
	ManagerRanking     []ManagerPerformance `json:"manager_ranking"`
	SuggestedActions   []SuggestedAction    `json:"suggested_actions"`
	MonteCarloResults  []MonteCarloResult   `json:"monte_carlo_results"`
}

type DataStore struct {
	Projects   []Project   `json:"projects"`
	LastSync   *time.Time  `json:"last_sync,omitempty"`
	LastUpdate time.Time   `json:"last_update"`
}
