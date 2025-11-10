package main

import (
	"github.com/gogf/gf/v2/os/gctx"

	"github.com/everfid-ever/ThinkForge/internal/cmd"
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
