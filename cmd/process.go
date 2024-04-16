package cmd

import (
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/shirou/gopsutil/v3/process"
)

//
// type ProcessData struct {
// 	Pid           int32
// 	PPid          int32
// 	Exe           string
// 	Memory        uint64
// 	Cpu           float64
// 	PercentMemory float32
// 	PercentCpu    float64
// }

// type commandExecutor interface {
// 	Output() ([]byte, error)
// }
//
// var NewCommand = func(name string, arg ...string) commandExecutor {
// 	return *exec.Command(name, arg...)
// }

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}

func NewProcess(pid int32) (*process.Process, error) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}
	return proc, err
}

func Kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
		// negative Pid represents a PgId
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	return cmd.Process.Kill()
}

func IsRunning(cmd *exec.Cmd) (bool, error) {
	proc, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		return false, err
	}
	isRunning, err := proc.IsRunning()
	return isRunning, nil
}
