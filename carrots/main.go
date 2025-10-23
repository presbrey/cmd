package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	userAgent     = "carrots/1.0"
)

type Config struct {
	Owner  string
	Repo   string
	Branch string
	Token  string
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

type Comment struct {
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

func main() {
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub personal access token")
	dir := flag.String("dir", ".", "Git repository directory")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "Error: GitHub token required. Set GITHUB_TOKEN env var or use -token flag")
		os.Exit(1)
	}

	config, err := getRepoConfig(*dir, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository: %s/%s\n", config.Owner, config.Repo)
	fmt.Printf("Branch: %s\n\n", config.Branch)

	pr, err := findPRForBranch(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding PR: %v\n", err)
		os.Exit(1)
	}

	if pr == nil {
		fmt.Println("No open PR found for this branch")
		os.Exit(0)
	}

	fmt.Printf("Found PR #%d: %s\n\n", pr.Number, pr.Title)

	prompts, err := extractAIPrompts(config, pr.Number)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting prompts: %v\n", err)
		os.Exit(1)
	}

	if len(prompts) == 0 {
		fmt.Println("No CodeRabbitAI prompts found in this PR")
		os.Exit(0)
	}

	fmt.Printf("Found %d AI prompt(s):\n\n", len(prompts))
	for i, prompt := range prompts {
		fmt.Printf("=== Prompt %d ===\n%s\n\n", i+1, prompt)
	}
}

func getRepoConfig(dir, token string) (*Config, error) {
	// Get current branch
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Get remote URL
	cmd = exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	remoteOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL: %w", err)
	}
	remoteURL := strings.TrimSpace(string(remoteOutput))

	// Parse owner and repo from URL
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return nil, err
	}

	return &Config{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Token:  token,
	}, nil
}

func parseGitHubURL(url string) (owner, repo string, err error) {
	// Handle HTTPS URLs: https://github.com/owner/repo.git
	httpsRegex := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/]+?)(\.git)?$`)
	matches := httpsRegex.FindStringSubmatch(url)
	if len(matches) >= 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("unable to parse GitHub URL: %s", url)
}

func findPRForBranch(config *Config) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?head=%s:%s&state=open",
		githubAPIBase, config.Owner, config.Repo, config.Owner, config.Branch)

	body, err := makeGitHubRequest(url, config.Token)
	if err != nil {
		return nil, err
	}

	var prs []PullRequest
	if err := json.Unmarshal(body, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil
	}

	return &prs[0], nil
}

func extractAIPrompts(config *Config, prNumber int) ([]string, error) {
	// Get PR comments
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	body, err := makeGitHubRequest(url, config.Token)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	// Also get review comments
	reviewURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	reviewBody, err := makeGitHubRequest(reviewURL, config.Token)
	if err != nil {
		return nil, err
	}

	var reviewComments []Comment
	if err := json.Unmarshal(reviewBody, &reviewComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	// Combine all comments
	allComments := append(comments, reviewComments...)

	var prompts []string
	promptRegex := regexp.MustCompile(`(?s)Prompt for AI Agents.*?\n\s*\x60\x60\x60[^\n]*\n(.*?)\n\s*\x60\x60\x60`)

	for _, comment := range allComments {
		// Check if comment is from coderabbitai bot
		if comment.User.Login != "coderabbitai" && comment.User.Type != "Bot" {
			continue
		}

		// Extract prompts from comment body
		matches := promptRegex.FindAllStringSubmatch(comment.Body, -1)
		for _, match := range matches {
			if len(match) > 1 {
				prompts = append(prompts, strings.TrimSpace(match[1]))
			}
		}
	}

	return prompts, nil
}

func makeGitHubRequest(url, token string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}
