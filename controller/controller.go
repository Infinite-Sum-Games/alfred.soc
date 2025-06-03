package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	"github.com/IAmRiteshKoushik/alfred/db"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestEndpointHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server is LIVE",
	})
	pkg.Log.Success(c)
}

func WebhookHandler(c *gin.Context) {
	id := c.Param("id")
	repoUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository UUID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := cmd.DBPool.Acquire(ctx)
	defer conn.Release()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to connect to database",
		})
		return
	}
	queries := db.New()

	// Check repository existance for each webhook event before processing it
	exists, err := queries.RepositoryExistsQuery(c, conn, repoUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Oops! Something happened. Please try again later.",
		})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Repository not found",
		})
		return
	}

	// Actuall processing of events
	event := c.GetHeader("X-GitHub-Event")
	switch event {
	case "ping":
		handlePingEvent(c, repoUUID)
	case "issue_comment":
		handleIssueCommentEvent(c, repoUUID)
	case "issues":
		handleIssueEvent(c, repoUUID)
	case "pull_request":
		handlePullRequestEvent(c, repoUUID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported event type",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook event handled successfully",
	})
	pkg.Log.Success(c)
}
