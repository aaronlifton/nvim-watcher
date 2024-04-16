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
	"time"

	"github.com/aaronlifton/nvim-watcher/log"

	// "github.com/shirou/gopsutil/v3/cpu"
	// "github.com/shirou/gopsutil/v3/load"
	// "github.com/shirou/gopsutil/v3/net"
	// "github.com/shirou/gopsutil/v3/process"
	"github.com/go-co-op/gocron"
)

type ProcessSupervisor interface {
	Start() error
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
	job, err := scheduler.Every(s.config.durationMinutes).Minutes().Do(Task)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.ConsoleLogger.Info("Starting gocron")
	log.ConsoleLogger.Infof("Job will run next at %s", job.NextRun)
	// scheduler.StartAsync()
	scheduler2 := gocron.NewScheduler(time.UTC)
	job2, err := scheduler2.Every(10).Seconds().Do(StreamLog)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
	}
	log.ConsoleLogger.Infof("Job will run next at %s", job2.NextRun)
	log.ConsoleLogger.Info("Starting blocking log stream")
	scheduler2.StartBlocking()
}

func StreamLog() error {
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

func Task() error {
	log.ConsoleLogger.Info("Running task")
	ts := time.Now().Format(time.RFC3339)
	log.CronLogger.Info("Task ran at: ", ts)

	processList := RunWatchProcesses()
	processDataList := make([]ProcessData, len(processList))
	for _, p := range processList {
		if p == nil {
			continue
		}
		pd, err := p.GetStats()
		if err != nil {
			log.ConsoleLogger.Fatal(err)
			return err
		}
		log.ConsoleLogger.Infof(
			"Found process: %v (%d: %v)\t Memory: %v\tCPU: %v\nAffinity: %v\nMemory RSS: %v\n",
			pd.Name,
			pd.Pid,
			pd.Exe,
			pd.PercentMemory,
			pd.PercentCpu,
			pd.CpuAffinity,
			pd.PercentMemory,
		)
		processDataList = append(processDataList, pd)
	}
	return nil
}
