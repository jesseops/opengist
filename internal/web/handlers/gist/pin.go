package gist

import (
	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/web/context"
)

// TogglePin toggles the pinned status of a gist. Only admins can pin/unpin.
func TogglePin(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	newPinned := !gist.Pinned
	if err := gist.SetPinned(newPinned); err != nil {
		return ctx.ErrorRes(500, "Error toggling pin status", err)
	}

	if newPinned {
		ctx.AddFlash(ctx.Tr("flash.gist.pinned"), "success")
	} else {
		ctx.AddFlash(ctx.Tr("flash.gist.unpinned"), "success")
	}
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier())
}
