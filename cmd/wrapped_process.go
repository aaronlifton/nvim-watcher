/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewWatchProcessesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "watch-processes",
		Short: `Visualize memory and CPU usage by Neovim and Neovim-related
			processes such as AI plugins including ChatGPT, Codeium, Copilot,
			Sourcegraph, and TabNine.`,
		Run: func(cmd *cobra.Command, args []string) {},
	}
}

type WrappedProcess struct {
	Exe           string
	Name          string
	Pid           int32
	PPid          int32
	CpuAffinity   []int32
	PercentMemory float32
	Memory        uint64
	PercentCpu    float64
}

type WrappedProcessManager interface {
	Kill(cmd *exec.Cmd)
}

type ProcessWatcher struct{}

func (p *WrappedProcess) Kill() error {
	osProcess := os.Process{Pid: int(p.Pid)}
	cmd := &exec.Cmd{Process: &osProcess}
	return Kill(cmd)
}
