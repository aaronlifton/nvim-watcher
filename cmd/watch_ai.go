/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	kill "github.com/jesseduffield/kill"
	ps "github.com/mitchellh/go-ps"
	"github.com/spf13/cobra"
)

// watchAisCmd represents the watchAis command
var watchAisCmd = &cobra.Command{
	Use:   "watch-ai",
	Short: "Supervise AI plugins like ChatGPT CodeGPT, TabNine, Codeium, and Copilot.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("watchAis called")
		processList, err := ps.Processes()
		if err != nil {
			log.Println("ps.Processes() Failed, are you using windows?")
			return
		}

		// map ages
		tabNineProcesses := make(map[int]ps.Process)
		nvimProcesses := make(map[int]ps.Process)
		for x := range processList {
			var process ps.Process = processList[x]
			log.Printf("%d\t%s\n", process.Pid(), process.Executable())
			if process.Executable() == "nvim" {
				nvimProcesses[process.Pid()] = process
			}
			if process.Executable() == "TabNine" {
				tabNineProcesses[process.Pid()] = process
			}
			// do os.* stuff on the pid
		}

		if len(nvimProcesses) == 0 {
			log.Printf("No nvim processes found\n")
			return
		}
		if len(tabNineProcesses) > 2 {
			// exec pkill
			for x := range tabNineProcesses {
				var process ps.Process = tabNineProcesses[x]
				log.Printf("%d\t%s\n", process.Pid(), process.Executable())
				cmd := exec.Command("pkill", "-9", process.Executable())
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					log.Printf("pkill -9 %s failed: %s\n", process.Executable(), err)
				}
			}

			log.Printf("Filtered\n")
			for x := range nvimProcesses {
				var process ps.Process = nvimProcesses[x]
				log.Printf("%d\t%s\n", process.Pid(), process.Executable())
			}

			// kill
			for x := range nvimProcesses {
				var process ps.Process = nvimProcesses[x]
				var osProcess os.Process = os.Process{Pid: process.Pid()}
				// use std library to kill
				cmd := exec.Cmd{Process: &osProcess}
				err := kill.Kill(&cmd)
				if err != nil {
					log.Printf("killed %d\t%s\n", process.Pid(), process.Executable())
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(watchAisCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// watchAisCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// watchAisCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
