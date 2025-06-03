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
	"github.com/jackc/pgx/v5/pgtype"
)

func handleIssueEvent(c *gin.Context, id uuid.UUID) {
	// Only [ labelled, assigned, unassigned, closed, reopened ]
	// issue-events are to be handled
	switch event := c.GetString("event"); event {
	case "labelled":
		issueLabelledAction(c, id)
	case "assigned":
		issueUserAction(c, id, event)
	case "unassigned":
		issueUserAction(c, id, event)
	case "closed":
		issueStateChangeAction(c, "closed")
	case "reopened":
		issueStateChangeAction(c, "reopened")
	default:
		c.JSON(http.StatusOK, gin.H{
			"message": "This issue event-type" + event + " is not handled.",
		})
		pkg.Log.Warn(c, event+"is not handled.")
		return
	}
}

func issueLabelledAction(c *gin.Context, id uuid.UUID) {
	// TODO: Get issue details from context (parse the payload)
	url := c.GetString("url")
	title := c.GetString("title")
	if url == "" || title == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Issue URL or title not found",
		})
		pkg.Log.Warn(c, "Missing issue details - URL or title empty")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		pkg.Log.Error(c, "Failed to begin transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to check issue status",
		})
		return
	}

	q := db.New()
	// Check if issue already exists and is open before attempting to create it
	isOpen, err := q.CheckOpenIssueByUrlQuery(ctx, tx, url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to check issue status",
		})
		pkg.Log.Error(c, "Failed to check issue status", err)
		return
	}

	if isOpen {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Issue already exists and is open",
		})
		pkg.Log.Warn(c, "Attempted to add duplicate issue: "+url)
		return
	}

	params := db.AddNewIssueQueryParams{
		ID:     uuid.New(),
		Title:  title,
		Repoid: id,
		Url:    url,
	}

	_, err = q.AddNewIssueQuery(ctx, tx, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to add new issue",
		})
		pkg.Log.Error(c, "Failed to add new issue", err)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to commit transaction",
		})
		pkg.Log.Error(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully added new issue: "+url)
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue added successfully",
	})

}

func issueUserAction(c *gin.Context, id uuid.UUID, action string) {
	// TODO: Fix url and username acquiring from context after parsing the request
	url := c.GetString("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Issue URL not found",
		})
		return
	}
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Username not found",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		pkg.Log.Error(c, "Failed to begin transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Oops! Something happened. Please try again later.",
		})
		return
	}
	defer tx.Rollback(ctx)

	q := db.New()

	switch action {
	case "assigned":
		params := db.CreateIssueClaimQueryParams{
			Url:        url,
			Ghusername: username,
			ClaimedOn:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			ElapsedOn:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		}
		err = q.CreateIssueClaimQuery(ctx, tx, params)

	case "unassigned":
		params := db.RemoveIssueClaimsQueryParams{
			Url:        url,
			Ghusername: username,
		}
		_, err = q.RemoveIssueClaimsQuery(ctx, tx, params)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid action provided",
		})
		pkg.Log.Warn(c, "Invalid action provided for issue user action")
		return
	}

	if err != nil {
		pkg.Log.Error(c, "Failed to process issue user action", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Oops! Something happened. Please try again later.",
		})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		pkg.Log.Error(c, "Failed to commit transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Oops! Something happened. Please try again later.",
		})
		return
	}

	pkg.Log.Info(c, "User "+action+" successfully.")
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue user action processed successfully",
	})
}

func issueStateChangeAction(c *gin.Context, state string) {
	// TODO: Get issue URL from context after parsing the payload
	url := c.GetString("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Issue URL not found",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		pkg.Log.Error(c, "Failed to begin transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to check issue status",
		})
		return
	}
	defer tx.Rollback(ctx)
	q := db.New()

	switch state {
	case "closed":
		_, err = q.CloseIssueQuery(c, tx, url)
	case "reopened":
		_, err = q.ReopenIssueQuery(c, tx, url)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid state change provided for issue",
		})
		pkg.Log.Warn(c, "Invalid state change provided for issue")
		return
	}

	if err != nil {
		pkg.Log.Error(c, "Failed to update issue state", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Oops! Something happened. Please try again later.",
		})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		pkg.Log.Error(c, "Failed to commit transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Oops! Something happened. Please try again later.",
		})
		return
	}

	pkg.Log.Info(c, "Successfully updated issue state to "+state)
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue " + state + " successfully",
	})
}
