package main

import (
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	"github.com/IAmRiteshKoushik/alfred/jobs"
	"github.com/IAmRiteshKoushik/alfred/middleware"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	failMsg := "[CRASH]: Could not start the app due to %s"

	// Setup environment
	err := cmd.SetupEnv()
	if err != nil {
		log.Printf(failMsg, err)
		return
	}

	// Setup logger
	f, err := os.OpenFile("app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf(failMsg, err)
	}
	defer f.Close()
	pkg.Log = pkg.NewLoggerService(cmd.EnvVars.Environment, f)
	pkg.Log.SetupInfo("[ON]: Logging service has been activated.")

	// Setup valkey client
	client, err := cmd.SetupValkey()
	if err != nil {
		return
	}
	pkg.Valkey = client
	pkg.Log.SetupInfo("[ON]: Valkey service has been activated.")

	// Setup gin server
	ginLogs, err := os.Create("gin.log")
	if err != nil {
		pkg.Log.SetupFail("Error creating log file for Gin", err)
		return
	}
	defer ginLogs.Close()
	multiWriter := io.MultiWriter(os.Stdout, ginLogs)
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(middleware.PanicRecovery)
	router.Use(gin.Logger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/test", jobs.TestEndpointHandler)
	router.POST("/webhook/:id", jobs.WebhookHandler)

	port := strconv.Itoa(cmd.EnvVars.ServerPort)
	pkg.Log.SetupInfo("[ON]: Server configured and starting on PORT:" + port)
	routerErr := router.Run(":" + port)
	if routerErr != nil {
		pkg.Log.SetupFail("[CRASH]: Server failed to start due to:", err)
		panic(routerErr)
	}

	pkg.Log.SetupInfo("[OFF]: Server deactivated.")
	pkg.Log.SetupInfo("[OFF]: Valkey deactivated.")
	pkg.Log.SetupInfo("[OFF]: Logging service deactivated.")
}
