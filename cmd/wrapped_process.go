/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"os"
	"os/exec"
)

type WrappedProcessManager interface {
	Kill(cmd *exec.Cmd)
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

func (p *WrappedProcess) Kill() error {
	osProcess := os.Process{Pid: int(p.Pid)}
	cmd := &exec.Cmd{Process: &osProcess}
	return Kill(cmd)
}
