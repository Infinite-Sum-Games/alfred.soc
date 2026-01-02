package controller

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	db "github.com/IAmRiteshKoushik/alfred/db/gen"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
	"github.com/jackc/pgx/v5/pgtype"
)

func handleIssueEvent(c *gin.Context, payload any) {

	issueEvent, ok := payload.(*github.IssuesEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to parse Issue-Event",
			fmt.Errorf("Malformed event payload received in Issue-Event"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	event := *issueEvent.Action

	switch event {

	case "labeled":
		label := strings.ToUpper(*issueEvent.Label.Name)
		if label == "AMSOC-ACCEPTED" {
			title := *issueEvent.Issue.Title
			repoUrl := *issueEvent.Repo.HTMLURL
			issueAccepted(c, title, repoUrl, *issueEvent.Issue.HTMLURL)
			return
		}
		if slices.Contains([]string{"EASY", "MEDIUM", "HARD"}, label) {
			updateIssueDifficulty(c, *issueEvent.Issue.HTMLURL, label)
			return
		}
		if strings.HasPrefix(label, "BOUNTY-") {
			updateIssueBounty(c, *issueEvent.Issue.HTMLURL, label)
			return
		}
		issueTagUpdate(c, *issueEvent.Issue.HTMLURL, label)
		return

	case "assigned", "unassigned":
		if issueEvent.Assignee == nil {
			pkg.Log.Warn(c, "Assignee is nil, skipping issue user action")
			c.JSON(http.StatusOK, gin.H{"message": "Assignee is nil, skipping action"})
			return
		}
		username := *issueEvent.Assignee.Login
		issueUrl := *issueEvent.Issue.HTMLURL
		issueUserAction(c, username, issueUrl, event)
		return

	case "closed", "reopened":
		issueStateChangeAction(c, *issueEvent.Issue.HTMLURL, event)
		return

	default:
		c.JSON(http.StatusOK, gin.H{
			"message": "This issue event-type" + event + " is not handled.",
		})
		pkg.Log.Warn(c, event+" is not handled.")
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
		Difficulty: difficulty,
		Url:        issueUrl,
	}
	result, err := q.UpdateIssueDifficultyQuery(ctx, tx, params)
	if err != nil || len(result) == 0 {
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
		ArrayAppend: tag,
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

func updateIssueBounty(c *gin.Context, issueUrl string, bounty string) {
	bountyVal, err := strconv.Atoi(strings.TrimPrefix(bounty, "BOUNTY-"))
	if err != nil {
		pkg.Log.Error(c, "Failed to parse bounty value", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid bounty value",
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
	params := db.UpdateIssueBountyQueryParams{
		BountyPromised: int32(bountyVal),
		Url:            issueUrl,
	}
	result, err := q.UpdateIssueBountyQuery(ctx, tx, params)
	if err != nil || len(result) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to update issue bounty",
		})
		pkg.Log.Error(c, "Failed to update issue bounty", err)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to commit transaction",
		})
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully updated issue bounty")
	c.JSON(http.StatusOK, gin.H{
		"message": "Issue bounty updated successfully",
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

	exists, err := q.ParticipantExistsQuery(ctx, tx, pgtype.Text{String: username, Valid: true})
	if err != nil {
		pkg.Log.Error(c, "Failed to check participant existence", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to check participant existence"})
		return
	}
	if !exists {
		pkg.Log.Warn(c, "Participant not found")
		c.JSON(http.StatusNotFound, gin.H{"message": "Participant not found"})
		return
	}

	switch action {
	case "assigned":
		params := db.IssueAssignQueryParams{
			IssueUrl:   url,
			Ghusername: username,
			ClaimedOn:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			ElapsedOn:  pgtype.Timestamp{Time: time.Now().Add(8 * 24 * time.Hour), Valid: true},
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
		if err != nil {
			pkg.Log.Error(c, "Failed to unassign issue", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if ok == "" {
			pkg.Log.Warn(c, "Issue not found or user not assigned")
			c.AbortWithStatus(http.StatusNotFound)
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

// Issue: CLOSED, REOPENED
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
