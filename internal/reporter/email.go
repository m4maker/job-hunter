package reporter

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"time"

	"job-hunter/internal/models"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	ToEmail      string
}

type JobReport struct {
	Date     time.Time
	Jobs     []models.Job
	NewJobs  []models.Job // Jobs that weren't in the previous report
	Location string
	Title    string
}

const emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .job { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .new { background-color: #e6ffe6; }
        .title { color: #2c5282; font-size: 18px; margin-bottom: 5px; }
        .company { color: #4a5568; font-size: 16px; font-weight: bold; margin-bottom: 5px; }
        .source { color: #718096; font-size: 14px; }
        .location { color: #4a5568; font-style: italic; margin-bottom: 5px; }
    </style>
</head>
<body>
    <h1>Job Search Report - {{.Date.Format "Jan 02, 2006"}}</h1>
    <h2>Search Parameters</h2>
    <p>Title: {{.Title}}</p>
    <p>Location: {{.Location}}</p>
    
    {{if .NewJobs}}
    <h2>New Jobs Since Last Report</h2>
    {{range .NewJobs}}
    <div class="job new">
        <div class="title">{{.Title}}</div>
        <div class="company">Company: {{.Company}}</div>
        {{if .Location}}<div class="location">Location: {{.Location}}</div>{{end}}
        <div class="source">Source: {{.Source}}</div>
        {{if .URL}}<a href="{{.URL}}">View Job</a>{{end}}
    </div>
    {{end}}
    {{end}}

    <h2>All Jobs</h2>
    {{range .Jobs}}
    <div class="job">
        <div class="title">{{.Title}}</div>
        <div class="company">Company: {{.Company}}</div>
        {{if .Location}}<div class="location">Location: {{.Location}}</div>{{end}}
        <div class="source">Source: {{.Source}}</div>
        {{if .URL}}<a href="{{.URL}}">View Job</a>{{end}}
    </div>
    {{end}}
</body>
</html>
`

func SendJobReport(config EmailConfig, report JobReport) error {
	log.Printf("Generating email for %d jobs (%d new)", len(report.Jobs), len(report.NewJobs))
	// Parse template
	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Generate HTML
	var body bytes.Buffer
	log.Printf("Executing email template")
	if err := tmpl.Execute(&body, report); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Email headers
	headers := map[string]string{
		"From":         config.FromEmail,
		"To":           config.ToEmail,
		"Subject":      fmt.Sprintf("Job Search Report for %s - %s", report.Title, report.Date.Format("Jan 02, 2006")),
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	// Construct message
	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.Write(body.Bytes())

	// Send email
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	log.Printf("Sending email from %s to %s via %s", config.FromEmail, config.ToEmail, addr)
	sendErr := smtp.SendMail(addr, auth, config.FromEmail, []string{config.ToEmail}, message.Bytes())
	if sendErr != nil {
		log.Printf("Email content: %s", body.String())
		return fmt.Errorf("sending mail: %w", sendErr)
	}
	return nil
}

func SaveJobsToFile(jobs []models.Job, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	for _, job := range jobs {
		_, err := fmt.Fprintf(f, "%s\t%s\t%s\t%s\n", job.ID, job.Title, job.Company, job.URL)
		if err != nil {
			return fmt.Errorf("writing job: %w", err)
		}
	}
	return nil
}

func LoadPreviousJobs(filename string) ([]models.Job, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var jobs []models.Job
	lines := bytes.Split(content, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		parts := bytes.Split(line, []byte("\t"))
		if len(parts) >= 4 {
			jobs = append(jobs, models.Job{
				ID:      string(parts[0]),
				Title:   string(parts[1]),
				Company: string(parts[2]),
				URL:     string(parts[3]),
			})
		}
	}
	return jobs, nil
}

func FindNewJobs(previous, current []models.Job) []models.Job {
	seen := make(map[string]bool)
	for _, job := range previous {
		seen[job.ID] = true
	}

	var newJobs []models.Job
	for _, job := range current {
		if !seen[job.ID] {
			newJobs = append(newJobs, job)
		}
	}
	return newJobs
}
