package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/presbrey/cmd/ai-sync-conventions/internal/sync"
)

func main() {
	startPath := flag.String("path", "", "Starting path to search for sync files (defaults to current directory)")
	flag.Parse()

	root, err := sync.FindSyncRoot(*startPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding sync root: %v\n", err)
		os.Exit(1)
	}

	syncManager := sync.NewSyncManager()
	plan, err := syncManager.CreatePlan(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating sync plan: %v\n", err)
		os.Exit(1)
	}

	if len(plan.TargetPaths) == 0 {
		info, err := syncManager.GetFileInfo(plan.SourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("All rules files are equal.\nSize: %d bytes\nMD5: %s\n", info.Size, info.Hash)
		return
	}

	// Show confirmation prompt
	fmt.Printf("Will copy from:\n  %s\n\nTo:\n", plan.SourcePath)
	for _, target := range plan.TargetPaths {
		fmt.Printf("  %s\n", target)
	}
	for {
		fmt.Print("\nProceed? [Y/n] ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
			os.Exit(1)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response == "n" || response == "no" {
			fmt.Println("Operation cancelled")
			os.Exit(0)
		} else if response == "y" || response == "yes" || response == "" {
			break
		} else {
			fmt.Println("Invalid response. Please enter 'y', 'yes', 'n', or 'no'.")
		}
	}

	if err := plan.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Files synchronized successfully")
}
