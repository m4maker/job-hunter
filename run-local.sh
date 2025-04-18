#!/bin/bash

# Build the application
go build -o job-hunter ./cmd/report

# Set environment variables
export SMTP_HOST="smtp.gmail.com"
export SMTP_USERNAME="bmrjunkie@gmail.com"  # Replace with your Gmail
export SMTP_PASSWORD="vwwl lfrf efgv cdeo"     # Replace with your Gmail app password
export FROM_EMAIL="bmrjunkie@gmail.com"     # Replace with your Gmail

# Run the job hunter
./job-hunter \
  -title="IT-Director" \
  -location="San Diego" \
  -email="bmrjunkie@gmail.com" \
  -data-dir="./data"
