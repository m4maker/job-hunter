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

func (jc *JobCrawler) SearchJobs(ctx context.Context, params JobSearchParams) ([]models.Job, error) {
	log.Printf("Starting job search with %d sources", len(jc.sources))
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

		log.Printf("[%s] Found %d jobs", sourceName, len(jobs))
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
