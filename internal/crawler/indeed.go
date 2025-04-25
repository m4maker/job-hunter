package crawler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"job-hunter/internal/logger"
	"job-hunter/internal/models"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var baseIndeedURL = "https://www.indeed.com/jobs"

var userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

type IndeedCrawler struct {
	client *http.Client
}

func NewIndeedCrawler() *IndeedCrawler {
	return &IndeedCrawler{
		client: &http.Client{
			Timeout: 15 * time.Second, // Increased timeout
		},
	}
}

func (c *IndeedCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	log := logger.Get().With().Str("source", "Indeed").Logger()
	baseURL := baseIndeedURL
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("l", params.Location)
	urlParams.Add("sort", "date") // Sort by date to get newest jobs
	urlParams.Add("limit", "25")

	log.Info().Str("url", baseURL+"?"+urlParams.Encode()).Msg("Creating request")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add more headers to mimic a real browser
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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

	// Find job listings - updated selectors for current Indeed structure
	var f func(*html.Node)
	f = func(n *html.Node) {
		// Look for job cards with updated class names
		if n.Type == html.ElementNode && (n.Data == "div" || n.Data == "li") {
			isJobCard := false
			for _, a := range n.Attr {
				// Check multiple possible class names that Indeed might use
				if a.Key == "class" && (strings.Contains(a.Val, "job_seen_beacon") ||
					strings.Contains(a.Val, "jobsearch-ResultsList") ||
					strings.Contains(a.Val, "tapItem") ||
					strings.Contains(a.Val, "job-container")) {
					isJobCard = true
					break
				}
			}

			if isJobCard {
				// Extract job details
				job := models.Job{Source: "Indeed"}

				// Find title, company, location, and link
				var findDetails func(*html.Node)
				findDetails = func(node *html.Node) {
					if node.Type == html.ElementNode {
						// Check for job title in various elements
						if (node.Data == "h2" || node.Data == "a") &&
							(hasClass(node, "jobTitle") ||
								hasClass(node, "jcs-JobTitle") ||
								hasClass(node, "title")) {
							if text := getTextContent(node); text != "" {
								job.Title = text
							}
						}

						// Check for company name
						if (node.Data == "span" || node.Data == "div") &&
							(hasClass(node, "companyName") ||
								hasClass(node, "company-name") ||
								hasClass(node, "companyInfo")) {
							if text := getTextContent(node); text != "" {
								job.Company = text
							}
						}

						// Check for location
						if (node.Data == "div" || node.Data == "span") &&
							(hasClass(node, "companyLocation") ||
								hasClass(node, "location") ||
								hasClass(node, "job-location")) {
							if text := getTextContent(node); text != "" {
								job.Location = text
							}
						}

						// Check for job URL
						if node.Data == "a" {
							for _, a := range node.Attr {
								if a.Key == "href" && (strings.Contains(a.Val, "/viewjob?") ||
									strings.Contains(a.Val, "/job/") ||
									strings.Contains(a.Val, "/pagead/")) {
									// Ensure it's a full URL
									if strings.HasPrefix(a.Val, "/") {
										job.URL = "https://www.indeed.com" + a.Val
									} else {
										job.URL = a.Val
									}
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
				job.ID = fmt.Sprintf("indeed-%s-%s", url.QueryEscape(job.Title), url.QueryEscape(job.Company))

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

	log.Info().Int("job_count", len(jobs)).Msg("Completed Indeed crawl")
	return jobs, nil
}
