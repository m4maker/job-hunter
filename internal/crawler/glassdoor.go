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

type GlassdoorCrawler struct {
	client *RateLimitedClient
}

func NewGlassdoorCrawler() *GlassdoorCrawler {
	return &GlassdoorCrawler{
		client: NewRateLimitedClient(0.5), // 1 request every 2 seconds to be conservative
	}
}

func (c *GlassdoorCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	var baseGlassdoorURL = "https://www.glassdoor.com/Job/jobs.htm"
	urlParams := url.Values{}
	urlParams.Add("sc.keyword", params.Title)
	urlParams.Add("locT", params.Location)
	urlParams.Add("format", "json")
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseGlassdoorURL+"?"+urlParams.Encode(), nil)
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
		if n.Type == html.ElementNode && n.Data == "li" {
			// Check if this is a job card
			isJobCard := false
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "react-job-listing") {
					isJobCard = true
					break
				}
			}

			if isJobCard {
				// Extract job details
				job := models.Job{Source: "Glassdoor"}

				// Find title, company, location, and link
				var findDetails func(*html.Node)
				findDetails = func(node *html.Node) {
					if node.Type == html.ElementNode {
						switch {
						case node.Data == "a" && hasClass(node, "jobLink"):
							// Job title
							if text := getTextContent(node); text != "" {
								job.Title = text
							}
							// Job URL
							for _, a := range node.Attr {
								if a.Key == "href" {
									job.URL = "https://www.glassdoor.com" + a.Val
									break
								}
							}
						case node.Data == "span" && hasClass(node, "companyName"):
							// Company name
							if text := getTextContent(node); text != "" {
								job.Company = text
							}
						case node.Data == "span" && hasClass(node, "location"):
							// Location
							if text := getTextContent(node); text != "" {
								job.Location = text
							}
						}
					}
					for c := node.FirstChild; c != nil; c = c.NextSibling {
						findDetails(c)
					}
				}

				findDetails(n)
				
				// Generate a unique ID
				job.ID = fmt.Sprintf("glassdoor-%s-%s", url.QueryEscape(job.Title), url.QueryEscape(job.Company))
				
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
