package crawler

import (
	"context"
	"job-hunter/internal/models"
	"sync"
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
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []models.Job
	)

	for _, source := range jc.sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			
			jobs, err := s.Crawl(ctx, params)
			if err != nil {
				// Log error but continue with other sources
				return
			}
			
			mu.Lock()
			results = append(results, jobs...)
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return results, nil
}
