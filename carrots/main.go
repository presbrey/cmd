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

	"github.com/presbrey/pkg/envtree"
)

func init() {
	envtree.AutoLoad()
}

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
	Body                string    `json:"body"`
	User                User      `json:"user"`
	CreatedAt           time.Time `json:"created_at"`
	PullRequestReviewID *int      `json:"pull_request_review_id,omitempty"`
	InReplyToID         *int      `json:"in_reply_to_id,omitempty"`
	SubjectType         string    `json:"subject_type,omitempty"`
}

type ReviewThread struct {
	ID       int       `json:"id"`
	Comments []Comment `json:"comments"`
	Resolved bool      `json:"resolved"`
}

type User struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

func getEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	// Get defaults from environment variables with CARROTS_ prefix
	defaultToken := getEnvOrDefault("CARROTS_TOKEN", os.Getenv("GITHUB_TOKEN"))
	defaultDir := getEnvOrDefault("CARROTS_DIR", ".")
	defaultOutput := getEnvOrDefault("CARROTS_OUTPUT", "CARROTS.md")

	token := flag.String("token", defaultToken, "GitHub personal access token")
	dir := flag.String("dir", defaultDir, "Git repository directory")
	output := flag.String("output", defaultOutput, "Output file (default: CARROTS.md)")
	includeResolved := flag.Bool("include-resolved", false, "Include prompts from resolved comment threads")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "Error: GitHub token required. Set CARROTS_TOKEN or GITHUB_TOKEN env var or use -token flag")
		os.Exit(1)
	}

	// Set up output writer
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	outputWriter := file

	config, err := getRepoConfig(*dir, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(outputWriter, "Repository: %s/%s\n", config.Owner, config.Repo)
	fmt.Fprintf(outputWriter, "Branch: %s\n\n", config.Branch)

	pr, err := findPRForBranch(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding PR: %v\n", err)
		os.Exit(1)
	}

	if pr == nil {
		fmt.Fprintln(outputWriter, "No open PR found for this branch")
		os.Exit(0)
	}

	fmt.Fprintf(outputWriter, "Found PR #%d: %s\n\n", pr.Number, pr.Title)

	prompts, err := extractAIPrompts(config, pr.Number, *includeResolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting prompts: %v\n", err)
		os.Exit(1)
	}

	if len(prompts) == 0 {
		fmt.Fprintln(outputWriter, "No CodeRabbitAI prompts found in this PR")
		os.Exit(0)
	}

	fmt.Fprintf(outputWriter, "Found %d AI prompt(s):\n\n", len(prompts))
	for i, prompt := range prompts {
		fmt.Fprintf(outputWriter, "=== Prompt %d ===\n%s\n\n", i+1, prompt)
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

func getResolvedThreadIDs(config *Config, prNumber int) (map[int]bool, error) {
	// Fetch review threads using the conversations API
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	// Use the comfort-fade preview to get conversation/resolution info
	body, err := makeGitHubRequestWithAccept(url, config.Token,
		"application/vnd.github.comfort-fade-preview+json")
	if err != nil {
		return nil, err
	}

	var reviewComments []struct {
		ID           int    `json:"id"`
		InReplyToID  *int   `json:"in_reply_to_id"`
		SubjectType  string `json:"subject_type"`
		Line         *int   `json:"line"`
		OriginalLine *int   `json:"original_line"`
	}

	if err := json.Unmarshal(body, &reviewComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments for resolution: %w", err)
	}

	// Build a map of thread root IDs to track which are resolved
	// GitHub marks the subject_type as "line" for unresolved and "resolved_line" or doesn't include resolved info
	// We'll use a different approach: fetch actual review threads

	threadsURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	threadsBody, err := makeGitHubRequestWithAccept(threadsURL, config.Token,
		"application/vnd.github.v3+json")
	if err != nil {
		return nil, err
	}

	var allReviewComments []struct {
		ID                  int    `json:"id"`
		PullRequestReviewID *int   `json:"pull_request_review_id"`
		InReplyToID         *int   `json:"in_reply_to_id"`
		Body                string `json:"body"`
	}

	if err := json.Unmarshal(threadsBody, &allReviewComments); err != nil {
		return nil, fmt.Errorf("failed to parse review threads: %w", err)
	}

	// Build map of resolved comment IDs
	// For now, we'll check if a comment body contains resolution markers
	// A better approach would be to use GraphQL, but this works with REST API
	resolvedIDs := make(map[int]bool)

	for _, comment := range allReviewComments {
		// Check if this comment or its thread contains resolution markers
		// GitHub doesn't expose resolution status directly in REST API v3
		// so we look for the conversation being marked as resolved in replies
		if strings.Contains(strings.ToLower(comment.Body), "marked this conversation as resolved") ||
			strings.Contains(strings.ToLower(comment.Body), "resolved this conversation") {
			// Mark this comment and its thread as resolved
			resolvedIDs[comment.ID] = true
			if comment.InReplyToID != nil {
				resolvedIDs[*comment.InReplyToID] = true
			}
			// Mark all comments in the same thread
			threadRoot := comment.ID
			if comment.InReplyToID != nil {
				threadRoot = *comment.InReplyToID
			}
			for _, c := range allReviewComments {
				if c.InReplyToID != nil && *c.InReplyToID == threadRoot {
					resolvedIDs[c.ID] = true
				}
				if c.ID == threadRoot {
					resolvedIDs[c.ID] = true
				}
			}
		}
	}

	return resolvedIDs, nil
}

func extractAIPrompts(config *Config, prNumber int, includeResolved bool) ([]string, error) {
	// Get resolved thread IDs (only if we need to skip them)
	var resolvedThreads map[int]bool
	if !includeResolved {
		var err error
		resolvedThreads, err = getResolvedThreadIDs(config, prNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get resolved threads: %w", err)
		}
	}

	// Get PR comments (issue comments - not part of code review threads)
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

	var reviewComments []struct {
		ID                  int       `json:"id"`
		Body                string    `json:"body"`
		User                User      `json:"user"`
		CreatedAt           time.Time `json:"created_at"`
		PullRequestReviewID *int      `json:"pull_request_review_id"`
		InReplyToID         *int      `json:"in_reply_to_id"`
	}
	if err := json.Unmarshal(reviewBody, &reviewComments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	var prompts []string
	promptRegex := regexp.MustCompile(`(?s)Prompt for AI Agents.*?\n\s*\x60\x60\x60[^\n]*\n(.*?)\n\s*\x60\x60\x60`)

	// Process issue comments (these are never part of resolved threads)
	for _, comment := range comments {
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

	// Process review comments, filtering out resolved threads if requested
	for _, comment := range reviewComments {
		// Check if comment is from coderabbitai bot
		if comment.User.Login != "coderabbitai" && comment.User.Type != "Bot" {
			continue
		}

		// Skip if this comment is part of a resolved thread (only if not including resolved)
		if !includeResolved && resolvedThreads[comment.ID] {
			continue
		}

		// Also skip if it's a reply to a resolved comment (only if not including resolved)
		if !includeResolved && comment.InReplyToID != nil && resolvedThreads[*comment.InReplyToID] {
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
	return makeGitHubRequestWithAccept(url, token, "application/vnd.github.v3+json")
}

func makeGitHubRequestWithAccept(url, token, acceptHeader string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", acceptHeader)
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
