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

var baseLinkedInURL = "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search"

type LinkedInCrawler struct {
	client *http.Client
}

func NewLinkedInCrawler() *LinkedInCrawler {
	return &LinkedInCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *LinkedInCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	baseURL := baseLinkedInURL
	urlParams := url.Values{}
	urlParams.Add("keywords", params.Title)
	urlParams.Add("location", params.Location)
	urlParams.Add("start", "0")
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var jobs []models.Job
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&jobs); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Set the source for all jobs
	for i := range jobs {
		jobs[i].Source = "LinkedIn"
	}

	return jobs, nil
}
