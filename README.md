# Job Hunter

Job Hunter is a powerful job search aggregator that collects job listings from multiple sources (LinkedIn, Indeed, Monster, and Glassdoor) and sends you daily email reports. It can be run locally or as a GitHub Action for automated daily job searches.

## Features

- 🔍 Multi-source job search (LinkedIn, Indeed, Monster, Glassdoor)
- 📧 Daily email reports with new job listings
- 🔄 Tracks and highlights new jobs since last search
- 🌍 Location-based search support
- 🚀 Can be run locally or as a GitHub Action
- 📱 Mobile-friendly HTML email format

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/job-hunter.git
cd job-hunter

# Build the project
go build -o job-hunter ./cmd/report
```

## Usage

### Local Usage

```bash
./job-hunter \
  -title="Software Engineer" \
  -location="San Francisco" \
  -email="your-email@example.com"
```

### Command Line Arguments

- `-title` (required): Job title to search for
- `-location` (optional): Job location
- `-email` (required): Email address to send the report to
- `-data-dir` (optional): Directory to store job data (default: ~/.job-hunter)

### Environment Variables

The following environment variables are required for email functionality:

```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_USERNAME=your-email@gmail.com
export SMTP_PASSWORD=your-app-password
export FROM_EMAIL=your-email@gmail.com
```

## Email Setup

The job hunter uses Gmail SMTP to send email reports. Follow these steps to set up your Gmail account:

1. Go to your [Google Account settings](https://myaccount.google.com/)
2. Click on "Security" in the left sidebar
3. Under "Signing in to Google," enable "2-Step Verification" if not already enabled
4. Go back to Security and find "App passwords"
5. Click "App passwords" and sign in if prompted
6. At the bottom:
   - Click "Select app" and choose "Mail"
   - Select "Other (Custom name)" and enter "Job Hunter"
   - Click "Generate"
7. Copy the 16-character password that Google generates

This app password will be used in the GitHub Actions setup.

## GitHub Actions Setup

1. Fork this repository

2. Add the following secrets in your GitHub repository (Settings → Secrets and variables → Actions → New repository secret):
   - `SMTP_HOST`: smtp.gmail.com
   - `SMTP_USERNAME`: Your Gmail address
   - `SMTP_PASSWORD`: The 16-character app password generated above
   - `FROM_EMAIL`: Your Gmail address

3. Add the following variables in your GitHub repository:
   - `JOB_TITLE`: The job title you're looking for
   - `JOB_LOCATION`: Your preferred job location
   - `NOTIFICATION_EMAIL`: Email address to receive reports

The GitHub Action will run daily at 8 AM UTC and send you an email report.

### Manual Trigger

You can also manually trigger the job search from the GitHub Actions tab in your repository.

## Email Report Format

The email report includes:

- Search parameters used (title and location)
- New jobs found since the last search (highlighted)
- Complete list of all jobs found
- Direct links to job postings (when available)

Each job listing includes:
- Job title
- Company name
- Source (LinkedIn, Indeed, etc.)
- Application link

## Development

### Project Structure

```
job-hunter/
├── cmd/
│   ├── report/        # CLI application
│   └── server/        # API server
├── internal/
│   ├── api/          # API handlers
│   ├── crawler/      # Job source crawlers
│   ├── logger/       # Logging utilities
│   ├── models/       # Data models
│   └── reporter/     # Email reporting
└── .github/
    └── workflows/    # GitHub Actions
```

### Running Tests

```bash
go test ./... -v
```

### Adding New Job Sources

To add a new job source:

1. Create a new crawler in `internal/crawler/`
2. Implement the `Source` interface:
   ```go
   type Source interface {
       Crawl(ctx context.Context, params JobSearchParams) ([]models.Job, error)
   }
   ```
3. Add the new source to `NewJobCrawler()` in `crawler.go`

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Thanks to LinkedIn, Indeed, Monster, and Glassdoor for providing job data
- Built with Go and ❤️

## Disclaimer

This tool is for educational purposes only. Please review and comply with the terms of service of each job site before using this tool.
