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
	p = mpb.New(mpb.WithWaitGroup(&wg))
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
	wg.Add(batchSize)
	for i := 0; i < maxJobs; i += batchSize {
		log.Printf("Processing batch #%d", i/batchSize+1)
		wg.Add(batchSize)
		go func(i int) {
			defer wg.Done()
			batch := dirNames[i:min(i+batchSize, len(dirNames))]
			for j, dirName := range batch {
				bar := addBar(i, j, dirName)
				absPath, err := filepath.Abs(filepath.Join(lazypath, dirName))
				if err != nil {
					log.Printf("Failed to get absolute path for %s: %v", dirName, err)
					continue
				}
				if err := fetchGit(absPath, bar); err != nil {
					log.Printf("Failed to fetch git in %s: %v", dirName, err)
				}
			}
		}(i)
		wg.Wait()
	}
}

// fetchGit runs 'git fetch' in the specified directory
func fetchGit(dirName string, bar *mpb.Bar) error {
	args := []string{
		"fetch",
		"--recurse-submodules",
		"--tags",  // also fetch remote tags
		"--force", // overwrite existing tags if needed
		"--progress",
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dirName



	log.Println("Running git fetch in", dirName)
	log.Printf("cmd: %s", cmd)
	cmd.Dir = dirName
	start := time.Now() // Start time for tracking progress

	// Create a goroutine to increment the progress bar based on the elapsed time
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-done:
				return
			default:
				bar.EwmaIncrement(time.Since(start))
				time.Sleep(100 * time.Millisecond) // Adjust the sleep duration as needed
			}
		}
	}()

	err := cmd.Run()

	// Signal the progress bar increment goroutine to stop
	done <- struct{}{}

	if err != nil {
		bar.SetCurrent(0) // Reset the bar to 0 on error
		return err
	} else {
		bar.SetCurrent(barTotal) // Set the bar to 100% on success
	}

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
