/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"bufio"
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/aaronlifton/nvim-watcher/log"
	"go.uber.org/zap"

	"github.com/go-co-op/gocron"
	ps "github.com/shirou/gopsutil/v3/process"
)

var (
	Executables           = []string{"nvim", "language_server_arm_macos", "copilot", "sourcery", "biomesyncd", "biomed"}
	PartialMatches        = []string{"lsp", "biome", "rubocop", "codeium", "sourcery", "TabNine", "Copilot"}
	currentSort    string = "cpu"
	currentView    string = "children"
	initialized    bool   = false
	Data           chan interface{}
)

type ProcessSupervisor interface {
	Start() error
	PeriodicTask()
	GetAllProcessData() ([]WrappedProcess, error)
	GetProcesses() []*WrappedProcess
}
type SupervisorConfig struct {
	durationMinutes int
}
type Supervisor struct {
	config SupervisorConfig
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		config: SupervisorConfig{
			durationMinutes: 30,
		},
	}
}

func (s *Supervisor) Start() {
	var err error
	scheduler := gocron.NewScheduler(time.UTC)
	job, err := scheduler.Every(s.config.durationMinutes).Minutes().Do(s.PeriodicTask)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.FileLogger.Info("Starting gocron")
	log.FileLogger.Infof("Job will run next at %s", job.NextRun)
	scheduler2 := gocron.NewScheduler(time.UTC)
	job2, err := scheduler2.Every(1).Seconds().Do(s.PeriodicTask)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.FileLogger.Infof("Job will run next at %s", job2.NextRun)
	log.FileLogger.Info("Starting blocking log stream")
	go ListenForInput()
	scheduler2.StartBlocking()
}

func (s *Supervisor) StreamLog() error {
	files, err := filepath.Glob("logs/*.log")
	if err != nil {
		return err
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Supervisor) PeriodicTask() {
	log.Init()
	log.FileLogger.Info("Running task")
	ts := time.Now().Format(time.RFC3339)
	log.CronLogger.Info("Task ran at: ", ts)

	processes := s.GetRelevantProcesses()
	logProcesses(processes)

	go Draw(processes)
}

func logProcesses(ps []*WrappedProcess) {
	for _, process := range ps {
		log.FileLogger.Info("Top 10 processes",
			zap.Dict(
				"process",
				zap.String("name", process.Name),
				zap.String("exe", process.Exe),
				zap.String("memPercent", fmt.Sprintf("%f", process.PercentMemory)),
				zap.String("cpuPercent", fmt.Sprintf("%f", process.PercentCpu)),
			),
		)
	}
}

func (s *Supervisor) GetRelevantProcesses() []*WrappedProcess {
	processList, err := ps.Processes()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}
	nvimProcessIds := make(map[int32]bool)
	for _, process := range processList {
		exe, _ := process.Exe()
		name := filepath.Base(exe)
		parent, _ := process.Parent()
		if parent == nil {
			continue
		}
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

	children := []*ps.Process{}
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
			if tree[parent] == nil {
				tree[parent] = []*ps.Process{process}
			} else {
				tree[parent] = append(tree[parent], process)
			}
			children = append(children, process)
		}
	}

	parents := make([]*ps.Process, 0, len(tree))
	for parent, children := range tree {
		for _, child := range children {
			children = append(children, child)
			idx := slices.IndexFunc(parents, func(p *ps.Process) bool {
				return p.Pid == parent.Pid
			})
			if idx == -1 {
				parents = append(parents, parent)
			}
		}
	}
	var wrappedProcesses []*WrappedProcess
	var processGroup []*ps.Process
	if currentView == "parent" {
		wrappedProcesses = make([]*WrappedProcess, len(parents))
		processGroup = parents
	} else {
		wrappedProcesses = make([]*WrappedProcess, len(children))
		processGroup = children
	}
	for i, process := range processGroup {
		wp := NewWrappedProcess(process)
		wrappedProcesses[i] = wp
	}

	slices.SortFunc(wrappedProcesses, func(i, j *WrappedProcess) int {
		if currentSort == "cpu" {
			if j == nil || i == nil {
				log.ConsoleLogger.Fatal("here")
			}
			return cmp.Compare(j.PercentCpu, i.PercentCpu)
		} else {
			return cmp.Compare(j.Memory, i.Memory)
		}
	})
	return wrappedProcesses
}

func NewWrappedProcess(p *ps.Process) *WrappedProcess {
	exe, _ := p.Exe()
	name := filepath.Base(exe)
	memPercent, err := p.MemoryPercent()
	if err != nil {
		log.FileLogger.Infof("Failed to get %s.MemoryPercent: %v", name, err)
		return &WrappedProcess{}
	}
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		log.FileLogger.Fatalf("Failed to get %s.CpuPercent: %v", name, err)
		return &WrappedProcess{}
	}
	cpuAffinity, _ := p.CPUAffinity()
	memInfo, err := p.MemoryInfo()
	if err != nil {
		log.FileLogger.Fatalf("Failed to get %s.Meminfo: %v", name, err)
		return &WrappedProcess{}
	}

	ppid, err := p.Ppid()
	if err != nil {
		log.FileLogger.Fatalf("Failed to get %s.Ppid: %v", name, err)
		return &WrappedProcess{}
	}
	return &WrappedProcess{
		Exe:           exe,
		Pid:           p.Pid,
		PPid:          ppid,
		Name:          name,
		Memory:        memInfo.RSS,
		CpuAffinity:   cpuAffinity,
		PercentMemory: memPercent,
		PercentCpu:    cpuPercent,
	}
}

func printProcesses(procs []*ps.Process) {
	for _, proc := range procs {
		log.FileLogger.Infof("Process: %v", proc)
		name, err := proc.Name()
		if err != nil {
			log.ConsoleLogger.Fatalf("Failed to get process name: %v", err)
		}
		log.FileLogger.Infof("Process: %v", name)
	}
}
