package main

import (
	"os"

	"github.com/aaronlifton/nvim-watcher/cmd"
	"github.com/aaronlifton/nvim-watcher/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "NvimSupervisor",
		Usage: "Supervise nvim and AI ghost processes that are slowing down your computer. Never accept a slow computer again.",
		Description: `Visualize memory and CPU usage by Neovim and Neovim-related
			processes such as AI plugins including ChatGPT, Codeium, Copilot,
			Sourcegraph, and TabNine.`,
		Action: func(*cli.Context) error {
			log.Init()
			sv := cmd.NewSupervisor()
			sv.Start()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.CombinedLogger.Fatal(err)
	}
}
