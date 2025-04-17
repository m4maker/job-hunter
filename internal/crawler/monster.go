package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"job-hunter/internal/models"
)

type MonsterCrawler struct {
	client *RateLimitedClient
}

func NewMonsterCrawler() *MonsterCrawler {
	return &MonsterCrawler{
		client: NewRateLimitedClient(1), // 1 request per second to be conservative
	}
}

func (c *MonsterCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	baseURL := "https://www.monster.com/jobs/search"
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("where", params.Location)
	urlParams.Add("intcid", "api_search")
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating monster request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making monster request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("monster unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		JobResults []struct {
			Title       string    `json:"title"`
			Company     string    `json:"company"`
			Location    string    `json:"location"`
			URL         string    `json:"jobUrl"`
			PostedDate  time.Time `json:"postedDate"`
			Salary      string    `json:"estimatedSalary,omitempty"`
			Description string    `json:"description"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding monster response: %w", err)
	}

	jobs := make([]models.Job, len(response.JobResults))
	for i, result := range response.JobResults {
		jobs[i] = models.Job{
			Title:       result.Title,
			Company:     result.Company,
			Location:    result.Location,
			URL:         result.URL,
			PostedDate:  result.PostedDate,
			Salary:      result.Salary,
			Description: result.Description,
			Source:      "Monster",
		}
	}

	return jobs, nil
}
