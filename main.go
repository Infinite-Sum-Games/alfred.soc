package main

import (
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/IAmRiteshKoushik/alfred/bootstrap"
	"github.com/IAmRiteshKoushik/alfred/cmd"
	"github.com/IAmRiteshKoushik/alfred/controller"
	"github.com/IAmRiteshKoushik/alfred/middleware"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	failMsg := "[CRASH]: Could not start the app due to %s"

	// Setup environment variables
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
	pkg.Log = pkg.NewLoggerService(cmd.AppConfig.Environment, f)
	pkg.Log.SetupInfo("[ACTIVE]: Logging service is online.")

	// Setup connection pooling for Postgres
	pool, err := cmd.InitDB()
	if err != nil {
		pkg.Log.SetupFail("[CRASH]: Could not initialize database pool", err)
		return
	}
	cmd.DBPool = pool
	pkg.Log.SetupInfo("[ACTIVE]: Database pool has been created.")

	// Setup valkey client
	client, err := cmd.InitValkey()
	if err != nil {
		return
	}
	pkg.Valkey = client
	pkg.Log.SetupInfo("[ACTIVE]: Valkey service is online.")

	// Bootstrap Valkey data structures
	check := bootstrap.BootstrapValkey()
	if !check {
		return
	}

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
	router.Use(pkg.TagRequestWithId)
	router.Use(middleware.PanicRecovery)
	router.Use(gin.Logger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/api/test", controller.TestEndpointHandler)
	router.POST("/api/webhook", controller.WebhookHandler)
	router.POST("/api/webhook/install", controller.InstallationHandler)

	port := strconv.Itoa(cmd.AppConfig.ServerPort)
	pkg.Log.SetupInfo("[ON]: Server configured and starting on PORT:" + port)

	routerErr := router.Run(":" + port)
	if routerErr != nil {
		pkg.Log.SetupFail("[FAIL]: Server failed to start due to:", err)
		panic(routerErr)
	}
	pkg.Log.SetupInfo("[DEACTIVE]: Server offline.")

	cmd.CloseValkey(pkg.Valkey)
	pkg.Log.SetupInfo("[DEACTIVE]: Valkey offline.")
	pkg.Log.SetupInfo("[DEACTIVE]: Logging service offline.")
}
