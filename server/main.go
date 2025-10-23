package server

import (
	"github.com/gogf/gf/v2/os/gctx"

	"github.com/everfid-ever/ThinkForge/internal/cmd"
)

func Main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
