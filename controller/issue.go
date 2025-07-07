package controller

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	"github.com/IAmRiteshKoushik/alfred/db"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func handleIssueEvent(c *gin.Context, payload any) {
	// Only [ labelled, assigned, unassigned, closed, reopened ] to be handled

	issueEvent, ok := payload.(*github.IssueEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to parse Issue-Event",
			fmt.Errorf("Malformed event payload received in Issue-Event"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	username := *issueEvent.Issue.User.Login
	issueUrl := *issueEvent.URL
	event := *issueEvent.Event

	switch event {
	case "labelled":
		label := strings.ToLower(*issueEvent.Label.Name)
		if label == "amsoc-accepted" {
			title := *issueEvent.Issue.Title
			repoUrl := *issueEvent.Repository.HTMLURL
			issueAccepted(c, title, issueUrl, repoUrl)
			return
		}
		if slices.Contains([]string{"easy", "medium", "hard"}, label) {
			updateIssueDifficulty(c, issueUrl, label)
			return
		}
		issueTagUpdate(c, issueUrl, label)
		return
	case "assigned", "unassigned":
		issueUserAction(c, username, issueUrl, event)
		return
	case "closed", "reopened":
		issueStateChangeAction(c, issueUrl, event)
		return
	default:
		c.JSON(http.StatusOK, gin.H{
			"message": "This issue event-type" + event + " is not handled.",
		})
		pkg.Log.Warn(c, event+"is not handled.")
		return
	}
}

func issueAccepted(c *gin.Context, title, repoUrl string, url string) {
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
	// Check if issue already exists and is open before attempting to create it
	isOpen, err := q.CheckOpenIssueQuery(ctx, tx, url)
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
		ID:      uuid.New(),
		Title:   title,
		Repourl: repoUrl,
		Url:     url,
	}
	err = q.AddNewIssueQuery(ctx, tx, params)
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
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully added new issue: "+title)
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue added successfully",
	})
}

func updateIssueDifficulty(c *gin.Context, issueUrl string, difficulty string) {

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
	params := db.UpdateIssueDifficultyQueryParams{
		Difficulty: pgtype.Text{String: strings.ToTitle(difficulty), Valid: true},
		Url:        issueUrl,
	}
	result, err := q.UpdateIssueDifficultyQuery(ctx, tx, params)
	if err != nil || len(result) != 1 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to update issue difficulty",
		})
		pkg.Log.Error(c, "Failed to update issue difficulty", err)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to commit transaction",
		})
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully updated issue difficulty")
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue added successfully",
	})
}

func issueTagUpdate(c *gin.Context, issueUrl string, tag string) {

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

	params := db.AddIssueTagQueryParams{
		ArrayAppend: []string{tag},
		Url:         issueUrl,
	}

	tags, err := q.AddIssueTagQuery(ctx, tx, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to add issue tag",
		})
		pkg.Log.Error(c, "Failed to add issue tag", err)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to commit transaction",
		})
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully updated issue difficulty")
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue added successfully",
		"tags":    tags,
	})
}

func issueUserAction(c *gin.Context, username string, url string, action string) {

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
		params := db.IssueAssignQueryParams{
			IssueUrl:   url,
			Ghusername: username,
			ClaimedOn:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			ElapsedOn:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		}
		err = q.IssueAssignQuery(ctx, tx, params)
		if err != nil {
			pkg.Log.Error(c, "Failed to assign issue", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

	case "unassigned":
		params := db.IssueUnassignQueryParams{
			IssueUrl:   url,
			Ghusername: username,
		}
		ok, err := q.IssueUnassignQuery(ctx, tx, params)
		if err != nil || ok == "" {
			pkg.Log.Error(c, "Failed to unassign issue", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

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

func issueStateChangeAction(c *gin.Context, url string, state string) {
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
		_, err = q.OpenIssueQuery(c, tx, url)
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
