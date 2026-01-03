package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IAmRiteshKoushik/alfred/cmd"
	db "github.com/IAmRiteshKoushik/alfred/db/gen"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
	"github.com/jackc/pgx/v5/pgtype"
)

type Commentator int
type Comment int

const (
	Participant Commentator = iota
	Maintainer
	UnknownUser
)

const (
	BountyComment Comment = iota
	PenaltyComment

	TestComment
	HelpComment
	DocComment
	ImpactComment
	BugReport

	Assign
	Unassign
	// Extend

	NoAction
)

func findCommentator(username string, repoUrl string) (Commentator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := cmd.DBPool.Acquire(ctx)
	if err != nil {
		return Commentator(UnknownUser), err
	}
	defer conn.Release()

	q := db.New()
	maintainers, err := q.GetMaintainersQuery(ctx, conn, repoUrl)
	if err != nil {
		return Commentator(UnknownUser), err
	}

	ok := slices.Contains(maintainers, username)
	if ok {
		return Commentator(Maintainer), err
	}

	ok, err = q.ParticipantExistsQuery(ctx, conn, pgtype.Text{
		String: username,
		Valid:  true,
	})
	if err != nil {
		return Commentator(UnknownUser), err
	}

	if ok {
		return Commentator(Participant), nil
	}

	return Commentator(UnknownUser), nil
}

// Claims and unclaims
type IssueAction struct {
	ParticipantUsername string `json:"github_username"`
	Url                 string `json:"url"`
	Claimed             bool   `json:"claimed"`
	// Extend              bool   `json:"extend"`
}

// Serialize the data and drop it inside Redis
func marshalAssign(username string, issueUrl string) IssueAction {
	return IssueAction{
		ParticipantUsername: username,
		Url:                 issueUrl,
		Claimed:             true,
	}
}

func marshalUnassign(username string, issueUrl string) IssueAction {
	return IssueAction{
		ParticipantUsername: username,
		Url:                 issueUrl,
		Claimed:             false,
	}
}

/*
func handleExtension(username string, days int, issueUrl string) (IssueAction, error) {
	fmt.Println(username, days, issueUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		return IssueAction{}, err
	}
	defer tx.Rollback(ctx)

	q := db.New()
	ok, err := q.ExtendClaimQuery(ctx, tx, db.ExtendClaimQueryParams{
		Ghusername: username,
		IssueUrl:   issueUrl,
		Days:       int32(days),
	})
	if err != nil || ok == "" {
		return IssueAction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return IssueAction{}, err
	}

	return IssueAction{
		ParticipantUsername: username,
		Url:                 issueUrl,
		Claimed:             false,
		Extend:              true,
	}, err
}
*/

// Bounties and penalties
type BountyAction struct {
	ParticipantUsername string `json:"github_username"`
	Amount              int    `json:"amount"`
	Url                 string `json:"url"`
	Action              string `json:"action"`
}

func marshalAmt(username string, amt int, action string, url string) BountyAction {
	return BountyAction{
		ParticipantUsername: username,
		Amount:              amt,
		Action:              action,
		Url:                 url,
	}
}

// Different badges : bug, impact, doc, test, help
type Achievement struct {
	ParticipantUsername string `json:"github_username"`
	Url                 string `json:"url"`
	Type                string `json:"type"`
}

func marshalAchievement(username string, action string, url string) Achievement {
	return Achievement{
		ParticipantUsername: username,
		Type:                action,
		Url:                 url,
	}
}

