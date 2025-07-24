package controller

import (
	"fmt"
	"net/http"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
)

func handlePingEvent(c *gin.Context, payload any) {

	_, ok := payload.(*github.PingEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to parse PING event",
			fmt.Errorf("Malformed PING event payload received"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	pkg.Log.Info(c, "Repository successfully onboarded")
	c.JSON(http.StatusOK, gin.H{
		"message": "PING processed successfully",
	})
}
