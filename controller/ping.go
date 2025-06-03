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

func handlePingEvent(c *gin.Context, id uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := cmd.DBPool.Acquire(ctx)
	defer conn.Release()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to connect to database",
		})
		pkg.Log.Error(c, "Database connection failed during ping event", err)
		return
	}

	queries := db.New()
	repoName, err := queries.UpdateRepositoryOnboardedQuery(c, conn, id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to mark repository as onboarded",
		})
		pkg.Log.Error(c, "Failed to update repository onboarding status", err)
		return
	}

	pkg.Log.Info(c, "Repository "+repoName+" successfully onboarded")
	c.JSON(http.StatusOK, gin.H{
		"message": "Repository successfully onboarded",
	})

}
