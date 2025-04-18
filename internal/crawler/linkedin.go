package crawler

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	// Read all response bytes
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Parse HTML response
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	// Find job listings
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			// Check if this is a job card
			isJobCard := false
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "job-search-card") {
					isJobCard = true
					break
				}
			}

			if isJobCard {
				// Extract job details
				job := models.Job{Source: "LinkedIn"}

				// Find title, company, and link
				var findDetails func(*html.Node)
				findDetails = func(node *html.Node) {
					if node.Type == html.ElementNode {
						switch node.Data {
						case "h3":
							// Job title
							if node.FirstChild != nil {
								job.Title = node.FirstChild.Data
							}
						case "h4":
							// Company name
							if node.FirstChild != nil {
								job.Company = node.FirstChild.Data
							}
						case "a":
							// Job URL
							for _, a := range node.Attr {
								if a.Key == "href" {
									job.URL = a.Val
									break
								}
							}
						}
					}
					for c := node.FirstChild; c != nil; c = c.NextSibling {
						findDetails(c)
					}
				}

				findDetails(n)
				
				// Generate a unique ID
				job.ID = fmt.Sprintf("linkedin-%s-%s", url.QueryEscape(job.Title), url.QueryEscape(job.Company))
				
				// Add job if we have the minimum required fields
				if job.Title != "" && job.Company != "" {
					jobs = append(jobs, job)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return jobs, nil
}
