package crawler

import (
	"context"
	"log"
	"sync"

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
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []models.Job
	)

	for _, source := range jc.sources {
		wg.Add(1)
		go func(s Source) {
			log.Printf("Searching source: %T", s)
			defer wg.Done()
			
			jobs, err := s.Crawl(ctx, params)
			if err != nil {
				log.Printf("Error from source %T: %v", s, err)
				// Log error but continue with other sources
				return
			}
			
			mu.Lock()
			log.Printf("Found %d jobs from source %T", len(jobs), s)
			results = append(results, jobs...)
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return results, nil
}
