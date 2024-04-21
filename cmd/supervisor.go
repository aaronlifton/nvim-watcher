/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (

	// "os"
	// "os/exec"

	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/aaronlifton/nvim-watcher/log"
	"go.uber.org/zap"

	// "github.com/shirou/gopsutil/v3/cpu"
	// "github.com/shirou/gopsutil/v3/load"
	// "github.com/shirou/gopsutil/v3/net"
	// "github.com/shirou/gopsutil/v3/process"
	"github.com/go-co-op/gocron"
	// gps "github.com/mitchellh/go-ps"
	ps "github.com/shirou/gopsutil/v3/process"
)

var Executables = []string{"nvim", "TabNine", "codeium", "Copilot", "sourcery", "biomesyncd", "biome"}

var (
	initialized bool = false
	Data        chan interface{}
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
	// every 30 minutes, run GetProcesses, and if the returned array does not contain nvim or vscode, kill all the processes
	// start the schedule
	// do this with gocron
	var err error
	scheduler := gocron.NewScheduler(time.UTC)
	job, err := scheduler.Every(s.config.durationMinutes).Minutes().Do(s.PeriodicTask)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.FileLogger.Info("Starting gocron")
	log.FileLogger.Infof("Job will run next at %s", job.NextRun)
	// scheduler.StartAsync()
	scheduler2 := gocron.NewScheduler(time.UTC)
	job2, err := scheduler2.Every(1).Seconds().Do(s.PeriodicTask)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.FileLogger.Infof("Job will run next at %s", job2.NextRun)
	log.FileLogger.Info("Starting blocking log stream")
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
	relevantProcesses := []*ps.Process{}
	processList, err := ps.Processes()
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to get processes: %v", err)
	}

	for _, process := range processList {
		name, _ := process.Exe()
		name = filepath.Base(name)
		if slices.Contains(Executables, name) {
			relevantProcesses = append(relevantProcesses, process)
		}
	}
	if len(relevantProcesses) == 0 {
		log.ConsoleLogger.Fatal("No relevant processes found")
		return []*WrappedProcess{}
	}
	wrappedProcesses := make([]*WrappedProcess, len(relevantProcesses))
	for i, process := range relevantProcesses {
		wp, err := NewWrappedProcess(process)
		if err != nil {
			log.ConsoleLogger.Fatalf("Failed to get process data: %v", err)
		}

		wrappedProcesses[i] = wp
	}
	return wrappedProcesses
}

func NewWrappedProcess(p *ps.Process) (*WrappedProcess, error) {
	exe, _ := p.Exe()
	name := filepath.Base(exe)
	memPercent, err := p.MemoryPercent()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return &WrappedProcess{}, err
	}
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return &WrappedProcess{}, err
	}
	cpuAffinity, _ := p.CPUAffinity()
	memInfo, err := p.MemoryInfo()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return &WrappedProcess{}, err
	}

	ppid, err := p.Ppid()
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return &WrappedProcess{}, err
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
	}, nil
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
