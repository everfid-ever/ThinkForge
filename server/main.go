package main

import (
	"github.com/gogf/gf/v2/os/gctx"

	"github.com/everfid-ever/ThinkForge/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
