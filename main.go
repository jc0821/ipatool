package main

import (
	"os"

	"github.com/majd/ipatool/v2/cmd"
)

func main() {
	// [新增] 如果用户直接双击运行程序（不带任何参数），自动为其追加 "web" 命令
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "web")
	}
	os.Exit(cmd.Execute())
}
