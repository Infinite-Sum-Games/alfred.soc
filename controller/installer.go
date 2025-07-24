package controller

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	"github.com/IAmRiteshKoushik/alfred/db"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
	"github.com/jackc/pgx/v5/pgtype"
)

func isValidSignature(signatureHeader string, payload, secret []byte) bool {
	if len(signatureHeader) < 7 || signatureHeader[:7] != "sha256=" {
		return false
	}
	actualSign := signatureHeader[7:]

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expectedMac := mac.Sum(nil)

	return hmac.Equal([]byte(actualSign), []byte(hex.EncodeToString(expectedMac)))
}

func InstallationHandler(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pkg.Log.Error(c, "Error reading request body during DevPool installation: %v",
			err,
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(payload))

	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		pkg.Log.Error(c, "Installation failed due to missing verification signature",
			fmt.Errorf("Missing X-Hub-Signature-256 header"),
		)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if !isValidSignature(signature, payload, []byte(cmd.AppConfig.WebhookSecret)) {
		pkg.Log.Error(c, "Invalid webhook signature",
			fmt.Errorf("Could not validate webhook signature"),
		)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		pkg.Log.Error(c, "Event type does not exist",
			fmt.Errorf("X-GitHub-Event header is missing"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	parsedPayload, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		pkg.Log.Error(c, "Failed to parse installation payload", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Extracting installation ID and populating database
	installationEvent, ok := parsedPayload.(*github.InstallationEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to parse installation payload", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	installationId := *installationEvent.Installation.ID

	q := db.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Acquire the transaction to insert all the repositories
	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	for _, repo := range installationEvent.Repositories {
		repoURL := *repo.HTMLURL

		_, err := q.VerifyRepositoryQuery(ctx, tx, db.VerifyRepositoryQueryParams{
			InstallationID: pgtype.Int8{Int64: installationId, Valid: true},
			Url:            repoURL,
		})
		if err != nil {
			pkg.Log.Error(c, "Failed to verify repository: "+repoURL, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		pkg.Log.Fatal(c, "Failed to commit transaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to mark repositories post-installation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Installation successful",
	})
	pkg.Log.Info(c, "Successfully installed DevPool")
}
