package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/presbrey/pkg/envtree"
)

func init() {
	envtree.AutoLoad()
}

const (
	githubAPIBase = "https://api.github.com"
	userAgent     = "carrots/1.0"
)

var debugMode bool

// Config holds environment-based configuration
type Config struct {
	Debug  bool   `env:"DEBUG"                       envDefault:"false"`
	Dir    string `env:"DIR"                         envDefault:"."`
	Token  string `env:"TOKEN,required"              envDefault:""`
	Output string `env:"OUTPUT"                      envDefault:"CARROTS.md"`

	IncludeResolved bool `env:"INCLUDE_RESOLVED"            envDefault:"false"`

	// These are populated from git, not environment
	Owner  string `env:"-"`
	Repo   string `env:"-"`
	Branch string `env:"-"`
}

var cfg *Config

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

func main() {
	cfg = &Config{}

	// Parse environment variables with CARROTS_ prefix
	// Also check GITHUB_TOKEN as fallback for TOKEN
	if os.Getenv("CARROTS_TOKEN") == "" && os.Getenv("GITHUB_TOKEN") != "" {
		os.Setenv("CARROTS_TOKEN", os.Getenv("GITHUB_TOKEN"))
	}

	if err := env.ParseWithOptions(cfg, env.Options{Prefix: "CARROTS_"}); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Required: CARROTS_TOKEN or GITHUB_TOKEN")
		os.Exit(1)
	}

	debugMode = cfg.Debug

	// Set up output writer
	file, err := os.Create(cfg.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	outputWriter := file

	if err := populateRepoConfig(cfg.Dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(outputWriter, "Repository: %s/%s\n", cfg.Owner, cfg.Repo)
	fmt.Fprintf(outputWriter, "Branch: %s\n\n", cfg.Branch)

	pr, err := findPRForBranch(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding PR: %v\n", err)
		os.Exit(1)
	}

	if pr == nil {
		fmt.Fprintln(outputWriter, "No open PR found for this branch")
		os.Exit(0)
	}

	fmt.Fprintf(outputWriter, "Found PR #%d: %s\n\n", pr.Number, pr.Title)

	prompts, err := extractAIPrompts(cfg, pr.Number, cfg.IncludeResolved)
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

func populateRepoConfig(dir string) error {
	// Get current branch
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	cfg.Branch = strings.TrimSpace(string(branchOutput))

	// Get remote URL
	cmd = exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	remoteOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}
	remoteURL := strings.TrimSpace(string(remoteOutput))

	// Parse owner and repo from URL
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return err
	}

	cfg.Owner = owner
	cfg.Repo = repo
	return nil
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
	// Fetch all review comments using pagination
	threadsURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	var allReviewComments []struct {
		ID                  int    `json:"id"`
		PullRequestReviewID *int   `json:"pull_request_review_id"`
		InReplyToID         *int   `json:"in_reply_to_id"`
		Body                string `json:"body"`
	}

	// Iterate through all pages
	for body, err := range iterGitHubPages(threadsURL, config.Token, "application/vnd.github.v3+json") {
		if err != nil {
			return nil, err
		}

		var pageComments []struct {
			ID                  int    `json:"id"`
			PullRequestReviewID *int   `json:"pull_request_review_id"`
			InReplyToID         *int   `json:"in_reply_to_id"`
			Body                string `json:"body"`
		}
		if err := json.Unmarshal(body, &pageComments); err != nil {
			return nil, fmt.Errorf("failed to parse review threads: %w", err)
		}
		allReviewComments = append(allReviewComments, pageComments...)
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

	var prompts []string
	promptRegex := regexp.MustCompile(`(?s)Prompt for AI Agents.*?\n\s*\x60\x60\x60[^\n]*\n(.*?)\n\s*\x60\x60\x60`)

	// Get PR comments (issue comments - not part of code review threads) with pagination
	issueCommentsURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	for body, err := range iterGitHubPages(issueCommentsURL, config.Token, "application/vnd.github.v3+json") {
		if err != nil {
			return nil, err
		}

		var comments []Comment
		if err := json.Unmarshal(body, &comments); err != nil {
			return nil, fmt.Errorf("failed to parse comments: %w", err)
		}

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
	}

	// Get review comments with pagination
	reviewURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments",
		githubAPIBase, config.Owner, config.Repo, prNumber)

	for body, err := range iterGitHubPages(reviewURL, config.Token, "application/vnd.github.v3+json") {
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
		if err := json.Unmarshal(body, &reviewComments); err != nil {
			return nil, fmt.Errorf("failed to parse review comments: %w", err)
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
	}

	return prompts, nil
}

func makeGitHubRequest(url, token string) ([]byte, error) {
	body, _, err := makeGitHubRequestWithAccept(url, token, "application/vnd.github.v3+json")
	return body, err
}

// iterGitHubPages returns an iterator that yields each page of results from a paginated GitHub API endpoint.
// It automatically adds per_page=100 and follows Link headers.
func iterGitHubPages(baseURL, token, acceptHeader string) func(yield func([]byte, error) bool) {
	return func(yield func([]byte, error) bool) {
		// Add per_page=100 to the URL
		url := baseURL
		if strings.Contains(url, "?") {
			url += "&per_page=100"
		} else {
			url += "?per_page=100"
		}

		for url != "" {
			body, nextURL, err := makeGitHubRequestWithAccept(url, token, acceptHeader)
			if !yield(body, err) {
				return
			}
			if err != nil {
				return
			}
			url = nextURL
		}
	}
}

// parseNextLink extracts the "next" URL from a GitHub Link header.
// Example: <https://api.github.com/...?page=2>; rel="next", <https://...>; rel="last"
func parseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}

	// Parse Link header format: <url>; rel="next", <url>; rel="last"
	linkRegex := regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
	matches := linkRegex.FindStringSubmatch(linkHeader)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func makeGitHubRequestWithAccept(url, token, acceptHeader string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("User-Agent", userAgent)

	if debugMode {
		fmt.Fprintf(os.Stderr, "\n=== API REQUEST ===\n")
		fmt.Fprintf(os.Stderr, "Method: %s\n", req.Method)
		fmt.Fprintf(os.Stderr, "URL: %s\n", url)
		fmt.Fprintf(os.Stderr, "Headers:\n")
		for k, v := range req.Header {
			// Redact the token for security
			if k == "Authorization" {
				fmt.Fprintf(os.Stderr, "  %s: Bearer [REDACTED]\n", k)
			} else {
				fmt.Fprintf(os.Stderr, "  %s: %s\n", k, strings.Join(v, ", "))
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	// Extract next page URL from Link header
	nextURL := parseNextLink(resp.Header.Get("Link"))

	if debugMode {
		fmt.Fprintf(os.Stderr, "=== API RESPONSE ===\n")
		fmt.Fprintf(os.Stderr, "Status: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Fprintf(os.Stderr, "Headers:\n")
		for k, v := range resp.Header {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", k, strings.Join(v, ", "))
		}
		if nextURL != "" {
			fmt.Fprintf(os.Stderr, "Next Page: %s\n", nextURL)
		}
		fmt.Fprintf(os.Stderr, "\nBody:\n")

		// Try to pretty print JSON
		var prettyJSON interface{}
		if err := json.Unmarshal(body, &prettyJSON); err == nil {
			prettyBody, err := json.MarshalIndent(prettyJSON, "", "  ")
			if err == nil {
				fmt.Fprintf(os.Stderr, "%s\n", string(prettyBody))
			} else {
				fmt.Fprintf(os.Stderr, "%s\n", string(body))
			}
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", string(body))
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nextURL, nil
}
