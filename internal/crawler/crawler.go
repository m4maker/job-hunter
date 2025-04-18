package crawler

import (
	"context"
	"log"
	"time"

	"job-hunter/internal/models"
)

type JobCrawler struct {
	sources []Source
}

type JobSearchParams struct {
	Title    string
	Location string
}

type Source interface {
	Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error)
}

func NewJobCrawler() *JobCrawler {
	return &JobCrawler{
		sources: []Source{
			NewLinkedInCrawler(),
			NewIndeedCrawler(),
			NewMonsterCrawler(),
			NewGlassdoorCrawler(),
		},
	}
}

// getSourceName returns a human-readable name for a crawler source
func getSourceName(s Source) string {
	switch s.(type) {
	case *LinkedInCrawler:
		return "LinkedIn"
	case *IndeedCrawler:
		return "Indeed"
	case *MonsterCrawler:
		return "Monster"
	case *GlassdoorCrawler:
		return "Glassdoor"
	default:
		return "Unknown"
	}
}

func (jc *JobCrawler) SearchJobs(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	log.Printf("Starting job search with %d sources: %v", len(jc.sources), func() []string {
		var names []string
		for _, s := range jc.sources {
			names = append(names, getSourceName(s))
		}
		return names
	}())
	var (
		results []models.Job
		errors  []error
	)

	// Create a timeout context for each crawler
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute) // Increased timeout since we're running sequentially
	defer cancel()

	for _, source := range jc.sources {
		sourceName := getSourceName(source)
		log.Printf("[%s] Starting search for %s in %s", sourceName, params.Title, params.Location)

		// Run crawler with timeout
		jobs, err := source.Crawl(ctxWithTimeout, params)
		
		if err != nil {
			if ctxWithTimeout.Err() != nil {
				log.Printf("[%s] Timed out", sourceName)
			} else {
				log.Printf("[%s] Error: %v", sourceName, err)
			}
			errors = append(errors, err)
			continue // Continue with next source even if this one fails
		}

		log.Printf("[%s] Found %d jobs with titles: %v", sourceName, len(jobs), func() []string {
			var titles []string
			for i, j := range jobs {
				if i < 3 { // Only show first 3 jobs to avoid log spam
					titles = append(titles, j.Title)
				}
			}
			if len(jobs) > 3 {
				titles = append(titles, "...")
			}
			return titles
		}())
		results = append(results, jobs...)

		// Add a small delay between crawlers to be nice to the servers
		time.Sleep(2 * time.Second)
	}

	// Log summary
	log.Printf("Search complete. Found %d total jobs", len(results))
	if len(errors) > 0 {
		log.Printf("Warning: %d sources had errors", len(errors))
	}

	return results, nil
}
