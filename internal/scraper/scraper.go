package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"milestone-tracker/internal/models"
)

type Scraper struct {
	client    *http.Client
	baseURL   string
	cookies   []*http.Cookie
	userAgent string
}

type ScraperConfig struct {
	BaseURL      string
	SessionCookie string
	CookieFile   string
	Username     string
	Password     string
	LoginURL     string
}

func NewScraper(cfg ScraperConfig) (*Scraper, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}

	client := &http.Client{
		Jar:       jar,
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	s := &Scraper{
		client:    client,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	if cfg.SessionCookie != "" {
		if err := s.setSessionCookie(cfg.SessionCookie); err != nil {
			return nil, err
		}
	}
	if cfg.CookieFile != "" {
		if err := s.loadCookiesFromFile(cfg.CookieFile); err != nil {
			return nil, err
		}
	}

	if cfg.Username != "" && cfg.Password != "" && cfg.LoginURL != "" {
		if err := s.login(cfg.LoginURL, cfg.Username, cfg.Password); err != nil {
			return nil, fmt.Errorf("login failed: %w", err)
		}
	}

	return s, nil
}

func (s *Scraper) setSessionCookie(cookieStr string) error {
	parsedURL, err := url.Parse(s.baseURL)
	if err != nil {
		return fmt.Errorf("parse base URL: %w", err)
	}
	cookies := parseCookieString(cookieStr)
	s.client.Jar.SetCookies(parsedURL, cookies)
	s.cookies = cookies
	return nil
}

func parseCookieString(cookieStr string) []*http.Cookie {
	cookies := []*http.Cookie{}
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			cookies = append(cookies, &http.Cookie{
				Name:  strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			})
		}
	}
	return cookies
}

func (s *Scraper) loadCookiesFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cookie file: %w", err)
	}
	return s.setSessionCookie(string(data))
}

func (s *Scraper) login(loginURL, username, password string) error {
	formData := url.Values{
		"username": {username},
		"password": {password},
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("do login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("login returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *Scraper) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	fullURL := s.baseURL
	if strings.HasPrefix(path, "http") {
		fullURL = path
	} else {
		fullURL = s.baseURL + path
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	return s.client.Do(req)
}

func (s *Scraper) FetchProjects(projectsPath string) ([]models.Project, error) {
	resp, err := s.doRequest("GET", projectsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch projects page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch projects returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse projects html: %w", err)
	}

	projects := []models.Project{}
	now := time.Now()

	doc.Find("table.project-table tbody tr").Each(func(i int, sel *goquery.Selection) {
		cells := sel.Find("td")
		if cells.Length() < 5 {
			return
		}

		projectID := strings.TrimSpace(cells.Eq(0).Text())
		encryptedID, exists := cells.Eq(0).Attr("data-encrypted-id")
		if exists && encryptedID != "" {
			projectID = encryptedID
		}
		if projectID == "" {
			projectID = fmt.Sprintf("PRJ%05d", i+1)
		}

		name := strings.TrimSpace(cells.Eq(1).Text())
		client := strings.TrimSpace(cells.Eq(2).Text())
		pm := strings.TrimSpace(cells.Eq(3).Text())
		contractAmount := 0.0
		fmt.Sscanf(strings.TrimSpace(cells.Eq(4).Text()), "%f", &contractAmount)

		p := models.Project{
			ID:             projectID,
			Name:           name,
			Client:         client,
			ContractAmount: contractAmount,
			PM:             pm,
			Status:         models.ProjectStatusActive,
			StartDate:      now.AddDate(0, -1, 0),
			PlannedEndDate: now.AddDate(0, 2, 0),
			Milestones:     []models.Milestone{},
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		projects = append(projects, p)
	})

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects found on page, table selector may be wrong")
	}

	return projects, nil
}

func (s *Scraper) FetchMilestones(projectDetailPath string) ([]models.Milestone, error) {
	resp, err := s.doRequest("GET", projectDetailPath, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch milestone page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch milestones returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse milestone html: %w", err)
	}

	milestones := []models.Milestone{}
	now := time.Now()

	doc.Find("table.milestone-table tbody tr").Each(func(i int, sel *goquery.Selection) {
		cells := sel.Find("td")
		if cells.Length() < 7 {
			return
		}

		ms := models.Milestone{
			Number:      i + 1,
			Name:        strings.TrimSpace(cells.Eq(1).Text()),
			PlannedDate: now.AddDate(0, 0, i*7),
			Owner:       strings.TrimSpace(cells.Eq(4).Text()),
			Status:      models.StatusNotStarted,
			Deliverable: strings.TrimSpace(cells.Eq(6).Text()),
			Acceptor:    strings.TrimSpace(cells.Eq(7).Text()),
		}

		statusText := strings.TrimSpace(cells.Eq(5).Text())
		switch statusText {
		case "已完成", "完成":
			ms.Status = models.StatusCompleted
		case "进行中", "进行":
			ms.Status = models.StatusInProgress
		case "已延期", "延期":
			ms.Status = models.StatusDelayed
		case "已取消", "取消":
			ms.Status = models.StatusCanceled
		}

		milestones = append(milestones, ms)
	})

	return milestones, nil
}

func (s *Scraper) SaveCookiesToFile(path string) error {
	parsedURL, err := url.Parse(s.baseURL)
	if err != nil {
		return fmt.Errorf("parse base URL: %w", err)
	}
	cookies := s.client.Jar.Cookies(parsedURL)
	parts := []string{}
	for _, c := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	return os.WriteFile(path, []byte(strings.Join(parts, ";")), 0600)
}

func (s *Scraper) CheckSession(healthCheckPath string) bool {
	resp, err := s.doRequest("GET", healthCheckPath, nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
