/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"slices"
	// "os"
	// "os/exec"

	"github.com/aaronlifton/nvim-watcher/log"

	// kill "github.com/jesseduffield/kill"
	ps "github.com/mitchellh/go-ps"
	"github.com/spf13/cobra"
)

func NewWatchProcessesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch-processes",
		Short: "Supervise processes left behind by AI plugins like ChatGPT, CodeGPT, TabNine, Codeium, and Copilot.",
		Run: func(cmd *cobra.Command, args []string) {
			RunWatchProcesses()
		},
	}
}

type ProcessManager struct {
	Calls int
}

func (pm *ProcessManager) GetProcesses() ([]ps.Process, error) {
	pm.Calls++
	processList, err := ps.Processes()
	return processList, err
}

var Executables = []string{"nvim", "TabNine", "Codeium", "Copilot"}

func RunWatchProcesses() []ps.Process {
	log.Init()
	// processList, err := ps.Processes()
	pm := &ProcessManager{}
	relevantProcesses := make([]ps.Process, 2)
	processList, err := pm.GetProcesses()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}

	for _, process := range processList {
		log.FileLogger.Infof("Process: %v", log.ProcessAction{ActionType: "list", Process: &process})
		exe := process.Executable()
		if slices.Contains(Executables, exe) {
			log.ConsoleLogger.Infof("%d\t%s\n", process.Pid(), process.Executable())
			relevantProcesses = append(relevantProcesses, process)
		}
	}
	// for _, process := range processList {
	// 	osProcess := os.Process{Pid: process.Pid()}
	// 	cmd := exec.Cmd{Process: &osProcess}
	// 	err := kill.Kill(&cmd)
	// 	if err != nil {
	// 		log.CombinedGitLogger.Fatal("kill %s failed: %s", process.Executable(), err)
	// 	}
	// 	// cmd := exec.Command("pkill", "-9", process.Executable())
	// 	// log.ConsoleLogger.Info("pkill -9 %s failed: %s\n", process.Executable(), err)
	// }
	return relevantProcesses
}

// watchProcesses represents the watchAis command
// // map ages
// tabNineProcesses := make(map[int]ps.Process)
// nvimProcesses := make(map[int]ps.Process)
// for x := range processList {
// 	var process ps.Process = processList[x]
// 	log.Printf("%d\t%s\n", process.Pid(), process.Executable())
// 	if process.Executable() == "nvim" {
// 		nvimProcesses[process.Pid()] = process
// 	}
// 	if process.Executable() == "TabNine" {
// 		tabNineProcesses[process.Pid()] = process
// 	}
// 	// do os.* stuff on the pid
// }
//
// if len(nvimProcesses) == 0 {
// 	log.Printf("No nvim processes found\n")
// 	return
// }
// if len(tabNineProcesses) > 2 {
// 	// exec pkill
// 	for x := range tabNineProcesses {
// 		var process ps.Process = tabNineProcesses[x]
// 		log.Printf("%d\t%s\n", process.Pid(), process.Executable())
// 		cmd := exec.Command("pkill", "-9", process.Executable())
// 		cmd.Stdout = os.Stdout
// 		cmd.Stderr = os.Stderr
// 		err := cmd.Run()
// 		if err != nil {
// 			log.Printf("pkill -9 %s failed: %s\n", process.Executable(), err)
// 		}
// 	}
//
// 	log.Printf("Filtered\n")
// 	for x := range nvimProcesses {
// 		var process ps.Process = nvimProcesses[x]
// 		log.Printf("%d\t%s\n", process.Pid(), process.Executable())
// 	}
//
// 	// kill
// 	for x := range nvimProcesses {
// 		var process ps.Process = nvimProcesses[x]
// 		var osProcess os.Process = os.Process{Pid: process.Pid()}
// 		// use std library to kill
// 		cmd := exec.Cmd{Process: &osProcess}
// 		err := kill.Kill(&cmd)
// 		if err != nil {
// 			log.Printf("killed %d\t%s\n", process.Pid(), process.Executable())
// 		}
// 	}
// },
