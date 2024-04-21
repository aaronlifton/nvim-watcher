package cmd

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
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
