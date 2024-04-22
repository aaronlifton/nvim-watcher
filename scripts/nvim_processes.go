/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package main

import (

	// "os"
	// "os/exec"

	"path/filepath"
	"slices"
	"strings"

	"github.com/aaronlifton/nvim-watcher/log"

	// "github.com/shirou/gopsutil/v3/cpu"
	// "github.com/shirou/gopsutil/v3/load"
	// "github.com/shirou/gopsutil/v3/net"
	// "github.com/shirou/gopsutil/v3/process"

	// gps "github.com/mitchellh/go-ps"
	ps "github.com/shirou/gopsutil/v3/process"
)

func main() {
	log.Init()
	log.FileLogger.Info("Running task: nvim_processes")

	// GetParents()
	GetNvimChildren()
}

var (
	// Executables    = []string{"nvim", "biomesyncd", "biomed", "rubocop", , "codeium", "sourcery", "TabNine", "Copilot"}
	Executables = []string{"nvim", "language_server_arm_macos", "copilot", "sourcery"}
	// PartialMatches = []string{"lsp", "codeium", "biome", "sourcery", "TabNine", "Copilot"}
	PartialMatches = []string{"lsp", "biome"}
)

func logParents(process *ps.Process, name string) {
	parent, err := process.Parent()
	parentName, _ := parent.Name()
	pparentName := "nil"
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get parent: %v", err)
	} else {
		pparent, _ := parent.Parent()
		pparentName, _ = pparent.Name()
	}
	log.ConsoleLogger.Infof("%s -> Parent: %s -> %s", name, parentName, pparentName)
}

func GetParents() {
	processList, err := ps.Processes()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}

	for _, process := range processList {
		exe, _ := process.Exe()
		name := filepath.Base(exe)
		// log.FileLogger.Info("Checking process", zap.String("name", exe))
		if slices.Contains(Executables, name) {
			logParents(process, name)
		} else {
			for _, partialMatch := range PartialMatches {
				if strings.Contains(strings.ToLower(exe), strings.ToLower(partialMatch)) {
					logParents(process, name)
				}
			}
		}
	}

	log.ConsoleLogger.Fatal("Done")
}

func GetNvimChildren() {
	processList, err := ps.Processes()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}
	nvimProcessIds := make(map[int32]bool)
	for _, process := range processList {
		exe, _ := process.Exe()
		name := filepath.Base(exe)
		parent, _ := process.Parent()
		parentName, _ := parent.Name()
		parentIsNvim := parentName == "nvim"
		if name == "nvim" {
			nvimProcessIds[process.Pid] = parentIsNvim
		}
	}
	keys := make([]int32, 0, len(nvimProcessIds))
	for k := range nvimProcessIds {
		keys = append(keys, k)
	}
	log.ConsoleLogger.Infof("Found %d nvim processes", len(nvimProcessIds))

	// children := []*ps.Process{}
	tree := make(map[int32][]*ps.Process)
	for _, process := range processList {
		name, _ := process.Name()
		if name == "nvim" {
			continue
		}
		parent, err := process.Parent()
		if err != nil {
			continue
		}
		if slices.Contains(keys, parent.Pid) {
			// children = append(children, process)
			if tree[parent.Pid] == nil {
				tree[parent.Pid] = []*ps.Process{process}
			} else {
				tree[parent.Pid] = append(tree[parent.Pid], process)
			}
		}
	}
	// for _, process := range children {
	// 	name, _ := process.Name()
	// 	log.ConsoleLogger.Infof("Child: %s", name)
	// }
	for parent, children := range tree {
		log.ConsoleLogger.Infof("Parent: %d", parent)
		for _, child := range children {
			name, _ := child.Name()
			exe, _ := child.Exe()
			log.ConsoleLogger.Infof("\tChild: %s (%s)", name, exe)
		}
	}
}
