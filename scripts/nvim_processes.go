/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package main

import (
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/aaronlifton/nvim-watcher/log"
	ps "github.com/shirou/gopsutil/v3/process"
)

func main() {
	log.Init()
	log.FileLogger.Info("Running task: nvim_processes")

	GetNvimChildren()
}

var (
	Executables    = []string{"nvim", "language_server_arm_macos", "copilot", "sourcery", "biomesyncd", "biomed"}
	PartialMatches = []string{"lsp", "biome", "rubocop", "codeium", "sourcery", "TabNine", "Copilot"}
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

	children := []*ps.Process{}
	parents := []*ps.Process{}
	tree := make(map[*ps.Process][]*ps.Process)
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
			children = append(children, process)
			parents = append(parents, parent)
			if tree[parent] == nil {
				tree[parent] = []*ps.Process{process}
			} else {
				tree[parent] = append(tree[parent], process)
			}
		}
	}
	log.ConsoleLogger.Info("Child processes:\n")
	for _, process := range children {
		name, _ := process.Name()
		log.ConsoleLogger.Infof("Child: %s", name)
	}
	user, err := user.Current()
	home := ""
	if err == nil {
		home = user.HomeDir
	}
	childrenMaxNameLen := maxNameLen(children)
	parentsMaxNameLen := maxNameLen(parents)
	log.ConsoleLogger.Info("\n\nTree:\n")
	for parent, children := range tree {
		parentName, err := parent.Name()
		if err != nil {
			parentName = "unknown"
		}
		exe, _ := parent.Exe()
		exe = strings.Replace(exe, home, "~", 1)
		log.ConsoleLogger.Infof(
			"Parent: %s (%d)%s%s",
			parentName,
			parent.Pid,
			strings.Repeat(" ", parentsMaxNameLen-len(parentName)+len(strconv.Itoa(int(parent.Pid)))+2),
			exe,
		)
		for _, child := range children {
			name, _ := child.Name()
			exe, _ := child.Exe()
			exe = strings.Replace(exe, home, "~", 1)
			log.ConsoleLogger.Infof(
				"\tChild: %s%s%s",
				name,
				strings.Repeat(" ", childrenMaxNameLen-len(name)+1),
				exe,
			)
		}
	}
}

func maxNameLen(processes []*ps.Process) int {
	nameLens := []int{}
	for _, proc := range processes {
		nameLen := 0
		name, err := proc.Name()
		if err == nil {
			nameLen = len(name)
		}
		nameLens = append(nameLens, nameLen)
	}
	return slices.Max(nameLens)
}
