package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	db "github.com/IAmRiteshKoushik/alfred/db/gen"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
)

type Solution struct {
	Username string `json:"github_username"`
	Url      string `json:"pull_request_url"`
	Merged   bool   `json:"merged"`
}

func handlePullRequestEvent(c *gin.Context, payload any) {
	prEvent, ok := payload.(*github.PullRequestEvent)
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	prUrl := *prEvent.PullRequest.HTMLURL
	repoUrl := *prEvent.Repo.HTMLURL
	username := *prEvent.PullRequest.User.Login
	action := *prEvent.Action
	isMerged := *prEvent.PullRequest.Merged

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		pkg.Log.Fatal(c, "Could not being transaction", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)
	q := db.New()

	switch action {
	case "opened":
		// DB Call
		_, err := q.AddSolutionQuery(ctx, tx, db.AddSolutionQueryParams{
			Url:        prUrl,
			RepoUrl:    repoUrl,
			Ghusername: username,
		})
		if err != nil {
			pkg.Log.Fatal(c, "Could not add solution to database", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Redis Call
		jsonData, err := json.Marshal(Solution{
			Username: username,
			Url:      prUrl,
			Merged:   false,
		})
		if err != nil {
			pkg.Log.Error(c, "Failed to marshal payload", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		err = cmd.AddToStream(pkg.Valkey, pkg.SolutionMerge, string(jsonData))
		if err != nil {
			pkg.Log.Error(c, "Failed to insert into Redis", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	case "closed":
		if isMerged {
			// DB Call
			ok, err := q.CheckIfSolutionExist(ctx, tx, prUrl)
			if err != nil {
				pkg.Log.Error(c, "Could not check if solution exist", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if ok == 0 {
				pkg.Log.Warn(c, "Solution does not exist")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			update, err := q.MergeSolutionQuery(ctx, tx, prUrl)
			if err != nil {
				pkg.Log.Error(c, "Could not update solution as merged", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if update == "" {
				pkg.Log.Warn(c, "No solution with the given prUrl found")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			// Redis Call
			jsonData, err := json.Marshal(Solution{
				Username: username,
				Url:      prUrl,
				Merged:   true,
			})
			if err != nil {
				pkg.Log.Error(c, "Failed to marshal payload", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			err = cmd.AddToStream(pkg.Valkey, pkg.SolutionMerge, string(jsonData))
			if err != nil {
				pkg.Log.Error(c, "Failed to insert into Redis", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		} else {
			_, err := q.DeleteSolutionQuery(ctx, tx, prUrl)
			if err != nil {
				pkg.Log.Error(c, "Could not delete solution", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
	default:
		pkg.Log.Warn(c, "Will not handle events other than PR opening and merging")
		c.AbortWithStatus(http.StatusOK)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		pkg.Log.Fatal(c, "Could not commit transaction", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	pkg.Log.Success(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "Pull-request event handled successfully",
	})
}
