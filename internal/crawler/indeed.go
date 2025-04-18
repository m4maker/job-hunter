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

var baseIndeedURL = "https://www.indeed.com/jobs"

var userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

type IndeedCrawler struct {
	client *http.Client
}

// Helper function to check if a node has a specific class
func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, class) {
			return true
		}
	}
	return false
}

// Helper function to get text content from a node
func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += getTextContent(c)
	}
	return strings.TrimSpace(result)
}

func NewIndeedCrawler() *IndeedCrawler {
	return &IndeedCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *IndeedCrawler) Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	log := logger.Get().With().Str("source", "Indeed").Logger()
	baseURL := baseIndeedURL
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("l", params.Location)
	urlParams.Add("format", "json")
	urlParams.Add("limit", "25")
	urlParams.Add("start", "0")
	// Note: In a production environment, you would need to sign up for Indeed's API
	// and include your publisher ID and other authentication parameters
	
	log.Info().Str("url", baseURL+"?"+urlParams.Encode()).Msg("Creating request")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("User-Agent", userAgent)
	
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
				if a.Key == "class" && strings.Contains(a.Val, "job_seen_beacon") {
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
						switch {
						case node.Data == "h2" && hasClass(node, "jobTitle"):
							// Job title
							if text := getTextContent(node); text != "" {
								job.Title = text
							}
						case node.Data == "span" && hasClass(node, "companyName"):
							// Company name
							if text := getTextContent(node); text != "" {
								job.Company = text
							}
						case node.Data == "div" && hasClass(node, "companyLocation"):
							// Location
							if text := getTextContent(node); text != "" {
								job.Location = text
							}
						case node.Data == "a":
							// Job URL
							for _, a := range node.Attr {
								if a.Key == "href" && strings.Contains(a.Val, "/viewjob?") {
									job.URL = "https://www.indeed.com" + a.Val
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
