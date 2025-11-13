package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type BranchStatus struct {
	Name    string
	IsDirty bool
	Ahead   int
	Behind  int
	Status  string
	Current bool
}

type RepoStatus struct {
	Path          string
	Branches      []BranchStatus
	CurrentBranch string
	Error         string
}

func main() {
	// CLI flags
	dir := flag.String("dir", ".", "Directory to scan for git repositories")
	showClean := flag.Bool("show-clean", false, "Show clean branches in addition to dirty ones")
	verbose := flag.Bool("verbose", false, "Verbose output")
	maxDepth := flag.Int("max-depth", 10, "Maximum directory depth to search")
	parallel := flag.Bool("parallel", false, "Process repositories in parallel (faster)")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")

	flag.Parse()

	// Resolve absolute path
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	if *verbose && !*jsonOutput {
		fmt.Printf("Scanning directory: %s\n", absDir)
		fmt.Printf("Show clean branches: %v\n", *showClean)
		fmt.Printf("Parallel processing: %v\n", *parallel)
		fmt.Println()
	}

	repos := findGitRepos(absDir, *maxDepth, *verbose && !*jsonOutput)

	if len(repos) == 0 {
		if !*jsonOutput {
			fmt.Println("No git repositories found.")
		}
		return
	}

	var statuses []RepoStatus

	if *parallel {
		statuses = analyzeReposParallel(repos, *showClean, *verbose && !*jsonOutput)
	} else {
		statuses = analyzeReposSequential(repos, *showClean, *verbose && !*jsonOutput)
	}

	if *jsonOutput {
		displayJSONOutput(statuses)
	} else {
		fmt.Printf("Found %d git repositor%s:\n\n", len(repos), pluralize(len(repos), "y", "ies"))
		for _, status := range statuses {
			displayRepoStatus(status, *showClean)
		}
	}
}

func findGitRepos(root string, maxDepth int, verbose bool) []string {
	var repos []string
	visited := make(map[string]bool)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: cannot access %s: %v\n", path, err)
			}
			return nil
		}

		// Calculate depth
		rel, _ := filepath.Rel(root, path)
		depth := len(strings.Split(rel, string(os.PathSeparator)))
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common directories that shouldn't be searched
		if info.IsDir() {
			basename := filepath.Base(path)
			if basename == "node_modules" || basename == "vendor" || basename == ".git" {
				return filepath.SkipDir
			}
		}

		// Check if this is a .git directory
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)

			// Avoid duplicates
			if !visited[repoPath] {
				visited[repoPath] = true
				repos = append(repos, repoPath)
				if verbose {
					fmt.Printf("Found repository: %s\n", repoPath)
				}
			}
			return filepath.SkipDir
		}

		return nil
	})

	return repos
}

func analyzeReposSequential(repos []string, includeClean bool, verbose bool) []RepoStatus {
	var statuses []RepoStatus
	for _, repoPath := range repos {
		status := analyzeRepo(repoPath, includeClean, verbose)
		statuses = append(statuses, status)
	}
	return statuses
}

func analyzeReposParallel(repos []string, includeClean bool, verbose bool) []RepoStatus {
	var wg sync.WaitGroup
	statusChan := make(chan RepoStatus, len(repos))

	for _, repoPath := range repos {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			status := analyzeRepo(path, includeClean, verbose)
			statusChan <- status
		}(repoPath)
	}

	go func() {
		wg.Wait()
		close(statusChan)
	}()

	var statuses []RepoStatus
	for status := range statusChan {
		statuses = append(statuses, status)
	}

	return statuses
}

func analyzeRepo(repoPath string, includeClean bool, verbose bool) RepoStatus {
	status := RepoStatus{
		Path:     repoPath,
		Branches: []BranchStatus{},
	}

	// Save the current branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		status.Error = fmt.Sprintf("Error getting current branch: %v", err)
		return status
	}
	currentBranch := strings.TrimSpace(string(output))
	status.CurrentBranch = currentBranch

	// Get all local branches
	cmd = exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Error getting branches for %s: %v\n", repoPath, err)
		}
		status.Error = fmt.Sprintf("Error getting branches: %v", err)
		return status
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, branch := range branches {
		if branch == "" {
			continue
		}

		branchStatus := analyzeBranch(repoPath, branch, currentBranch, verbose)

		// Only include if dirty or if we're showing clean branches
		if branchStatus.IsDirty || includeClean {
			status.Branches = append(status.Branches, branchStatus)
		}
	}

	// Return to original branch
	if currentBranch != "" {
		cmd = exec.Command("git", "checkout", "-q", currentBranch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: cannot return to branch %s in %s: %v\n", currentBranch, repoPath, err)
			}
		}
	}

	return status
}

