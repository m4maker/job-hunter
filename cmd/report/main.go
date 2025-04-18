package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"job-hunter/internal/crawler"
	"job-hunter/internal/reporter"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	title := flag.String("title", "", "Job title to search for")
	location := flag.String("location", "", "Job location")
	email := flag.String("email", "", "Email address to send report to")
	dataDir := flag.String("data-dir", "", "Directory to store job data")
	flag.Parse()

	log.Printf("Starting job search with title=%s, location=%s", *title, *location)

	if *title == "" {
		log.Fatal("Job title is required")
	}

	if *email == "" {
		log.Fatal("Email address is required")
	}

	if *dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		*dataDir = filepath.Join(home, ".job-hunter")
	}

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize crawler
	c := crawler.NewJobCrawler()

	// Search for jobs
	params := crawler.JobSearchParams{
		Title:    *title,
		Location: *location,
	}

	log.Printf("Searching for jobs...")
	jobs, err := c.SearchJobs(context.Background(), params)
	if err != nil {
		log.Fatalf("Failed to search jobs: %v", err)
	}
	log.Printf("Found %d jobs", len(jobs))
	for i, job := range jobs {
		log.Printf("Job %d: %s at %s (%s)", i+1, job.Title, job.Company, job.Source)
	}

	// Load previous jobs
	prevJobsFile := filepath.Join(*dataDir, "previous_jobs.txt")
	prevJobs, err := reporter.LoadPreviousJobs(prevJobsFile)
	if err != nil {
		log.Printf("Warning: Failed to load previous jobs: %v", err)
	}

	// Find new jobs
	newJobs := reporter.FindNewJobs(prevJobs, jobs)

	// Save current jobs for next run
	if err := reporter.SaveJobsToFile(jobs, prevJobsFile); err != nil {
		log.Printf("Warning: Failed to save jobs: %v", err)
	}

	// Create report
	report := reporter.JobReport{
		Date:     time.Now(),
		Jobs:     jobs,
		NewJobs:  newJobs,
		Title:    *title,
		Location: *location,
	}

	// Send email
	config := reporter.EmailConfig{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     587, // Default port for TLS
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("FROM_EMAIL"),
		ToEmail:      *email,
	}

	log.Printf("Sending email report to %s", *email)
	if err := reporter.SendJobReport(config, report); err != nil {
		log.Fatalf("Failed to send email report: %v", err)
	}
	log.Printf("Email sent successfully")
}
