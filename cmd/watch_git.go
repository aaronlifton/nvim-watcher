/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aaronlifton/nvim-watcher/log"
	"github.com/mitchellh/go-ps"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"go.uber.org/zap"
)

var (
	p               *mpb.Progress
	wg              sync.WaitGroup
	outdatedPlugins []string
	ShouldFetchAll  bool
)

const (
	batchSize = 3
	barTotal  = 100
	lazypath  = "/Users/aaron/.local/share/nvim/lazy"
)

// UpdatePluginsCmd represents the updateGit command
var UpdatePluginsCmd = &cobra.Command{
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
	UpdatePluginsCmd.Flags().BoolVarP(&ShouldFetchAll, "fetch_all", "a", false, "Fetch all plugins, rather than just the outdated ones.")
	rootCmd.AddCommand(UpdatePluginsCmd)

	log.Init()
	// zap.RedirectStdLog(log.GetConsoleLogger().Desugar())
	// defer fileLogger.Sync()
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// UpdatePluginsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// UpdatePluginsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

func filterDirsBasedOnFetchOption(dirs []os.DirEntry) (filteredDirs []string) {
	if ShouldFetchAll {
		for _, dir := range dirs {
			if dir.IsDir() {
				filteredDirs = append(filteredDirs, dir.Name())
			}
		}
	} else {
		filteredDirs = outdatedPlugins
	}

	return filteredDirs
}

func fetchUpdatesInBatches(outdatedDirs []string) {
	done := make(chan interface{})
	dirs, err := os.ReadDir(lazypath)
	if err != nil {
		log.ConsoleLogger.Fatalf("Failed to read directory: %v", err)
	}

	// Filter out non-directory files
	var filteredDirs []string
	if ShouldFetchAll == true {
		for _, dir := range dirs {
			if dir.IsDir() {
				filteredDirs = append(filteredDirs, dir.Name())
			}
		}
	} else {
		filteredDirs = outdatedPlugins
	}
	// Process directories in batches of batchSize (default 3)
	maxJobs := len(filteredDirs)
	log.FileLogger.Debugf("max_jobs = %d", maxJobs)
	log.FileLogger.Info("")

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
			if index >= len(filteredDirs) {
				wg.Done()
				continue
			}
			name := fmt.Sprintf("Batch#%d Job#%d Plugin#%s:", i/batchSize, j, filteredDirs[index])
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
			go fetchGit(index, filteredDirs[index], bar, done)
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

func tryRunAndLog(cmd *exec.Cmd) (string, error) {
	log.ConsoleLogger.Infof("Running %s", cmd.String())
	log.LogGitCommand(cmd)
	stdoutStderr, err := cmd.CombinedOutput()
	log.GitLogger.Infoln(string(stdoutStderr))
	return string(stdoutStderr[:]), err
}

func checkoutMainBranch(dirPath string, branchName string) {
	dirName := filepath.Base(dirPath)
	cmd := exec.Command("git", "-C", dirPath, "fetch", "origin", branchName)
	stdoutStderr, err := tryRunAndLog(cmd)
	if err != nil {
		log.ConsoleLogger.Errorf(
			"Failed to fetch 'main' branch for %s: %v", dirName, err,
		)
	}
	log.CombinedGitLogger.Infoln(stdoutStderr)
	// Checkout the fetched main branch
	args := []string{"-C", dirPath, "checkout", branchName}
	cmd = exec.Command("git", args...)
	stdoutStderr, err = tryRunAndLog(cmd)
	if err != nil {
		log.ConsoleLogger.Errorf(
			"Failed to checkout 'main' branch for %s: %v", dirName, err,
		)
	}
	zap.S().Info(stdoutStderr)
	log.CombinedGitLogger.Info(stdoutStderr)
}

func findOutdatedPlugins() []string {
	gitPids := make([]int, 0)
	processes, err := ps.Processes()
	if err != nil {
		log.CombinedLogger.Fatalf("Failed to get processes: %v", err)
	}
	for _, p := range processes {
		if strings.Contains(p.Executable(), "git") {
			gitPids = append(gitPids, p.Pid())
		}
	}
	if len(gitPids) >= 0 {
		pidStrings := make([]string, len(gitPids))
		for i, pid := range gitPids {
			pidStrings[i] = strconv.Itoa(pid)
		}
		log.CombinedLogger.Fatalf(
			"Conflicting git processes running: %s",
			strings.Join(pidStrings, ", "),
		)
		log.CombinedLogger.Fatalf("No git processes found")
	}
	log.CombinedLogger.Infoln("Finding outdated plugins")
	lazypath := filepath.Join(os.Getenv("HOME"), ".local/share/nvim/lazy")
	dirs, err := os.ReadDir(lazypath)
	outdatedDirs := make([]string, len(dirs))
	if err != nil {
		log.ConsoleLogger.Errorf("Failed to read directory: %v", err)
		return []string{}
	}
	var wg sync.WaitGroup
	p = mpb.New(
		mpb.WithWidth(64),
		mpb.WithWaitGroup(&wg),
		// mpb.WithShutdownNotifier(done),
	)
	bar := p.AddBar(int64(barTotal),
		mpb.PrependDecorators(
			decor.Name("Checking packages..."),
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
	for _, dir := range dirs {
		bar.EwmaIncrement(dur)
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
		log.CombinedLogger.Debugln("CMD",
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
			log.CombinedLogger.Infof("Found an outdated plugin (%s)", dirName)
		} else if strings.Contains(strOut, "HEAD detached at") {
			// Check for the presence of a 'main' or 'master' branch
			mainBranchExists := false
			masterBranchExists := false
			outdatedDirs = append(outdatedDirs, dirPath)

			args := []string{"-C", dirPath, "show-ref", "--verify", "refs/heads/main"}
			cmd := exec.Command("git", args...)
			log.CombinedLogger.Infof("Running %s", cmd.String())
			log.CombinedLogger.Debug("CMD",
				zap.Dict(
					"command",
					zap.String("command", "git"),
					zap.String("args", strings.Join(args, " ")),
					zap.String("dir", dirPath)),
			)
			stdoutStderr, _ := cmd.CombinedOutput()
			log.CombinedLogger.Infoln(string(stdoutStderr))
			if strings.Contains(string(stdoutStderr), "not a valid ref") {
				mainBranchExists = false
			} else {
				mainBranchExists = true
			}

			if mainBranchExists {
				checkoutMainBranch(dirPath, "main")
			} else {
				checkoutMainBranch(dirPath, "master")
			}

			if !masterBranchExists && !mainBranchExists {
				log.CombinedLogger.Infof("Plugin %s does not have a 'main' or 'master' branch", dirName)
			}
		}
	}
	bar.SetCurrent(100)

	outdatedCount := len(outdatedDirs)
	log.CombinedLogger.Infof("Found %d directories in %s", len(dirs), lazypath)
	log.CombinedLogger.Infof("Found %d outdated plugins", outdatedCount)
	// Do we need to allocate for all dirs? Don't think so.'
	return outdatedPlugins
}
