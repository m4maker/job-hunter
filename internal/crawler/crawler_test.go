package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"encoding/json"
	"job-hunter/internal/models"
)

func TestLinkedInCrawler(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("keywords") != "golang" {
			t.Errorf("Expected keywords=golang, got %s", r.URL.Query().Get("keywords"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		mockJobs := []models.Job{
			{
				ID:          "1",
				Title:       "Software Engineer",
				Company:     "Test Company",
				Location:    "Remote",
				PostedDate:  time.Now(),
				Source:      "LinkedIn",
			},
		}
		json.NewEncoder(w).Encode(mockJobs)
	}))
	defer server.Close()

	// Create a test crawler with the mock server URL
	crawler := &LinkedInCrawler{
		client: &http.Client{Timeout: 10 * time.Second},
	}

	// Override the baseURL for testing
	oldURL := baseLinkedInURL
	baseLinkedInURL = server.URL
	defer func() { baseLinkedInURL = oldURL }()

	jobs, err := crawler.Crawl(context.Background(), "golang")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected jobs, got empty result")
	}

	if jobs[0].Title != "Software Engineer" {
		t.Errorf("Expected job title 'Software Engineer', got '%s'", jobs[0].Title)
	}
}

func TestIndeedCrawler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("q") != "golang" {
			t.Errorf("Expected q=golang, got %s", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("Expected format=json, got %s", r.URL.Query().Get("format"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		mockResponse := struct {
			Results []models.Job `json:"results"`
		}{
			Results: []models.Job{
				{
					ID:          "2",
					Title:       "Backend Developer",
					Company:     "Test Corp",
					Location:    "San Francisco",
					PostedDate:  time.Now(),
					Source:      "Indeed",
				},
			},
		}
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	crawler := &IndeedCrawler{
		client: &http.Client{Timeout: 10 * time.Second},
	}

	// Override the baseURL for testing
	oldURL := baseIndeedURL
	baseIndeedURL = server.URL
	defer func() { baseIndeedURL = oldURL }()

	jobs, err := crawler.Crawl(context.Background(), "golang")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected jobs, got empty result")
	}

	if jobs[0].Title != "Backend Developer" {
		t.Errorf("Expected job title 'Backend Developer', got '%s'", jobs[0].Title)
	}
}

func TestJobCrawlerConcurrency(t *testing.T) {
	// Create a JobCrawler with mock sources
	mockSource1 := &mockSource{jobs: []models.Job{{ID: "1", Source: "Mock1"}}}
	mockSource2 := &mockSource{jobs: []models.Job{{ID: "2", Source: "Mock2"}}}

	crawler := &JobCrawler{
		sources: []Source{mockSource1, mockSource2},
	}

	jobs, err := crawler.SearchJobs(context.Background(), JobSearchParams{
		Title:    "Software Engineer",
		Location: "San Francisco",
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
}

// Mock source for testing
type mockSource struct {
	jobs []models.Job
}

func (m *mockSource) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	return m.jobs, nil
}
