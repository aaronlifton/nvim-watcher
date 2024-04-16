/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"os"
	"os/exec"
	"slices"

	// "os"
	// "os/exec"

	"github.com/aaronlifton/nvim-watcher/log"

	// kill "github.com/jesseduffield/kill"
	gps "github.com/mitchellh/go-ps"

	// "github.com/shirou/gopsutil/v3/cpu"
	// "github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	ps "github.com/shirou/gopsutil/v3/process"

	// "github.com/shirou/gopsutil/v3/net"
	"github.com/spf13/cobra"
)

func NewWatchProcessesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch-processes",
		Short: "Supervise processes left behind by AI plugins like ChatGPT, CodeGPT, TabNine, Codeium, and Copilot.",
		Run: func(cmd *cobra.Command, args []string) {
			// RunWatchProcesses()
			Test()
		},
	}
}

type WrappedProcess struct {
	gpsProcess gps.Process
	psProcess  *ps.Process
}

type ProcessData struct {
	Pid           int
	PPid          int
	Exe           string
	Name          string
	Memory        uint64 `json:"rss"`
	CpuAffinity   []int32
	PercentMemory float32
	PercentCpu    float64
	// Pid           int32
	// PPid          int32
	// Exe           string
	// Memory        uint64
	// Cpu           float64
	// PercentMemory float32
	// PercentCpu    float64
}
type ProcessManager interface {
	Kill(cmd *exec.Cmd)
}

type ProcessWatcher struct {
	wrappedProcesses []*WrappedProcess
}

func (p *WrappedProcess) Kill() {
	osProcess := os.Process{Pid: p.gpsProcess.Pid()}
	cmd := &exec.Cmd{Process: &osProcess}
	Kill(cmd)
}

func (p *WrappedProcess) GetStats() (ProcessData, error) {
	exe := p.gpsProcess.Executable()
	name, err := p.psProcess.Name()
	if err != nil {
		return ProcessData{}, err
	}
	memPercent, err := p.psProcess.MemoryPercent()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return ProcessData{}, err
	}
	cpuPercent, err := p.psProcess.CPUPercent()
	cpuAffinity, err := p.psProcess.CPUAffinity()
	memInfo, err := p.psProcess.MemoryInfo()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return ProcessData{}, err
	}
	return ProcessData{
		Pid:           p.gpsProcess.Pid(),
		PPid:          p.gpsProcess.PPid(),
		Exe:           exe,
		Name:          name,
		Memory:        memInfo.RSS,
		CpuAffinity:   cpuAffinity,
		PercentMemory: memPercent,
		PercentCpu:    cpuPercent,
	}, nil
}

func (pm *ProcessWatcher) GetProcesses() ([]gps.Process, error) {
	return gps.Processes()
}

var Executables = []string{"nvim", "TabNine", "codeium", "Copilot", "sourcery", "biomesyncd", "biome"}

func Test() {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get memory: %v", err)
	}
	log.ConsoleLogger.Infof("Memory: %v", memInfo.UsedPercent)
}

func PrintProcesses(procs []*ps.Process) {
	for _, proc := range procs {
		log.ConsoleLogger.Infof("Process: %v", proc)
		name, err := proc.Name()
		if err != nil {
			log.ConsoleLogger.Fatalf("Failed to get process name: %v", err)
		}
		log.ConsoleLogger.Infof("Process: %v", name)
	}
}

func RunWatchProcesses() []*WrappedProcess {
	log.Init()
	// processList, err := gps.Processes()
	// readlink /proc/<pid>/exe

	pm := &ProcessWatcher{}
	relevantProcesses := make([]gps.Process, 2)
	processList, err := pm.GetProcesses()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}

	for _, process := range processList {
		exe := process.Executable()
		if slices.Contains(Executables, exe) {
			relevantProcesses = append(relevantProcesses, process)
		}
	}
	log.ConsoleLogger.Infof("Relevant processes: %v", relevantProcesses)
	if len(relevantProcesses) == 0 {
		log.ConsoleLogger.Fatal("No relevant processes found")
		return []*WrappedProcess{}
	}
	// wrappedProcesses := make([]*ps.Process, len(relevantProcesses))
	wrappedProcesses := make([]*WrappedProcess, len(relevantProcesses))
	for _, process := range relevantProcesses {
		if process == nil {
			continue
		}
		log.ConsoleLogger.Infof("Process: %v", process)
		pid := process.Pid()
		proc, err := ps.NewProcess(int32(pid))
		if err != nil {
			log.ConsoleLogger.Fatalf("Failed to get process: %v", err)
			return []*WrappedProcess{}
		}
		wrappedProcess := WrappedProcess{
			psProcess:  proc,
			gpsProcess: process,
		}
		wrappedProcesses = append(wrappedProcesses, &wrappedProcess)
	}
	// PrintProcesses(wrappedProcesses)
	return wrappedProcesses
}

type CleanupResult struct {
	cleanedProcesses []gps.Process
	status           int
}

// func CleanupProcesses(processes []gps.Process) CleanupResult {
// 	processResults := make(map[int]gps.Process)
// 	for _, process := range processes {
// 		log.FileLogger.Infof("Process: %v", log.ProcessAction{ActionType: "kill", Process: &process})
// 	}
// 	result := []gps.Process{}
// 	return &CleanupResult{
// 		cleanedProcesses: result,
// 		status:           1,
// 	}
// }

// watchProcesses represents the watchAis command
// // map ages
// tabNineProcesses := make(map[int]gps.Process)
// nvimProcesses := make(map[int]gps.Process)
// for x := range processList {
// 	var process gps.Process = processList[x]
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
// 		var process gps.Process = tabNineProcesses[x]
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
// 		var process gps.Process = nvimProcesses[x]
// 		log.Printf("%d\t%s\n", process.Pid(), process.Executable())
// 	}
//
// 	// kill
// 	for x := range nvimProcesses {
// 		var process gps.Process = nvimProcesses[x]
// 		var osProcess os.Process = os.Process{Pid: process.Pid()}
//zydocker 		// use std library to kill
// 		cmd := exec.Cmd{Process: &osProcess}
// 		err := kill.Kill(&cmd)
// 		if err != nil {
// 			log.Printf("killed %d\t%s\n", process.Pid(), process.Executable())
// 		}
// 	}// },
