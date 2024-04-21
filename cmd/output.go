package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/aaronlifton/nvim-watcher/log"

	// "github.com/vbauerster/mpb/v8"
	// "github.com/vbauerster/mpb/v8/decor"

	tgraph "github.com/daoleno/tgraph"
	"golang.org/x/term"
)

var (
	wg          sync.WaitGroup
	currentSort string = "cpu"
)

func Draw(pd []*WrappedProcess) {
	formattedData := Top10Processes(pd)
	PrintChart(formattedData)
}

func Top10Processes(pd []*WrappedProcess) []WrappedProcess {
	slices.SortFunc(pd, func(i, j *WrappedProcess) int {
		if currentSort == "cpu" {
			return cmp.Compare(j.PercentCpu, i.PercentCpu)
		} else {
			return cmp.Compare(j.Memory, i.Memory)
		}
	})
	first10 := make([]WrappedProcess, 10)
	for i := 0; i < 10; i++ {
		first10[i] = *pd[i]
	}
	return first10
}

func PrintChart(pd []WrappedProcess) {
	if term.IsTerminal(0) {
	} else {
		log.ConsoleLogger.Fatal("Not in a term")
	}
	width, height, err := term.GetSize(0)
	if err != nil {
		log.ConsoleLogger.Fatal(err)
		return
	}
	log.FileLogger.Infof("Terminal height: %d", height)
	maxHeight := min(len(pd)/2, height/2) + 2
	if height%2 == 0 {
		fmt.Println("")
	}
	log.FileLogger.Infof("Chart height: %d", height)
	labels := make([]string, maxHeight)
	data := make([][]float64, maxHeight)
	colors := []string{"green", "blue"}
	for i, p := range pd[:maxHeight] {
		labels[i] = fmt.Sprintf("%s (%d)", p.Name, p.Pid)
		mem := float64(p.PercentMemory)
		cpu := p.PercentCpu
		data[i] = []float64{mem, cpu}
	}
	tgraph.Chart(
		fmt.Sprintf("Top %d processes (Sort: %s)", len(data), currentSort),
		labels,
		data,
		[]string{"Memory", "Cpu"},
		colors,
		float64(width),
		false,
		"",
	)
}

func ListenForInput() {
	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		if key.Code == keys.CtrlC {
			return true, nil
		}
		if key.Code == keys.CtrlM {
			if currentSort == "cpu" {
				currentSort = "memory"
			} else {
				currentSort = "cpu"
			}
		}
		return false, nil
	})
}
