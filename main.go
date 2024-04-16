package main

import (
	"fmt"
	"os"

	"github.com/aaronlifton/nvim-watcher/cmd"
	"github.com/aaronlifton/nvim-watcher/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "NvimSupervisor",
		// Short: "Supervise processes left behind by AI plugins like ChatGPT, CodeGPT, TabNine, Codeium, and Copilot.",
		Usage: "Clean up nvim and AI ghost processes that are slowing down your computer. Never accept a slow computer again.",
		Action: func(*cli.Context) error {
			fmt.Println("boom! I say!")
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
