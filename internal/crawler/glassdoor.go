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

type GlassdoorCrawler struct {
	client *RateLimitedClient
}

func NewGlassdoorCrawler() *GlassdoorCrawler {
	return &GlassdoorCrawler{
		client: NewRateLimitedClient(0.5), // 1 request every 2 seconds to be conservative
	}
}

func (c *GlassdoorCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	baseURL := "https://www.glassdoor.com/Job/jobs.htm"
	urlParams := url.Values{}
	urlParams.Add("sc.keyword", params.Title)
	urlParams.Add("locT", params.Location)
	urlParams.Add("format", "json")
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating glassdoor request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	
	// In production, you would need to add authentication headers
	// req.Header.Set("Authorization", "Bearer YOUR_API_KEY")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making glassdoor request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("glassdoor unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		JobListings []struct {
			JobTitle     string    `json:"jobTitle"`
			CompanyName  string    `json:"employer"`
			Location     string    `json:"location"`
			ListingURL   string    `json:"jobViewUrl"`
			DatePosted   time.Time `json:"datePosted"`
			SalaryRange  string    `json:"salaryRange,omitempty"`
			Description  string    `json:"jobDescription"`
		} `json:"listings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding glassdoor response: %w", err)
	}

	jobs := make([]models.Job, len(response.JobListings))
	for i, listing := range response.JobListings {
		jobs[i] = models.Job{
			Title:       listing.JobTitle,
			Company:     listing.CompanyName,
			Location:    listing.Location,
			URL:         listing.ListingURL,
			PostedDate:  listing.DatePosted,
			Salary:      listing.SalaryRange,
			Description: listing.Description,
			Source:      "Glassdoor",
		}
	}

	return jobs, nil
}
