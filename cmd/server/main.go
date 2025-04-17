package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"job-hunter/internal/api"
	"job-hunter/internal/crawler"
	"job-hunter/internal/logger"
)

func main() {
	// Initialize logger
	logger.Init()

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	// Use gin middleware
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware())
	
	// Initialize crawlers
	jobCrawler := crawler.NewJobCrawler()
	
	// Initialize API handlers
	handler := api.NewHandler(jobCrawler)
	
	// Routes
	r.GET("/api/jobs", handler.GetJobs)
	r.GET("/api/jobs/search", handler.SearchJobs)
	
	// Start server
	log.Info().Msg("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
