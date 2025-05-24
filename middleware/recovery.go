package middleware

import (
	"fmt"
	"net/http"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
)

func PanicRecovery(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			pkg.Log.Fatal(
				c,
				"Panic recovered.",
				fmt.Errorf("%v\n", err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Oops! Something happened. Please try again later.",
			})
		}
	}()
	c.Next()
}
