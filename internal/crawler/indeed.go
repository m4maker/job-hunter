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

var baseIndeedURL = "https://api.indeed.com/ads/apisearch"

type IndeedCrawler struct {
	client *http.Client
}

func NewIndeedCrawler() *IndeedCrawler {
	return &IndeedCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *IndeedCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	baseURL := baseIndeedURL
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("l", params.Location)
	urlParams.Add("format", "json")
	urlParams.Add("limit", "25")
	urlParams.Add("start", "0")
	// Note: In a production environment, you would need to sign up for Indeed's API
	// and include your publisher ID and other authentication parameters
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Results []models.Job `json:"results"`
	}
	
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Set the source for all jobs
	for i := range response.Results {
		response.Results[i].Source = "Indeed"
	}

	return response.Results, nil
}
