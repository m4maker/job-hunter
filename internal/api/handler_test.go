package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"job-hunter/internal/crawler"
	"job-hunter/internal/models"
	"github.com/gin-gonic/gin"
)

func TestGetJobs(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create handler with mock crawler
	mockCrawler := &mockJobCrawler{
		jobs: []models.Job{
			{
				ID:      "1",
				Title:   "Test Job",
				Company: "Test Company",
				Source:  "Test Source",
			},
		},
	}

	handler := &Handler{crawler: mockCrawler}

	// Create test router
	r := gin.New()
	r.GET("/api/jobs", handler.GetJobs)

	// Create test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/jobs", nil)
	r.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []models.Job
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 job, got %d", len(response))
	}
}

func TestSearchJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockCrawler := &mockJobCrawler{
		jobs: []models.Job{
			{
				ID:      "1",
				Title:   "Golang Developer",
				Company: "Tech Corp",
				Source:  "Test Source",
			},
		},
	}

	handler := &Handler{crawler: mockCrawler}
	r := gin.New()
	r.GET("/api/jobs/search", handler.SearchJobs)

	// Test with query parameter
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/jobs/search?title=golang&location=remote", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test without query parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/jobs/search", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

type mockJobCrawler struct {
	jobs []models.Job
}

func (m *mockJobCrawler) SearchJobs(ctx context.Context, params crawler.JobSearchParams) ([]models.Job, error) {
	return m.jobs, nil
}
