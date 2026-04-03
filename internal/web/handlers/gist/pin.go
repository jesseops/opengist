package gist

import (
	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/web/context"
)

// TogglePin toggles the pinned status of a gist. Only admins can pin/unpin.
func TogglePin(ctx *context.Context) error {
	gist := ctx.GetData("gist").(*db.Gist)

	if err := gist.SetPinned(!gist.Pinned); err != nil {
		return ctx.ErrorRes(500, "Error toggling pin status", err)
	}

	if gist.Pinned {
		ctx.AddFlash(ctx.Tr("flash.gist.unpinned"), "success")
	} else {
		ctx.AddFlash(ctx.Tr("flash.gist.pinned"), "success")
	}
	return ctx.RedirectTo("/" + gist.User.Username + "/" + gist.Identifier())
}
