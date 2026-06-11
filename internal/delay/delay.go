package delay

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"milestone-tracker/internal/milestone"
	"milestone-tracker/internal/models"
)

const defaultSimulations = 10000

func AggregateDelayReasons(projects []models.Project) map[string]int {
	stats := make(map[string]int)
	for _, p := range projects {
		for _, m := range p.Milestones {
			for _, dr := range m.DelayRecords {
				stats[string(dr.Reason)]++
			}
		}
	}
	return stats
}

func HasUnsettledDelayBills(p models.Project) bool {
	for _, m := range p.Milestones {
		for _, dr := range m.DelayRecords {
			if !dr.Settled {
				return true
			}
		}
	}
	return false
}

func CountUnsettledDelayBills(p models.Project) int {
	count := 0
	for _, m := range p.Milestones {
		for _, dr := range m.DelayRecords {
			if !dr.Settled {
				count++
			}
		}
	}
	return count
}

type SimulationParams struct {
	BaseDelayStdDev     float64
	ReasonMultipliers   map[models.DelayReason]float64
	DependencyFactor    float64
}

func DefaultSimulationParams() SimulationParams {
	return SimulationParams{
		BaseDelayStdDev: 3.0,
		ReasonMultipliers: map[models.DelayReason]float64{
			models.DelayReasonClient:      1.5,
			models.DelayReasonRequirement: 2.0,
			models.DelayReasonResource:    1.8,
			models.DelayReasonTechnical:   2.5,
			models.DelayReasonExternal:    3.0,
		},
		DependencyFactor: 0.7,
	}
}

func MonteCarloSimulation(
	p models.Project,
	allProjects []models.Project,
	now time.Time,
	simulations int,
	params SimulationParams,
) models.MonteCarloResult {
	if simulations <= 0 {
		simulations = defaultSimulations
	}

	rng := rand.New(rand.NewSource(now.UnixNano()))

	results := make([]float64, simulations)
	delayOver2Weeks := 0
	delayOver1Month := 0
	totalExtra := 0.0

	projectMap := make(map[string]models.Project)
	for _, ap := range allProjects {
		projectMap[ap.ID] = ap
	}

	for i := 0; i < simulations; i++ {
		extraDays := simulateProjectDelay(p, projectMap, now, rng, params)
		results[i] = extraDays
		totalExtra += extraDays
		if extraDays >= 14 {
			delayOver2Weeks++
		}
		if extraDays >= 30 {
			delayOver1Month++
		}
	}

	sort.Float64s(results)
	p95Idx := int(math.Ceil(float64(simulations)*0.95)) - 1
	if p95Idx < 0 {
		p95Idx = 0
	}
	if p95Idx >= len(results) {
		p95Idx = len(results) - 1
	}

	dependencyImpacts := calculateDependencyImpacts(p, allProjects, now, params)

	return models.MonteCarloResult{
		ProjectID:           p.ID,
		Simulations:         simulations,
		ProbDelayOver2Weeks: float64(delayOver2Weeks) / float64(simulations) * 100,
		ProbDelayOver1Month: float64(delayOver1Month) / float64(simulations) * 100,
		AvgExtraDays:        totalExtra / float64(simulations),
		P95ExtraDays:        results[p95Idx],
		DependencyImpacts:   dependencyImpacts,
	}
}

func simulateProjectDelay(
	p models.Project,
	projectMap map[string]models.Project,
	now time.Time,
	rng *rand.Rand,
	params SimulationParams,
) float64 {
	totalExtra := 0.0

	for _, depID := range p.Dependencies {
		if dep, ok := projectMap[depID]; ok {
			depDelay := simulateProjectDelay(dep, projectMap, now, rng, params)
			totalExtra += depDelay * params.DependencyFactor
		}
	}

	for _, m := range p.Milestones {
		if m.Status == models.StatusCompleted || m.Status == models.StatusCanceled {
			continue
		}
		baseDelay := float64(milestone.CalculateDelayDays(m, now))
		reasonMultiplier := 1.0
		for _, dr := range m.DelayRecords {
			if mult, ok := params.ReasonMultipliers[dr.Reason]; ok {
				reasonMultiplier = math.Max(reasonMultiplier, mult)
			}
		}
		randomFactor := math.Abs(rng.NormFloat64()) * params.BaseDelayStdDev
		extra := (baseDelay + randomFactor) * reasonMultiplier
		totalExtra += extra
	}

	return totalExtra
}

func calculateDependencyImpacts(
	p models.Project,
	allProjects []models.Project,
	now time.Time,
	params SimulationParams,
) []models.DependencyImpact {
	impacts := []models.DependencyImpact{}
	pDelay := float64(milestone.CalculateDelayDays(models.Milestone{
		Status:      models.StatusDelayed,
		PlannedDate: p.PlannedEndDate,
	}, now))

	if pDelay <= 0 {
		pDelay = 5.0
	}

	for _, ap := range allProjects {
		if ap.ID == p.ID {
			continue
		}
		for _, depID := range ap.Dependencies {
			if depID == p.ID {
				impact := models.DependencyImpact{
					ProjectID:     ap.ID,
					ProjectName:   ap.Name,
					ExpectedDelay: pDelay * params.DependencyFactor,
				}
				impacts = append(impacts, impact)
				break
			}
		}
	}
	return impacts
}
