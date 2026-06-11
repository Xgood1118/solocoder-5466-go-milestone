package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"milestone-tracker/internal/models"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	data     *models.DataStore
}

func NewStore(dataPath string) (*Store, error) {
	if dataPath == "" {
		dataPath = "data"
	}
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	filePath := filepath.Join(dataPath, "projects.json")
	s := &Store{
		filePath: filePath,
		data: &models.DataStore{
			Projects:   []models.Project{},
			LastUpdate: time.Now(),
		},
	}
	if _, err := os.Stat(filePath); err == nil {
		if err := s.load(); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat data file: %w", err)
	} else {
		if err := s.Save(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("read data file: %w", err)
	}
	if err := json.Unmarshal(data, &s.data); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}
	if s.data == nil {
		s.data = &models.DataStore{Projects: []models.Project{}, LastUpdate: time.Now()}
	}
	if s.data.Projects == nil {
		s.data.Projects = []models.Project{}
	}
	return nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastUpdate = time.Now()
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func (s *Store) List() []models.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Project, len(s.data.Projects))
	copy(result, s.data.Projects)
	return result
}

func (s *Store) Get(id string) (*models.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Projects {
		if s.data.Projects[i].ID == id {
			p := s.data.Projects[i]
			return &p, true
		}
	}
	return nil, false
}

func (s *Store) Add(p models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Projects {
		if existing.ID == p.ID {
			return fmt.Errorf("project %s already exists", p.ID)
		}
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	s.data.Projects = append(s.data.Projects, p)
	s.data.LastUpdate = now
	return nil
}

func (s *Store) Update(p models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Projects {
		if s.data.Projects[i].ID == p.ID {
			p.UpdatedAt = time.Now()
			p.CreatedAt = s.data.Projects[i].CreatedAt
			s.data.Projects[i] = p
			s.data.LastUpdate = time.Now()
			return nil
		}
	}
	return fmt.Errorf("project %s not found", p.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Projects {
		if s.data.Projects[i].ID == id {
			s.data.Projects = append(s.data.Projects[:i], s.data.Projects[i+1:]...)
			s.data.LastUpdate = time.Now()
			return nil
		}
	}
	return fmt.Errorf("project %s not found", id)
}

func (s *Store) UpdateMilestone(projectID string, m models.Milestone) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for pi := range s.data.Projects {
		if s.data.Projects[pi].ID == projectID {
			found := false
			for mi := range s.data.Projects[pi].Milestones {
				if s.data.Projects[pi].Milestones[mi].ID == m.ID {
					s.data.Projects[pi].Milestones[mi] = m
					found = true
					break
				}
			}
			if !found {
				s.data.Projects[pi].Milestones = append(s.data.Projects[pi].Milestones, m)
			}
			s.data.Projects[pi].UpdatedAt = time.Now()
			s.data.LastUpdate = time.Now()
			return nil
		}
	}
	return fmt.Errorf("project %s not found", projectID)
}

func (s *Store) SetLastSync(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastSync = &t
	s.data.LastUpdate = time.Now()
}

func (s *Store) GetLastSync() *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.LastSync
}

func (s *Store) FilePath() string {
	return s.filePath
}
