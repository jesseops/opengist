package gist

import (
	"strings"

	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/web/context"
)

// Collaborators renders the collaborators management page for a gist.
func Collaborators(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	collaborators, err := db.GetCollaboratorsForGist(gist.ID)
	if err != nil {
		return ctx.ErrorRes(500, "Error fetching collaborators", err)
	}

	ctx.SetData("collaborators", collaborators)
	ctx.SetData("htmlTitle", ctx.TrH("gist.collaborators.title", gist.Title))
	return ctx.Html("collaborators.html")
}

// AddCollaborator adds a collaborator to a gist (owner-only).
func AddCollaborator(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	username := strings.TrimSpace(ctx.FormValue("username"))
	if username == "" {
		ctx.AddFlash(ctx.Tr("flash.gist.collaborator-username-required"), "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Look up the user
	user, err := db.GetUserByUsername(username)
	if err != nil {
		ctx.AddFlash(ctx.Tr("flash.gist.collaborator-user-not-found", username), "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Can't add yourself
	if user.ID == gist.UserID {
		ctx.AddFlash(ctx.Tr("flash.gist.collaborator-is-owner"), "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Check if already a collaborator
	isCollab, err := db.IsCollaborator(gist.ID, user.ID)
	if err != nil {
		return ctx.ErrorRes(500, "Error checking collaborator status", err)
	}
	if isCollab {
		ctx.AddFlash(ctx.Tr("flash.gist.collaborator-exists", username), "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	if err := db.AddCollaborator(gist.ID, user.ID); err != nil {
		return ctx.ErrorRes(500, "Error adding collaborator", err)
	}

	gist.AddInIndex()
	ctx.AddFlash(ctx.Tr("flash.gist.collaborator-added", username), "success")
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
}

// RemoveCollaborator removes a collaborator from a gist (owner-only).
func RemoveCollaborator(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	username := strings.TrimSpace(ctx.FormValue("username"))
	if username == "" {
		return ctx.ErrorRes(400, ctx.Tr("flash.gist.collaborator-username-required"), nil)
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		ctx.AddFlash(ctx.Tr("flash.gist.collaborator-user-not-found", username), "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	if err := db.RemoveCollaborator(gist.ID, user.ID); err != nil {
		return ctx.ErrorRes(500, "Error removing collaborator", err)
	}

	gist.AddInIndex()
	ctx.AddFlash(ctx.Tr("flash.gist.collaborator-removed", username), "success")
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
}

// LeaveCollaboration lets a collaborator remove themselves from a gist.
func LeaveCollaboration(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)
	gistUrl := "/" + gist.User.Username + "/" + gist.Identifier()

	isCollab, err := db.IsCollaborator(gist.ID, ctx.User.ID)
	if err != nil {
		return ctx.ErrorRes(500, "Error checking collaborator status", err)
	}
	if !isCollab {
		return ctx.RedirectTo(gistUrl)
	}

	if err := db.RemoveCollaborator(gist.ID, ctx.User.ID); err != nil {
		return ctx.ErrorRes(500, "Error leaving gist", err)
	}

	gist.AddInIndex()
	ctx.AddFlash(ctx.Tr("flash.gist.collaborator-left"), "success")

	// a private gist is no longer accessible once the user has left it
	if gist.Private == db.PrivateVisibility {
		return ctx.RedirectTo("/all")
	}
	return ctx.RedirectTo(gistUrl)
}

// SearchUsersApi returns a JSON list of users matching a query, for autocomplete.
func SearchUsersApi(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)
	query := strings.TrimSpace(ctx.QueryParam("q"))

	if query == "" {
		return ctx.JSON(200, []map[string]string{})
	}

	users, err := db.SearchUsers(strings.ToLower(query), gist.ID, gist.UserID, 10)
	if err != nil {
		return ctx.ErrorRes(500, "Error searching users", err)
	}

	results := make([]map[string]string, 0, len(users))
	for _, u := range users {
		results = append(results, map[string]string{
			"username": u.Username,
		})
	}

	return ctx.JSON(200, results)
}