func processBountyOrPenalty(bountyData BountyAction, dispatchedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := cmd.DBPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New()

	amount := int32(bountyData.Amount)
	if bountyData.Action == "PENALTY" {
		amount = -amount
	}

	_, err = q.UpdateUserBountyQuery(ctx, tx, db.UpdateUserBountyQueryParams{
		Bounty:     amount,
		Ghusername: pgtype.Text{String: bountyData.ParticipantUsername, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update user bounty: %w", err)
	}

	err = q.AddBountyLogQuery(ctx, tx, db.AddBountyLogQueryParams{
		Ghusername:   bountyData.ParticipantUsername,
		DispatchedBy: dispatchedBy,
		ProofUrl:     bountyData.Url,
		Amount:       amount,
	})
	if err != nil {
		return fmt.Errorf("failed to add bounty log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	err = cmd.UpdateLeaderboard(pkg.Valkey, pkg.Leaderboard, bountyData.ParticipantUsername, float64(amount))
	if err != nil {
		return fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return nil
}

// Super struct to enforce polymorphism
type AllowedComment struct {
	i IssueAction  `json:"issue_action"`
	b BountyAction `json:"bounty_action"`
	a Achievement  `json:"achivement"`
}

func parseComment(cm string, by Commentator, username string,
	url string) (Comment, AllowedComment, error) {

	cm = strings.TrimSpace(cm)
	switch by {

	case Participant:
		if strings.HasPrefix(cm, "/assign") {
			data := marshalAssign(username, url)
			return Comment(Assign), AllowedComment{i: data}, nil
		} else if strings.HasPrefix(cm, "/unassign") {
			data := marshalUnassign(username, url)
			return Comment(Unassign), AllowedComment{i: data}, nil
		}

	case Maintainer:
		parts := strings.Split(cm, " ")
		if len(parts) < 2 {
			return Comment(NoAction), AllowedComment{}, nil
		}

		command := parts[0] // Contains bounty or penalty
		args := parts[1:]   // Contains [amount] [username]

		switch command {
		case "/bounty", "/penalty":
			if len(args) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax for %s", command)
			}
			amt, err := strconv.Atoi(args[0])
			if err != nil {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid amount for %s", command)
			}
			action := "BOUNTY"
			commentType := BountyComment
			if command == "/penalty" {
				action = "PENALTY"
				commentType = PenaltyComment
			}
			username := strings.TrimPrefix(args[1], "@")
			data := marshalAmt(username, amt, action, url)
			return commentType, AllowedComment{b: data}, nil
		case "/help", "/doc", "/test", "/impact", "/bug":
			if len(args) != 1 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax for %s", command)
			}
			var commentType Comment
			switch command {
			case "/help":
				commentType = HelpComment
			case "/doc":
				commentType = DocComment
			case "/test":
				commentType = TestComment
			case "/impact":
				commentType = ImpactComment
			case "/bug":
				commentType = BugReport
			}
			username := strings.TrimPrefix(args[0], "@")
			data := marshalAchievement(username, strings.ToUpper(command[1:]), url)
			return commentType, AllowedComment{a: data}, nil
		}
	}
	return Comment(NoAction), AllowedComment{}, nil
}

func sendToStream(c *gin.Context, streamName string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		pkg.Log.Error(c, "Failed to marshal payload", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return err
	}
	err = cmd.AddToStream(pkg.Valkey, streamName, string(jsonData))
	if err != nil {
		pkg.Log.Error(c, "Failed to insert into Redis", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return err
	}
	return nil
}

// This function only handles the parsed results and sends them to appropriate
// redis streams for further processing by gravemind or devpool
func handleIssueCommentEvent(c *gin.Context, payload any) {
	issueCommentEvent, ok := payload.(*github.IssueCommentEvent)
	if !ok {
		pkg.Log.Error(c, "Failed to marshal issue-comment event",
			fmt.Errorf("Malformed event payload received in Issue-Event"),
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	issueUrl := *issueCommentEvent.Issue.HTMLURL
	repoUrl := *issueCommentEvent.Repo.HTMLURL
	commentBy := *issueCommentEvent.Comment.User.Login
	commentator, err := findCommentator(commentBy, repoUrl)
	if err != nil {
		pkg.Log.Error(c, "Failed to find the commentator", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	commentBody := *issueCommentEvent.Comment.Body
	action, result, err := parseComment(commentBody, commentator, commentBy, issueUrl)
	if err != nil {
		pkg.Log.Error(c, "Failed to parse issue-comment", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// No action
	switch action {

	case NoAction:
		pkg.Log.Info(c, "No action is being performed for issue comment")
		c.AbortWithStatus(http.StatusOK)
		return

	case BountyComment, PenaltyComment:
		// DB call
		fmt.Println(result.b.ParticipantUsername)
		err := processBountyOrPenalty(result.b, commentBy)
		if err != nil {
			pkg.Log.Error(c, "Failed to process bounty/penalty", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Redis call
		if err := sendToStream(c, pkg.Bounty, result.b); err != nil {
			return
		}

	case BugReport, DocComment, HelpComment, TestComment, ImpactComment:
		if err := sendToStream(c, pkg.AutomaticEvents, result.a); err != nil {
			return
		}

	case Assign:
		// Redis Call
		if err := sendToStream(c, pkg.IssueClaim, result.i); err != nil {
			return
		}

	case Unassign:
		// Redis call
		if err := sendToStream(c, pkg.IssueClaim, result.i); err != nil {
			return
		}
		/*
			// Currently not being used - "/extend"
			case Extend:
				jsonData, err := json.Marshal(result.i)
				if err != nil {
					pkg.Log.Error(c, "Failed to marshal payload", err)
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				err = cmd.AddToStream(pkg.Valkey, pkg.IssueClaim, string(jsonData))
				if err != nil {
					pkg.Log.Error(c, "Failed to insert into Redis", err)
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
		*/
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Issue-Comment Event handled successfully",
	})
	pkg.Log.Success(c)
}
