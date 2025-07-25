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
	"github.com/IAmRiteshKoushik/alfred/db"
	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v74/github"
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
	Extend

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

	ok, err = q.ParticipantExistsQuery(ctx, conn, username)
	if err != nil {
		return Commentator(UnknownUser), err
	}

	return Commentator(Participant), nil
}

// Claims and unclaims
type IssueAction struct {
	ParticipantUsername string `json:"github_username"`
	Url                 string `json:"url"`
	Claimed             bool   `json:"claimed"`
	Extend              bool   `json:"extend"`
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

// Super struct to enforce polymorphism
type AllowedComment struct {
	i IssueAction  `json:"issue_action"`
	b BountyAction `json:"bounty_action"`
	a Achievement  `json:"achivement"`
}

func parseComment(cm string, by Commentator, username string,
	url string) (Comment, AllowedComment, error) {

	cm = strings.TrimSpace(cm)
	if by == Participant {
		if strings.HasPrefix(cm, "/assign") {
			data := marshalAssign(username, url)
			return Comment(Assign), AllowedComment{i: data}, nil
		} else if strings.HasPrefix(cm, "/unassign") {
			data := marshalUnassign(username, url)
			return Comment(Unassign), AllowedComment{i: data}, nil
		} else {
			return Comment(NoAction), AllowedComment{}, nil
		}

	} else if by == Maintainer {
		if strings.HasPrefix(cm, "/bounty") {
			comment := strings.Split(cm, " ")
			if len(comment) != 3 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			amt, err := strconv.Atoi(comment[1])
			if err != nil {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAmt(comment[2], amt, "bounty", url)
			return Comment(BountyComment), AllowedComment{b: data}, nil
		} else if strings.HasPrefix(cm, "/penalty") {
			comment := strings.Split(cm, " ")
			if len(comment) != 3 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			amt, err := strconv.Atoi(comment[1])
			if err != nil {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAmt(comment[2], amt, "penalty", url)
			return Comment(PenaltyComment), AllowedComment{b: data}, nil
		} else if strings.HasPrefix(cm, "/help") {
			comment := strings.Split(cm, " ")
			if len(comment) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAchievement(comment[1], "help", url)
			return Comment(HelpComment), AllowedComment{a: data}, nil
		} else if strings.HasPrefix(cm, "/doc") {
			comment := strings.Split(cm, " ")
			if len(comment) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAchievement(comment[1], "doc", url)
			return Comment(DocComment), AllowedComment{a: data}, nil
		} else if strings.HasPrefix(cm, "/test") {
			comment := strings.Split(cm, " ")
			if len(comment) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAchievement(comment[1], "test", url)
			return Comment(TestComment), AllowedComment{a: data}, nil
		} else if strings.HasPrefix(cm, "/impact") {
			comment := strings.Split(cm, " ")
			if len(comment) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAchievement(comment[1], "impact", url)
			return Comment(ImpactComment), AllowedComment{a: data}, nil
		} else if strings.HasPrefix(cm, "/extend") {
			comment := strings.Split(cm, " ")
			if len(comment) != 3 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			days, err := strconv.Atoi(comment[1])
			if err != nil {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data, err := handleExtension(comment[2][1:], days, url)
			if err != nil {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Failed to grant extension: %w", err)
			}
			return Comment(Extend), AllowedComment{i: data}, nil
		} else if strings.HasPrefix(cm, "/bug") {
			comment := strings.Split(cm, " ")
			if len(comment) != 2 {
				return Comment(NoAction), AllowedComment{}, fmt.Errorf("Invalid comment syntax")
			}
			data := marshalAchievement(username, "bug", url)
			return Comment(BugReport), AllowedComment{a: data}, nil
		} else {
			return Comment(NoAction), AllowedComment{}, nil
		}
	}
	// Neither maintainer nor participant, do not process
	return Comment(NoAction), AllowedComment{}, nil
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
	if action == NoAction {
		pkg.Log.Info(c, "No action is being performed for issue comment")
		c.AbortWithStatus(http.StatusOK)
		return

		// bounties and penalties
	} else if action == BountyComment || action == PenaltyComment {
		jsonData, err := json.Marshal(result.b)
		if err != nil {
			pkg.Log.Error(c, "Failed to marshal payload", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		err = cmd.AddToStream(pkg.Valkey, pkg.Bounty, string(jsonData))
		if err != nil {
			pkg.Log.Error(c, "Failed to insert into Redis", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// achivements
	} else if action == BugReport || action == DocComment ||
		action == HelpComment || action == TestComment || action == ImpactComment {

		jsonData, err := json.Marshal(result.a)
		if err != nil {
			pkg.Log.Error(c, "Failed to marshal payload", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		err = cmd.AddToStream(pkg.Valkey, pkg.AutomaticEvents, string(jsonData))
		if err != nil {
			pkg.Log.Error(c, "Failed to insert into Redis", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// assignment
	} else if action == Assign {
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

		// unassignment
	} else if action == Unassign {
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
		// extension
	} else if action == Extend {
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
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Issue-Comment Event handled successfully",
	})
	pkg.Log.Success(c)
}
