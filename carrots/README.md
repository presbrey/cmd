# CARROTS

**C**odeRabbitAI **A**gent **R**eview **R**etrieval and **O**utput **T**ool for **S**ummaries

A Go tool to extract "Prompt for AI Agents" from CodeRabbitAI reviews on GitHub Pull Requests.

## Features

- Automatically detects the current branch and finds associated open PRs
- Extracts all AI prompts from CodeRabbitAI bot comments
- Works with both issue comments and review comments
- Simple CLI interface

## Installation

```bash
go build -o carrots carrots.go
```

Or install directly:

```bash
go install
```

## Usage

### Prerequisites

You need a GitHub personal access token with `repo` scope:
1. Go to GitHub Settings â†’ Developer settings â†’ Personal access tokens
2. Generate a new token with `repo` permissions
3. Set it as an environment variable:

```bash
export GITHUB_TOKEN="your_token_here"
```

### Basic Usage

Run in your git repository directory:

```bash
./carrots
```

This will:
1. Detect the current branch
2. Find the open PR for that branch
3. Extract all CodeRabbitAI prompts from the PR comments

### Options

```bash
./carrots [flags]

Flags:
  -token string
        GitHub personal access token (defaults to GITHUB_TOKEN env var)
  -dir string
        Git repository directory (default ".")
```

### Examples

Extract prompts from current directory:
```bash
./carrots
```

Specify a different repository directory:
```bash
./carrots -dir /path/to/repo
```

Use a specific token:
```bash
./carrots -token ghp_yourtoken
```

## Output Example

```
Repository: owner/repo
Branch: feature-branch

Found PR #123: Add new feature

Found 2 AI prompt(s):

=== Prompt 1 ===
In CLAUDE.md around lines 151 to 156, the documentation is missing security
guidance: add notes to (1) require sanitizing and normalizing any URL paths or
filenames derived from user input before writing to ./image-replacements/ (e.g.,
reject path traversal, remove ../, enforce a whitelist of characters and a fixed
base directory), (2) restrict downloads to allowed schemes (http/https only),
validate hostnames/resolve against an allowlist or blocklist...

=== Prompt 2 ===
[Additional prompt content...]
```

## How It Works

1. Reads git config to determine repository owner, name, and current branch
2. Queries GitHub API to find open PRs for the current branch
3. Retrieves all comments (both issue and review comments)
4. Filters for comments from the `coderabbitai` bot
5. Extracts text from "Prompt for AI Agents" code blocks using regex

## Project Structure

```
.
â”œâ”€â”€ carrots.go    # Main Go source code
â”œâ”€â”€ go.mod        # Go module definition
â””â”€â”€ CARROTS.md    # This file
```

## Requirements

- Go 1.21 or later
- Git repository with GitHub remote
- GitHub personal access token
- Network access to GitHub API

## Error Handling

The tool will exit with an error if:
- No GitHub token is provided
- Not run in a git repository
- Remote URL is not a GitHub repository
- GitHub API requests fail
- No open PR exists for the current branch

## Why "CARROTS"?

Because bunnies (like CodeRabbitAI) love carrots! ðŸ¥•ðŸ°

## License

MIT