func analyzeBranch(repoPath, branch, currentBranch string, verbose bool) BranchStatus {
	status := BranchStatus{
		Name:    branch,
		IsDirty: false,
		Current: branch == currentBranch,
	}

	// Only checkout if not already on this branch
	if branch != currentBranch {
		cmd := exec.Command("git", "checkout", "-q", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: cannot checkout %s in %s: %v\n", branch, repoPath, err)
			}
			status.Status = "Error checking out branch"
			return status
		}
	}

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: cannot get status for %s in %s: %v\n", branch, repoPath, err)
		}
		status.Status = "Error getting status"
		return status
	}

	if len(output) > 0 {
		status.IsDirty = true
		status.Status = parseGitStatus(string(output))
	} else {
		status.Status = "Clean"
	}

	// Check ahead/behind relative to upstream
	cmd = exec.Command("git", "rev-list", "--left-right", "--count", fmt.Sprintf("%s...@{u}", branch))
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err == nil {
		var ahead, behind int
		fmt.Sscanf(string(output), "%d\t%d", &ahead, &behind)
		status.Ahead = ahead
		status.Behind = behind
	}

	return status
}

func parseGitStatus(statusOutput string) string {
	scanner := bufio.NewScanner(strings.NewReader(statusOutput))

	modified := 0
	added := 0
	deleted := 0
	untracked := 0

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}

		status := line[0:2]
		switch {
		case strings.HasPrefix(status, "M") || strings.HasPrefix(status, " M"):
			modified++
		case strings.HasPrefix(status, "A") || strings.HasPrefix(status, " A"):
			added++
		case strings.HasPrefix(status, "D") || strings.HasPrefix(status, " D"):
			deleted++
		case strings.HasPrefix(status, "??"):
			untracked++
		}
	}

	var parts []string
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleted))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}

	if len(parts) == 0 {
		return "Clean"
	}

	return strings.Join(parts, ", ")
}

func displayRepoStatus(status RepoStatus, showClean bool) {
	if status.Error != "" {
		fmt.Printf("📁 %s - ERROR: %s\n\n", status.Path, status.Error)
		return
	}

	if len(status.Branches) == 0 {
		fmt.Printf("📁 %s\n", status.Path)
		if !showClean {
			fmt.Println("   ✓ All branches clean")
			fmt.Println()
		}
		return
	}

	fmt.Printf("📁 %s\n", status.Path)

	hasDirty := false
	for _, branch := range status.Branches {
		if branch.IsDirty {
			hasDirty = true
			break
		}
	}

	if !hasDirty && !showClean {
		fmt.Println("   ✓ All branches clean")
		fmt.Println()
		return
	}

	for _, branch := range status.Branches {
		var icon string
		if branch.IsDirty {
			icon = "⚠️ "
		} else {
			icon = "✓ "
		}

		branchName := branch.Name
		if branch.Current {
			branchName = fmt.Sprintf("%s *", branchName)
		}

		fmt.Printf("   %s %s", icon, branchName)

		// Add ahead/behind indicators
		if branch.Ahead > 0 || branch.Behind > 0 {
			fmt.Printf(" [")
			if branch.Ahead > 0 {
				fmt.Printf("↑%d", branch.Ahead)
			}
			if branch.Behind > 0 {
				if branch.Ahead > 0 {
					fmt.Print(" ")
				}
				fmt.Printf("↓%d", branch.Behind)
			}
			fmt.Print("]")
		}

		fmt.Printf(" - %s\n", branch.Status)
	}

	fmt.Println()
}

func displayJSONOutput(statuses []RepoStatus) {
	// Simple JSON output for scripting
	fmt.Println("[")
	for i, status := range statuses {
		fmt.Printf("  {\n")
		fmt.Printf("    \"path\": %q,\n", status.Path)
		fmt.Printf("    \"current_branch\": %q,\n", status.CurrentBranch)
		if status.Error != "" {
			fmt.Printf("    \"error\": %q,\n", status.Error)
		}
		fmt.Printf("    \"branches\": [\n")
		for j, branch := range status.Branches {
			fmt.Printf("      {\n")
			fmt.Printf("        \"name\": %q,\n", branch.Name)
			fmt.Printf("        \"current\": %v,\n", branch.Current)
			fmt.Printf("        \"dirty\": %v,\n", branch.IsDirty)
			fmt.Printf("        \"ahead\": %d,\n", branch.Ahead)
			fmt.Printf("        \"behind\": %d,\n", branch.Behind)
			fmt.Printf("        \"status\": %q\n", branch.Status)
			if j < len(status.Branches)-1 {
				fmt.Printf("      },\n")
			} else {
				fmt.Printf("      }\n")
			}
		}
		fmt.Printf("    ]\n")
		if i < len(statuses)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
