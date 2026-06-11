package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"milestone-tracker/internal/delay"
	"milestone-tracker/internal/models"
	"milestone-tracker/internal/progress"
	"milestone-tracker/internal/project"
	"milestone-tracker/internal/report"
)

type Server struct {
	store *project.Store
	mux   *http.ServeMux
	port  string
}

func GetPortFromEnv(defaultPort string) string {
	port := os.Getenv("MILESTONE_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = defaultPort
	}
	return port
}

func NewServer(store *project.Store, port string) *Server {
	if port == "" {
		port = GetPortFromEnv("9090")
	}
	s := &Server{
		store: store,
		mux:   http.NewServeMux(),
		port:  port,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", s.handleProjectByID)
	s.mux.HandleFunc("/api/progress", s.handleProgress)
	s.mux.HandleFunc("/api/report/weekly", s.handleWeeklyReport)
	s.mux.HandleFunc("/api/report/monthly", s.handleMonthlyReport)
	s.mux.HandleFunc("/api/delay/reasons", s.handleDelayReasons)
	s.mux.HandleFunc("/api/delay/montecarlo", s.handleMonteCarlo)
	s.mux.HandleFunc("/api/managers/ranking", s.handleManagerRanking)
	s.mux.HandleFunc("/api/actions/suggested", s.handleSuggestedActions)
	s.mux.HandleFunc("/api/grafana/projects", s.handleGrafanaProjects)
}

func (s *Server) Start() error {
	addr := ":" + s.port
	fmt.Printf("Milestone Tracker API server starting on port %s...\n", s.port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  GET /health\n")
	fmt.Printf("  GET /api/projects\n")
	fmt.Printf("  GET /api/projects/{id}\n")
	fmt.Printf("  GET /api/progress\n")
	fmt.Printf("  GET /api/report/weekly\n")
	fmt.Printf("  GET /api/report/monthly\n")
	fmt.Printf("  GET /api/delay/reasons\n")
	fmt.Printf("  GET /api/delay/montecarlo\n")
	fmt.Printf("  GET /api/managers/ranking\n")
	fmt.Printf("  GET /api/actions/suggested\n")
	fmt.Printf("  GET /api/grafana/projects (Grafana JSON datasource compatible)\n")
	return http.ListenAndServe(addr, s.mux)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
		"projects":  len(s.store.List()),
	})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.URL.Path[len("/api/projects/"):]
	if id == "" {
		writeError(w, http.StatusBadRequest, "project id required")
		return
	}
	p, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projects := s.store.List()
	now := time.Now()
	results := []models.ProjectProgress{}
	for _, p := range projects {
		results = append(results, progress.ComputeProjectProgress(p, now))
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleWeeklyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projects := s.store.List()
	rep := report.GenerateReport(projects, time.Now(), report.GranularityWeekly)
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projects := s.store.List()
	rep := report.GenerateReport(projects, time.Now(), report.GranularityMonthly)
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleDelayReasons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	stats := delay.AggregateDelayReasons(s.store.List())
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleMonteCarlo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	simCount := 10000
	if sc := r.URL.Query().Get("simulations"); sc != "" {
		if n, err := strconv.Atoi(sc); err == nil && n > 0 {
			simCount = n
		}
	}
	projects := s.store.List()
	now := time.Now()
	params := delay.DefaultSimulationParams()
	results := []models.MonteCarloResult{}
	for _, p := range projects {
		if p.Status == models.ProjectStatusActive || p.Status == models.ProjectStatusOnHold {
			results = append(results, delay.MonteCarloSimulation(p, projects, now, simCount, params))
		}
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleManagerRanking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projects := s.store.List()
	now := time.Now()
	rep := report.GenerateReport(projects, now, report.GranularityWeekly)
	writeJSON(w, http.StatusOK, rep.ManagerRanking)
}

func (s *Server) handleSuggestedActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projects := s.store.List()
	now := time.Now()
	rep := report.GenerateReport(projects, now, report.GranularityWeekly)
	writeJSON(w, http.StatusOK, rep.SuggestedActions)
}

type GrafanaTarget struct {
	Target string `json:"target"`
	RefID  string `json:"refId"`
}

type GrafanaQuery struct {
	Targets []GrafanaTarget `json:"targets"`
}

type GrafanaDataPoint struct {
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

type GrafanaResponse struct {
	Target     string          `json:"target"`
	DataPoints [][]interface{} `json:"datapoints"`
}

func (s *Server) handleGrafanaProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, []map[string]interface{}{
			{"text": "项目健康度评分", "value": "health_score"},
			{"text": "进度达成率", "value": "completion_rate"},
			{"text": "延期里程碑数", "value": "delayed_count"},
			{"text": "总延期天数", "value": "total_delay_days"},
		})
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var query GrafanaQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		writeError(w, http.StatusBadRequest, "invalid query")
		return
	}

	projects := s.store.List()
	now := time.Now()
	ts := now.Unix() * 1000

	response := []GrafanaResponse{}
	for _, target := range query.Targets {
		for _, p := range projects {
			pp := progress.ComputeProjectProgress(p, now)
			var value float64
			switch target.Target {
			case "health_score":
				value = pp.Health.Overall
			case "completion_rate":
				value = pp.CompletionRate
			case "delayed_count":
				value = float64(pp.DelayedCount)
			case "total_delay_days":
				value = float64(pp.TotalDelayDays)
			default:
				continue
			}
			response = append(response, GrafanaResponse{
				Target: fmt.Sprintf("%s - %s", p.Name, target.Target),
				DataPoints: [][]interface{}{
					{value, ts},
				},
			})
		}
	}

	writeJSON(w, http.StatusOK, response)
}
