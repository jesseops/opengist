package gist

import (
	"strings"

	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/git"
	"github.com/thomiceli/opengist/internal/web/context"
	"github.com/thomiceli/opengist/internal/web/handlers"
)

func Revisions(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)
	userName := gist.User.Username
	gistName := gist.Identifier()

	pageInt := handlers.GetPage(ctx)

	ignoreWhitespace := ctx.QueryParam("w") != "0"

	commits, err := gist.Log((pageInt-1)*10, git.DiffOptions{
		IgnoreWhitespace: ignoreWhitespace,
	})
	if err != nil {
		return ctx.ErrorRes(500, "Error fetching commits log", err)
	}

	var paginationParams *handlers.PaginationParams
	if !ignoreWhitespace {
		paginationParams = &handlers.PaginationParams{W: "0"}
	}

	if err := handlers.Paginate(ctx, commits, pageInt, 10, "commits", userName+"/"+gistName+"/revisions", 2, paginationParams); err != nil {
		return ctx.ErrorRes(404, ctx.Tr("error.page-not-found"), nil)
	}

	emailsSet := map[string]struct{}{}
	for _, commit := range commits {
		if commit.AuthorEmail == "" {
			continue
		}
		emailsSet[strings.ToLower(commit.AuthorEmail)] = struct{}{}
	}

	emailsUsers, err := db.GetUsersFromEmails(emailsSet)
	if err != nil {
		return ctx.ErrorRes(500, "Error fetching users emails", err)
	}

	ctx.SetData("page", "revisions")
	ctx.SetData("revision", "HEAD")
	ctx.SetData("emails", emailsUsers)
	ctx.SetData("ignoreWhitespace", ignoreWhitespace)
	ctx.SetData("htmlTitle", ctx.TrH("gist.revision-of", gist.Title))

	return ctx.Html("revisions.html")
}
