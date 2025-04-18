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
	var baseMonsterURL = "https://www.monster.com/jobs/search"
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("where", params.Location)
	urlParams.Add("intcid", "api_search")
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseMonsterURL+"?"+urlParams.Encode(), nil)
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
				if a.Key == "class" && strings.Contains(a.Val, "job-cardstyle__JobCardComponent") {
					isJobCard = true
					break
				}
			}

			if isJobCard {
				// Extract job details
				job := models.Job{Source: "Monster"}

				// Find title, company, location, and link
				var findDetails func(*html.Node)
				findDetails = func(node *html.Node) {
					if node.Type == html.ElementNode {
						switch {
						case node.Data == "h3" && hasClass(node, "job-cardstyle__JobTitle"):
							// Job title
							if text := getTextContent(node); text != "" {
								job.Title = text
							}
						case node.Data == "span" && hasClass(node, "job-cardstyle__CompanyName"):
							// Company name
							if text := getTextContent(node); text != "" {
								job.Company = text
							}
						case node.Data == "span" && hasClass(node, "job-cardstyle__Location"):
							// Location
							if text := getTextContent(node); text != "" {
								job.Location = text
							}
						case node.Data == "a":
							// Job URL
							for _, a := range node.Attr {
								if a.Key == "href" && strings.Contains(a.Val, "/job-openings/") {
									job.URL = "https://www.monster.com" + a.Val
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
				job.ID = fmt.Sprintf("monster-%s-%s", url.QueryEscape(job.Title), url.QueryEscape(job.Company))
				
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


