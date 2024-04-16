//go:build darwin
// +build darwin

package cmd

import (
	"bytes"
	"encoding/binary"
	"log"
	"slices"
	"syscall"
	"unsafe"

	gps "github.com/mitchellh/go-ps"
)

type DarwinProcess struct {
	pid    int
	ppid   int
	binary string
}

func (p *DarwinProcess) Pid() int {
	return p.pid
}

func (p *DarwinProcess) PPid() int {
	return p.ppid
}

func (p *DarwinProcess) Executable() string {
	return p.binary
}

func findProcess(pid int) (gps.Process, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

func processes() ([]gps.Process, error) {
	buf, err := darwinSyscall()
	if err != nil {
		return nil, err
	}

	procs := make([]*kinfoProc, 0, 50)
	k := 0
	for i := _KINFO_STRUCT_SIZE; i < buf.Len(); i += _KINFO_STRUCT_SIZE {
		proc := &kinfoProc{}
		err = binary.Read(bytes.NewBuffer(buf.Bytes()[k:i]), binary.LittleEndian, proc)
		if err != nil {
			return nil, err
		}

		k = i
		procs = append(procs, proc)
	}

	darwinProcs := make([]gps.Process, len(procs))
	for i, p := range procs {
		darwinProcs[i] = &DarwinProcess{
			pid:    int(p.Pid),
			ppid:   int(p.PPid),
			binary: darwinCstring(p.Comm),
		}
	}

	return darwinProcs, nil
}

func darwinCstring(s [16]byte) string {
	i := 0
	for _, b := range s {
		if b != 0 {
			i++
		} else {
			break
		}
	}

	return string(s[:i])
}

// This Go function `darwinSyscall()` is making a system call to retrieve information about all processes running on a Darwin-based system (like macOS).
//
// Here's a breakdown of what the code is doing:
//
// 1. It sets up the `mib` array containing specific values that are used by the `sysctl` system call to query information about processes.
//
// 2. It then calls `syscall.Syscall6` with a `size` value of 0 to determine the required buffer size to hold the process information.
//
// 3. If the call is successful, it allocates a byte slice `bs` of the determined size to store the process data.
//
// 4. It then calls `syscall.Syscall6` again, this time passing the allocated buffer as an argument to retrieve the process data.
//
// 5. Finally, it creates a `bytes.Buffer` using the retrieved data and returns it along with a `nil` error if the syscall is successful.
//
// In summary, this function is using low-level system calls to interact with the kernel and retrieve information about processes on a Darwin-based system.
func darwinSyscall() (*bytes.Buffer, error) {
	mib := [4]int32{_CTRL_KERN, _KERN_PROC, _KERN_PROC_ALL, 0}
	size := uintptr(0)

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	bs := make([]byte, size)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&bs[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	return bytes.NewBuffer(bs[0:size]), nil
}

const (
	_CTRL_KERN         = 1
	_KERN_PROC         = 14
	_KERN_PROC_ALL     = 0
	_KINFO_STRUCT_SIZE = 648
)

type kinfoProc struct {
	_    [40]byte
	Pid  int32
	_    [199]byte
	Comm [16]byte
	_    [301]byte
	PPid int32
	_    [84]byte
}

func GetProcesses() []gps.Process {
	buf, err := darwinSyscall()
	if err != nil {
		log.Fatal(err)
	}
	firstProc := &kinfoProc{}
	err = binary.Read(buf, binary.LittleEndian, firstProc)
	if err != nil {
		log.Fatal(err)
	}

	procs, err := processes()
	if err != nil {
		log.Fatal(err)
	}

	wantedExecutables := []string{
		"Codeium",
		"Copilot",
		"TabNine",
		"biome",
		"biome-agent",
		"biome-registry",
		"biome-service",
		"biomesyncd",
		"clangd",
		"cssls",
		"gopls",
		"html_ls",
		"jedi_language_server",
		"json_ls",
		"nvim",
		"pyls",
		"rust-analyzer",
		"sourcery",
		"sqls",
		"sumneko_lua",
		"terraform-ls",
		"tsserver",
	}
	relevantProcs := make([]gps.Process, 0, 50)
	for _, p := range procs {
		if slices.Contains(wantedExecutables, p.Executable()) {
			relevantProcs = append(relevantProcs, p)
		}
	}
	for _, p := range relevantProcs {
		log.Printf("PID: %d, PPID: %d, Executable: %s\n", p.Pid(), p.PPid(), p.Executable())
	}
	return relevantProcs
}
