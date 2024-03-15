/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aaronlifton/nvim-watcher/log"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

var p *mpb.Progress
var wg sync.WaitGroup
var outdatedPlugins []string
var ShouldFetchAll bool

const batchSize = 3
const barTotal = 100
const lazypath = "/Users/aaron/.local/share/nvim/lazy"

// updateGitCmd represents the updateGit command
var updateGitCmd = &cobra.Command{
	Use:   "watch-git",
	Short: "Updates all neovim plugins in batches.",
	Long: `* Updating packages asynchronously to prevent ERR_NO_FILES (OSX Ulimit) errors
  via parallelized small batches using go workgroups`,
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Println("watch-git called")
		log.FileLogger.Info("watch-git called")
		outdated := findOutdatedPlugins()
		log.FileLogger.Info(strings.Join(outdated, ","))
		// fetchUpdatesInBatches(outdated)
	},
}

func init() {
	updateGitCmd.Flags().BoolVarP(&ShouldFetchAll, "fetch_all", "a", false, "Fetch all plugins, rather than just the outdated ones.")
	rootCmd.AddCommand(updateGitCmd)

	log.Init()
	// zap.RedirectStdLog(log.GetConsoleLogger().Desugar())
	// defer fileLogger.Sync()
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

func fetchUpdatesInBatches(outdatedDirs []string) {
	done := make(chan interface{})
	dirs, err := os.ReadDir(lazypath)
	if err != nil {
		log.ConsoleLogger.Errorf("Failed to read directory: %v", err)
	}

	// Filter out non-directory files
	var dirNames []string
	if ShouldFetchAll == true {
		for _, dir := range dirs {
			if dir.IsDir() {
				dirNames = append(dirNames, dir.Name())
			}
		}
	} else  {
		dirNames = outdatedPlugins
	}
	// Process directories in batches of batchSize (default 3)
	maxJobs := len(dirNames)
	log.FileLogger.Debugf("max_jobs = %d", maxJobs)
	log.FileLogger.Debugln("")

	// p := mpb.New(
	// 		mpb.WithOutput(color.Output),
	// 		mpb.WithAutoRefresh(),
	// 	)
	for i := 0; i < maxJobs; i += batchSize {
		var wg sync.WaitGroup
		wg.Add(batchSize)
		p = mpb.New(
			mpb.WithWidth(64),
			mpb.WithWaitGroup(&wg),
			// mpb.WithShutdownNotifier(done),
		)

		for j := 0; j < batchSize; j++ {
			index := (batchSize * i) + j
			if index >= len(dirNames) {
				wg.Done()
				continue
			}
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
			const dur = 300 * time.Millisecond

			//
			var qwg sync.WaitGroup
			go func() {
				defer wg.Done()
				qwg.Add(1)
				ticker := time.NewTicker(dur)
				// defer ticker.Stop()
				// start := time.Now()
			tickLoop:
				for {
					select {
					case t := <-ticker.C:
						log.FileLogger.Infof("Tick at ", t)
						// bar.EwmaIncrement(time.Since(start))
						bar.EwmaIncrement(dur)
						// return
					case <-done:
						log.FileLogger.Infoln("here")
						ticker.Stop()
						bar.SetCurrent(barTotal)
						qwg.Done()
						break tickLoop
					}
				}
			}()
			qwg.Wait()
			go fetchGit(index, dirNames[index], bar, done)
			// p.Wait()
			// go func() {
			// 			defer wg.Done()
			// 			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			// 			max := 100 * time.Millisecond
			// 			for i := 0; i < maxJobs; i++ {
			// 				// start variable is solely for EWMA calculation
			// 				// EWMA's unit of measure is an iteration's duration
			// 				start := time.Now()
			// 				time.Sleep(time.Duration(rng.Intn(10)+1) * max / 10)
			// 				// we need to call EwmaIncrement to fulfill ewma decorator's contract
			// 				bar.EwmaIncrement(time.Since(start))
			// 			}
			// 		}()

		}
		p.Wait()
		// wait for passed wg and for all bars to complete and flush
	}
}

// fetchGit runs 'git fetch' in the specified directory
func fetchGit(index int, dirName string, bar *mpb.Bar, done chan interface{}) error {
	var output []byte

	args := []string{
		"fetch",
		"--recurse-submodules",
		"--tags",  // also fetch remote tags
		"--force", // overwrite existing tags if needed
		"--progress",
	}
	cmd := exec.Command("git", args...)
	absPath := filepath.Join(lazypath, dirName)
	cmd.Dir = absPath

	log.FileLogger.Debug("CMD",
		zap.Dict(
			"command",
			zap.String("cmd", "git"),
			zap.String("args", strings.Join(args, " ")),
			zap.String("dir", absPath)))
	output, err := cmd.CombinedOutput()

	log.FileLogger.Infoln(output)
	if err != nil {
		log.FileLogger.Error("Failed to fetch git", zap.Error(err))
	}

	log.GitLogger.Info(zap.String("dir", absPath), zap.String("output", string(output)))

	// err := cmd.Run()
	if err != nil {
		log.FileLogger.Info(zap.Dict("error",
			zap.String("dir", absPath),
			zap.String("error", err.Error()),
		))
	}
	done <- struct{}{}
	return nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// func writeCmdOutput() {
// 	outputFilePath := "logs/cmd_output.log"
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		log.FileLogger.Error("Failed to get current working directory", zap.Error(err))
// 		return
// 	}
// 	outputFilePath = filepath.Join(cwd, outputFilePath)
// 	file, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
// 	if err != nil {
// 		log.FileLogger.Error("Failed to open file", zap.Error(err))
// 		return
// 	}
// 	defer file.Close()

// 	for _, line := range cmdOutput {
// 		if _, err := file.WriteString(line + "\n"); err != nil {
// 			log.FileLogger.Error("Failed to write to file", zap.Error(err))
// 			return
// 		}
// 	}
// }
// func updateGit() {
// 	// cd to /Users/aaron/.local/share/nvim/lazy
// 	if err := os.Chdir("/Users/aaron/.local/share/nvim/lazy"); err != nil {
// 		log.ConsoleLogger.Fatalf("Failed to change directory: %v", err)
// 	}
// 	// FetchUpdatesInBatches fetches updates from remote git repositories in batches of three
// 	// Call FetchUpdatesInBatches to start the update process
// 	fetchUpdatesInBatches()
// 	// writeCmdOutput()
// 	log.CombinedLogger.Infoln("Done")

// }

func checkoutMainBranch(dirPath string, branchName string) {
	dirName := filepath.Base(dirPath)
	cmd := exec.Command("git", "-C", dirPath, "fetch", "origin", branchName)
	if err := cmd.Run(); err != nil {
		log.FileLogger.Errorf("Failed to fetch 'main' branch for %s: %v", dirName, err)
	} else {
		// Checkout the fetched main branch
		args := []string{"-C", dirPath, "checkout", branchName}
		cmd = exec.Command("git", args...)
		log.GitLogger.Info("CMD",
			zap.Dict(
				"command",
				zap.String("cmd", "git"),
				zap.String("args", strings.Join(args, " ")),
				zap.String("dir", dirPath)),
		)

		log.CombinedGitLogger.Infof("Running %s", cmd.String())
		stdoutStderr, err := cmd.CombinedOutput()
		log.GitLogger.Infoln(stdoutStderr)
		if err != nil {
			log.FileLogger.Errorf("Failed to checkout 'main' branch for %s: %v", dirName, err)
		} else {
			log.FileLogger.Infof("Checked out 'main' branch for %s", dirName)
		}
	}
}

func findOutdatedPlugins() []string {
	log.CombinedLogger.Infoln("Finding outdated plugins")
	lazypath := "/Users/aaron/.local/share/nvim/lazy"
	dirs, err := os.ReadDir(lazypath)
	outdatedDirs := make([]string, len(dirs))
	if err != nil {
		log.ConsoleLogger.Errorf("Failed to read directory: %v", err)
		return []string{}
	}

	outdatedCount := 0
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dirPath := filepath.Join(lazypath, dir.Name())
		// cmd := exec.Command("git", "-C", dirPath, "remote", "update")
		// if err := cmd.Run(); err != nil {
		//  log.FileLogger.Errorf("Failed to update git remote for %s: %v", dir.Name(), err)
		//  continue
		// }

		args := []string{"-C", dirPath, "status", "-uno"}
		cmd := exec.Command("git", args...)
		output, err := cmd.Output()
		log.FileLogger.Debug("CMD",
			zap.Dict(
				"command",
				zap.String("command", "git"),
				zap.String("args", strings.Join(args, " ")),
				zap.String("dir", dirPath)),
		)

		if err != nil {
			log.CombinedLogger.Errorf("Failed to get git status for %s: %v", dir.Name(), err)
			continue
		}

		strOut := string(output)
		dirName := filepath.Base(dirPath)
		if strings.Contains(strOut, "Your branch is behind") {
			outdatedDirs = append(outdatedDirs, dirPath)
			log.FileLogger.Infof("Found an outdated plugin (%s)", dirName)
		} else if strings.Contains(strOut, "HEAD detached at") {
			// Check for the presence of a 'main' or 'master' branch
			mainBranchExists := false
			masterBranchExists := false
			outdatedDirs = append(outdatedDirs, dirPath)

			args := []string{"-C", dirPath, "show-ref", "--verify",  "refs/heads/main"}
			cmd := exec.Command("git", args...)
			log.CombinedGitLogger.Infof("Running %s", cmd.String())
			log.GitLogger.Debug("CMD",
				zap.Dict(
					"command",
					zap.String("command", "git"),
					zap.String("args", strings.Join(args, " ")),
					zap.String("dir", dirPath)),
			)
			stdoutStderr, _ := cmd.CombinedOutput()
			log.GitLogger.Infoln(stdoutStderr)
			if strings.Contains(string(stdoutStderr), "not a valid ref") {
				mainBranchExists = false
			} else {
				mainBranchExists = true
			}

			if (mainBranchExists) {
				checkoutMainBranch(dirPath, "main")
			} else {
				checkoutMainBranch(dirPath, "master")
			}

			if (!masterBranchExists && !mainBranchExists) {
				log.FileLogger.Infof("Plugin %s does not have a 'main' or 'master' branch", dirName)
			}
		}
	}

	log.CombinedLogger.Infof("Found %d directories in %s", len(dirs), lazypath)
	log.CombinedLogger.Infof("Number of outdated plugins: %d", outdatedCount)
	// Do we need to allocate for all dirs? Don't think so.'
  return outdatedPlugins
}
