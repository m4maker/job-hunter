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

	"golang.org/x/net/html"
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
	log := logger.Get().With().Str("source", "Monster").Logger()
	var baseMonsterURL = "https://www.monster.com/jobs/search"
	urlParams := url.Values{}
	urlParams.Add("q", params.Title)
	urlParams.Add("where", params.Location)
	urlParams.Add("page", "1")
	urlParams.Add("so", "date.desc") // Sort by date, newest first

	log.Info().Str("url", baseMonsterURL+"?"+urlParams.Encode()).Msg("Creating request")
	req, err := http.NewRequestWithContext(ctx, "GET", baseMonsterURL+"?"+urlParams.Encode(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("creating monster request: %w", err)
	}

	// Add more headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	log.Debug().Msg("Sending request")
	resp, err := c.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send request")
		return nil, fmt.Errorf("making monster request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status_code", resp.StatusCode).Msg("Unexpected status code")
		return nil, fmt.Errorf("monster unexpected status code: %d", resp.StatusCode)
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

	// Find job listings - updated selectors for current Monster structure
	var f func(*html.Node)
	f = func(n *html.Node) {
		// Check for job cards with various possible class names
		if n.Type == html.ElementNode && (n.Data == "div" || n.Data == "article" || n.Data == "li") {
			isJobCard := false
			for _, a := range n.Attr {
				if a.Key == "class" && (strings.Contains(a.Val, "job-cardstyle__JobCardComponent") ||
					strings.Contains(a.Val, "results-card") ||
					strings.Contains(a.Val, "job-search-card") ||
					strings.Contains(a.Val, "card-content")) {
					isJobCard = true
					break
				}
			}

			if isJobCard {
				// Extract job details
				job := models.Job{Source: "Monster"}

				// Find title, company, location, and link with updated selectors
				var findDetails func(*html.Node)
				findDetails = func(node *html.Node) {
					if node.Type == html.ElementNode {
						// Check for job title
						if (node.Data == "h3" || node.Data == "h2" || node.Data == "a") && (hasClass(node, "job-cardstyle__JobTitle") ||
							hasClass(node, "title") ||
							hasClass(node, "name") ||
							hasClass(node, "job-title")) {
							if text := getTextContent(node); text != "" {
								job.Title = text
							}
						}

						// Check for company name
						if (node.Data == "span" || node.Data == "div") && (hasClass(node, "job-cardstyle__CompanyName") ||
							hasClass(node, "company") ||
							hasClass(node, "name") ||
							hasClass(node, "company-name")) {
							if text := getTextContent(node); text != "" {
								job.Company = text
							}
						}

						// Check for location
						if (node.Data == "span" || node.Data == "div") && (hasClass(node, "job-cardstyle__Location") ||
							hasClass(node, "location") ||
							hasClass(node, "address")) {
							if text := getTextContent(node); text != "" {
								job.Location = text
							}
						}

						// Check for job URL
						if node.Data == "a" {
							for _, a := range node.Attr {
								if a.Key == "href" && (strings.Contains(a.Val, "/job-openings/") ||
									strings.Contains(a.Val, "/job/") ||
									strings.Contains(a.Val, "/jobs/")) {
									// Ensure it's a full URL
									if strings.HasPrefix(a.Val, "http") {
										job.URL = a.Val
									} else {
										job.URL = "https://www.monster.com" + a.Val
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
				job.ID = fmt.Sprintf("monster-%s-%s", url.QueryEscape(job.Title), url.QueryEscape(job.Company))

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

	log.Info().Int("job_count", len(jobs)).Msg("Completed Monster crawl")
	return jobs, nil
}
