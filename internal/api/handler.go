package api

import (
	"context"
	"net/http"
	"job-hunter/internal/crawler"
	"job-hunter/internal/models"
	"github.com/gin-gonic/gin"
)

type JobSearcher interface {
	SearchJobs(ctx context.Context, params crawler.JobSearchParams) ([]models.Job, error)
}

type Handler struct {
	crawler JobSearcher
}

func NewHandler(crawler *crawler.JobCrawler) *Handler {
	return &Handler{crawler: crawler}
}

func (h *Handler) GetJobs(c *gin.Context) {
	jobs, err := h.crawler.SearchJobs(c.Request.Context(), crawler.JobSearchParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) SearchJobs(c *gin.Context) {
	title := c.Query("title")
	location := c.Query("location")

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'title' is required"})
		return
	}

	jobs, err := h.crawler.SearchJobs(c.Request.Context(), crawler.JobSearchParams{
		Title:    title,
		Location: location,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}
