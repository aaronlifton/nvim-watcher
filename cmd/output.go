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

var wg sync.WaitGroup

func Draw(pd []*WrappedProcess) {
	formattedData := Top10Processes(pd)
	PrintChart(formattedData)
}

func Top10Processes(pd []*WrappedProcess) []WrappedProcess {
	slices.SortFunc(pd, func(i, j *WrappedProcess) int {
		return cmp.Compare(j.PercentCpu, i.PercentCpu)
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
	maxHeight := min(len(pd)/2, height/2) + 3
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
		"Processes",
		labels,
		data,
		[]string{"Memory", "Cpu"},
		colors,
		float64(width),
		false,
		"",
	)
}

// TODO: Implement keyboard controls
func ListenForInput() {
	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		if key.Code == keys.CtrlC {
			return true, nil // Stop listener by returning true on Ctrl+C
		}

		fmt.Println("\r" + key.String()) // Print every key press
		return false, nil                // Return false to continue listening
	})
}
