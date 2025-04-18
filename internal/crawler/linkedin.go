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
	"job-hunter/internal/logger"
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

// cleanText removes extra whitespace and newlines from text
func cleanText(s string) string {
	// Remove newlines and extra spaces
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func (c *LinkedInCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	log := logger.Get().With().Str("source", "LinkedIn").Logger()
	baseURL := baseLinkedInURL
	urlParams := url.Values{}
	urlParams.Add("keywords", params.Title)
	urlParams.Add("location", params.Location)
	urlParams.Add("start", "0")
	
	log.Info().Str("url", baseURL+"?"+urlParams.Encode()).Msg("Creating request")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Add headers to mimic browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")
	
	log.Debug().Msg("Sending request")
	resp, err := c.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send request")
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status_code", resp.StatusCode).Msg("Unexpected status code")
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	log.Debug().Int("status_code", resp.StatusCode).Msg("Request successful")

	var jobs []models.Job

	// Read all response bytes
	log.Debug().Msg("Reading response body")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Parse HTML response
	log.Debug().Msg("Parsing HTML response")
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse HTML")
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
								job.Title = cleanText(node.FirstChild.Data)
							}
						case "h4":
							// Company name
							if node.FirstChild != nil {
								job.Company = cleanText(node.FirstChild.Data)
							}
						case "span":
							// Location
							for _, a := range node.Attr {
								if a.Key == "class" && strings.Contains(a.Val, "job-search-card__location") {
									if node.FirstChild != nil {
										job.Location = cleanText(node.FirstChild.Data)
									}
									break
								}
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
					log.Debug().Str("title", job.Title).Str("company", job.Company).Str("location", job.Location).Msg("Found job")
					jobs = append(jobs, job)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	log.Info().Int("job_count", len(jobs)).Msg("Completed LinkedIn crawl")
	return jobs, nil
}
