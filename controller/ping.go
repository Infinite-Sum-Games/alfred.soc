package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	db "github.com/IAmRiteshKoushik/alfred/db/gen"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
)

func handlePingEvent(c *gin.Context, incoming any) {

	payload, ok := incoming.(*github.PingEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to parse PING event",
			fmt.Errorf("Malformed PING event payload received"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	repoUrl := *payload.Repo.HTMLURL

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		pkg.Log.Error(c, "Failed to begin transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to set repository display status",
		})
		return
	}
	defer tx.Rollback(ctx)

	q := db.New()
	repoName, err := q.UpdateRepositoryOnDisplayQuery(ctx, tx, repoUrl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to set repository display status",
		})
		pkg.Log.Error(c, "Failed to set repository display status", err)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to commit transaction",
		})
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		return
	}

	pkg.Log.Info(c, "Successfully updated repository display status for: "+repoName)
	c.JSON(http.StatusOK, gin.H{
		"message": "PING processed successfully",
	})
}
