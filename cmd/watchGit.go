/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/aaronlifton/nvim-watcher/logger"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var p *mpb.Progress
var wg sync.WaitGroup

const batchSize = 3
const barTotal = 100
const lazypath = "/Users/aaron/.local/share/nvim/lazy"

// updateGitCmd represents the updateGit command
var updateGitCmd = &cobra.Command{
	Use:   "watch-git",
	Short: "Updates all neovim plugins in batches.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("watch-git called")
		FetchUpdatesInBatches()
	},
}

func init() {
	rootCmd.AddCommand(updateGitCmd)
	logger.New() // passed wg will be accounted at p.Wait() call

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateGitCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateGitCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func addBar(bnum int, i int, name string) *mpb.Bar {
	decorName := fmt.Sprintf("Batch#%d Plugin#%d %s:", bnum, i, name)
	bar := p.AddBar(int64(barTotal),
		mpb.PrependDecorators(
			// simple name decorator
			decor.Name(decorName),
			// decor.DSyncWidth bit enables column width synchronization
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(
				// ETA decorator with ewma age of 30
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), "done",
			),
		),
	)
	return bar
}

func FetchUpdatesInBatches() {
	command_chan := make(chan int)
	dirs, err := os.ReadDir(lazypath)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	// Filter out non-directory files
	var dirNames []string
	for _, dir := range dirs {
		if dir.IsDir() {
			dirNames = append(dirNames, dir.Name())
		}
	}
	log.Printf("Found %d directories", len(dirNames))
	// Process directories in batches of batchSize (default 3)
	maxJobs := len(dirNames)
	log.Printf("max_jobs = %d", maxJobs)
	log.Print("\n")

	for i := 0; i < maxJobs; i += batchSize {
		wg.Add(batchSize)
		p = mpb.New(mpb.WithWaitGroup(&wg))

		for j := 0; j < batchSize; j++ {
			index := (batchSize * i) + j
			name := fmt.Sprintf("Batch#%d Job#%d Plugin#%s:", i/batchSize, j, dirNames[index])
			bar := p.AddBar(int64(barTotal),
				mpb.PrependDecorators(
					decor.Name(name),
					// decor.DSyncWidth bit enables column width synchronization
					decor.Percentage(decor.WCSyncSpace),
				),
				mpb.AppendDecorators(
					decor.OnComplete(
						// ETA decorator with ewma age of 30
						decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), "done",
					),
				),
			)
			// simulating some work
			go func() {
				defer wg.Done()
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				if index < len(dirNames) {
					go fetchGit(index, dirNames[index], bar, command_chan)
				}
				for {
					select {
					case <-ticker.C:
						start := time.Now()
						bar.EwmaIncrement(time.Since(start))
						if bar.Current() >= barTotal {
							break
						}
					case <-command_chan:
						log.Printf("Command received")
						bar.SetCurrent(barTotal)
						break;
					case <-time.After(10 * time.Second):
						log.Printf("Timeout (10 s)")
					}
				} // rng := rand.New(rand.NewSource(time.Now().UnixNano()))
				// max := 100 * time.Millisecond
				// if index < len(dirNames) {
				//     go fetchGit(dirNames[index], bar)
				// }
				// for i := 0; i < barTotal; i++ {
				//     // start variable is solely for EWMA calculation
				//     // EWMA's unit of measure is an iteration's duration
				//     start := time.Now()
				//     time.Sleep(time.Duration(rng.Intn(10)+1) * max / 10)
				//     // we need to call EwmaIncrement to fulfill ewma decorator's contract
				//     bar.EwmaIncrement(time.Since(start))
				// }
			}()
		}
		// wait for passed wg and for all bars to complete and flush
		p.Wait()
	}
}

// fetchGit runs 'git fetch' in the specified directory
func fetchGit(index int, dirName string, bar *mpb.Bar, c chan int) error {
	args := []string{
		"fetch",
		"--recurse-submodules",
		"--tags",  // also fetch remote tags
		"--force", // overwrite existing tags if needed
		"--progress",
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dirName

	absPath := filepath.Join(lazypath, dirName)
	log.Printf("CMD: %s in %s", cmd, absPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run command %s in %s: %v", cmd, dirName, err)
		// bar.SetCurrent(0)
	}
	bar.SetCurrent(barTotal)
	c <- index
	return nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func updateGit() {
	// cd to /Users/aaron/.local/share/nvim/lazy
	if err := os.Chdir("/Users/aaron/.local/share/nvim/lazy"); err != nil {
		log.Fatalf("Failed to change directory: %v", err)
	}
	// FetchUpdatesInBatches fetches updates from remote git repositories in batches of three
	// Call FetchUpdatesInBatches to start the update process
	FetchUpdatesInBatches()
}
