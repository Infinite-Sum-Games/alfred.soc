package controller

import (
	"bytes"
	"io"
	"net/http"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
)

func TestEndpointHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server is LIVE",
	})
	pkg.Log.Success(c)
}

func WebhookHandler(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pkg.Log.Error(c, "Error reading request body during webhook event: %v",
			err,
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(payload))

	// Actuall processing of events
	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		pkg.Log.Warn(c, "Missing X-GitHub-Event header")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Missing X-GitHub-Event header",
		})
		return
	}
	parsedPayload, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		pkg.Log.Error(c, "Error parsing request body during webhook event: %v",
			err,
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch eventType {
	case "ping":
		handlePingEvent(c, parsedPayload)
	case "issue_comment":
		handleIssueCommentEvent(c, parsedPayload)
	case "issues":
		handleIssueEvent(c, parsedPayload)
	case "pull_request":
		handlePullRequestEvent(c, parsedPayload)
	default:
		pkg.Log.Warn(c, "Failed to process GitHub Event: "+eventType)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported event type",
		})
		return
	}
}
