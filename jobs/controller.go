package jobs

import (
	"net/http"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
)

func TestEndpointHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server is LIVE",
	})
	pkg.Log.Success(c)
	return
}

func WebhookHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"messsage": "Webhook event handled successfully",
	})
	pkg.Log.Success(c)
	return
}
