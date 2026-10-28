package gist

import (
	"regexp"
	"strconv"
	"time"

	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/render"
	"github.com/thomiceli/opengist/internal/web/context"
)

var validRevisionRe = regexp.MustCompile(`^[0-9a-f]{1,40}$`)

func CreateComment(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)
	user := ctx.User

	if user == nil {
		return ctx.RedirectTo("/login")
	}

	// gistInit already restricts private gists to the owner and collaborators.

	content := ctx.FormValue("content")
	if content == "" {
		ctx.AddFlash("Comment cannot be empty", "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier())
	}

	revision := ctx.FormValue("revision")
	if revision == "HEAD" {
		revision = ""
	}
	// Validate revision is empty or a valid hex hash
	if revision != "" && !validRevisionRe.MatchString(revision) {
		revision = ""
	}

	comment := &db.Comment{
		GistID:    gist.ID,
		UserID:    user.ID,
		Content:   content,
		Revision:  revision,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	if err := db.CreateComment(comment); err != nil {
		return ctx.ErrorRes(500, "Error creating comment", err)
	}

	redirectTo := "/" + gist.User.Username + "/" + gist.Identifier()
	if revision != "" {
		redirectTo += "/rev/" + revision
	}
	redirectTo += "#comments"
	return ctx.RedirectTo(redirectTo)
}

func DeleteComment(ctx *context.Context) error {
	user := ctx.User
	if user == nil {
		return ctx.RedirectTo("/login")
	}

	gist := ctx.GetData("gist").(*db.Gist)

	commentIDStr := ctx.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		return ctx.ErrorRes(400, "Invalid comment ID", err)
	}

	comment, err := db.GetCommentByID(uint(commentID))
	if err != nil {
		return ctx.ErrorRes(404, "Comment not found", err)
	}

	// Verify the comment belongs to this gist
	if comment.GistID != gist.ID {
		return ctx.ErrorRes(404, "Comment not found", nil)
	}

	// Only the comment author, the gist owner or an admin can delete
	if comment.UserID != user.ID && !gist.IsOwner(user) && !user.IsAdmin {
		return ctx.ErrorRes(403, "You cannot delete this comment", nil)
	}

	if err := db.DeleteComment(comment.ID); err != nil {
		return ctx.ErrorRes(500, "Error deleting comment", err)
	}

	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "#comments")
}

// CommentRendered holds a comment with its rendered Markdown HTML.
type CommentRendered struct {
	*db.Comment
	HTML          string
	ShortRevision string
}

// RenderComments converts raw Markdown content to HTML for display.
func RenderComments(comments []*db.Comment) []CommentRendered {
	rendered := make([]CommentRendered, len(comments))
	for i, c := range comments {
		html, _ := render.MarkdownString(c.Content)
		shortRev := c.Revision
		if len(shortRev) > 7 {
			shortRev = shortRev[:7]
		}
		rendered[i] = CommentRendered{
			Comment:       c,
			HTML:          html,
			ShortRevision: shortRev,
		}
	}
	return rendered
}
