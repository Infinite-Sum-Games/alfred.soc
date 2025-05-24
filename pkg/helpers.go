package pkg

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/ksuid"
)

func TagRequestWithId(c *gin.Context) {
	id := ksuid.New()
	c.Set("request_id", id.String())
	c.Next()
}

func GrabRequestId(c *gin.Context) string {
	reqId, ok := c.Get("request_id")
	if !ok {
		return "missing-id"
	}
	return fmt.Sprintf("%v", reqId)
}

func GrabUsername(c *gin.Context) string {
	// TODO:
	// Utility function to grab username from the request and re-use it
	// in the request or in error logs. Helpful in detecting spammer accounts
	return ""
}
