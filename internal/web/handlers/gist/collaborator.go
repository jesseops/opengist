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
	ctx.SetData("htmlTitle", "Collaborators - "+gist.Title)
	return ctx.Html("collaborators.html")
}

// AddCollaborator adds a collaborator to a gist (owner-only).
func AddCollaborator(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	username := strings.TrimSpace(ctx.FormValue("username"))
	if username == "" {
		ctx.AddFlash("Please enter a username", "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Look up the user
	user, err := db.GetUserByUsername(username)
	if err != nil {
		ctx.AddFlash("User not found: "+username, "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Can't add yourself
	if user.ID == gist.UserID {
		ctx.AddFlash("You can't add yourself as a collaborator", "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	// Check if already a collaborator
	isCollab, err := db.IsCollaborator(gist.ID, user.ID)
	if err != nil {
		return ctx.ErrorRes(500, "Error checking collaborator status", err)
	}
	if isCollab {
		ctx.AddFlash(username+" is already a collaborator", "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	if err := db.AddCollaborator(gist.ID, user.ID); err != nil {
		return ctx.ErrorRes(500, "Error adding collaborator", err)
	}

	ctx.AddFlash(username+" has been added as a collaborator", "success")
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
}

// RemoveCollaborator removes a collaborator from a gist (owner-only).
func RemoveCollaborator(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	username := strings.TrimSpace(ctx.FormValue("username"))
	if username == "" {
		return ctx.ErrorRes(400, "Username is required", nil)
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		ctx.AddFlash("User not found: "+username, "error")
		return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
	}

	if err := db.RemoveCollaborator(gist.ID, user.ID); err != nil {
		return ctx.ErrorRes(500, "Error removing collaborator", err)
	}

	ctx.AddFlash(username+" has been removed as a collaborator", "success")
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier() + "/collaborators")
}

// SearchUsersApi returns a JSON list of users matching a query, for autocomplete.
func SearchUsersApi(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)
	query := strings.TrimSpace(ctx.QueryParam("q"))

	if query == "" {
		return ctx.JSON(200, []map[string]string{})
	}

	users, err := db.SearchUsers(strings.ToLower(query), gist.UserID, 10)
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
