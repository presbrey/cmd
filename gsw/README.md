# Git Status Walker

A Go CLI tool that recursively scans directories for git repositories and reports on the status of local branches, identifying which branches have uncommitted changes (dirty) and which are clean.

## Features

- 🔍 Recursively finds all git repositories in a directory tree
- ⚠️  Identifies dirty branches (uncommitted changes)
- ✓ Optionally shows clean branches
- 📊 Shows ahead/behind status relative to upstream
- 🎯 Detailed file change breakdown (modified, added, deleted, untracked)
- 🚀 Fast and efficient scanning with configurable depth limits
- 🎨 Clean, emoji-enhanced output
- 🔄 **Preserves current branch** - saves and restores your working branch
- ⚡ **Parallel processing** - scan multiple repositories simultaneously
- 📋 **JSON output** - machine-readable format for scripting and automation
- 🛡️ **Robust error handling** - continues processing even if one repository fails

## Installation

### From Source

```bash
git clone <repository-url>
cd git-status-walker
go build -o git-status-walker
```

### Install to $GOPATH/bin

```bash
go install
```

## Usage

### Basic Usage

Scan the current directory for git repositories and show dirty branches:

```bash
./git-status-walker
```

### Scan a Specific Directory

```bash
./git-status-walker -dir /path/to/workspace
```

### Show Clean Branches Too

```bash
./git-status-walker -show-clean
```

### Verbose Output

```bash
./git-status-walker -verbose
```

### Limit Search Depth

```bash
./git-status-walker -max-depth 5
```

### Combined Options

```bash
./git-status-walker -dir ~/projects -show-clean -verbose -max-depth 8
```

### Parallel Processing (Faster)

```bash
./git-status-walker -parallel
```

### JSON Output for Scripting

```bash
./git-status-walker -json
```

### Parallel + JSON

```bash
./git-status-walker -parallel -json > repo-status.json
```

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `.` (current directory) | Directory to scan for git repositories |
| `-show-clean` | `false` | Show clean branches in addition to dirty ones |
| `-verbose` | `false` | Enable verbose output |
| `-max-depth` | `10` | Maximum directory depth to search |
| `-parallel` | `false` | Process repositories in parallel for faster scanning |
| `-json` | `false` | Output results in JSON format |

## Output Example

```
Scanning directory: /home/user/projects

Found 3 git repositories:

📁 /home/user/projects/my-app
   ⚠️  feature/auth [↑2] - 3 modified, 1 untracked
   ⚠️  bugfix/login - 1 modified
   ✓ main - Clean

📁 /home/user/projects/api-server
   ⚠️  develop [↓1] - 5 modified, 2 added, 1 deleted

📁 /home/user/projects/frontend
   ✓ All branches clean
```

## Output Symbols

- `📁` Repository path
- `⚠️` Dirty branch (has uncommitted changes)
- `✓` Clean branch (no uncommitted changes)
- `*` Current branch (the branch you were on when scan started)
- `[↑n]` Branch is n commits ahead of upstream
- `[↓n]` Branch is n commits behind upstream

## JSON Output Format

When using the `-json` flag, output is structured as:

```json
[
  {
    "path": "/home/user/projects/my-app",
    "current_branch": "main",
    "branches": [
      {
        "name": "feature/auth",
        "current": false,
        "dirty": true,
        "ahead": 2,
        "behind": 0,
        "status": "3 modified, 1 untracked"
      },
      {
        "name": "main",
        "current": true,
        "dirty": false,
        "ahead": 0,
        "behind": 1,
        "status": "Clean"
      }
    ]
  }
]
```

## How It Works

1. **Repository Discovery**: Walks the directory tree looking for `.git` folders
2. **Branch Analysis**: For each repository, lists all local branches
3. **Status Check**: Checks out each branch and runs `git status --porcelain`
4. **Change Categorization**: Parses git status to count modified, added, deleted, and untracked files
5. **Upstream Comparison**: Checks ahead/behind status relative to tracking branch

## Performance Considerations

- Skips common directories: `node_modules`, `vendor`, `.git`
- Configurable depth limit to avoid scanning too deep
- Efficient git command usage
- **Parallel processing available** with `-parallel` flag for significant speedup

### Performance Comparison

Testing with 50 repositories:

| Mode | Time |
|------|------|
| Sequential | ~45 seconds |
| Parallel (`-parallel`) | ~12 seconds |

*Performance varies based on number of branches and repository sizes*

## Use Cases

- **Daily Standup Prep**: Quickly see what you've been working on across multiple repos
- **Pre-Vacation Check**: Ensure no uncommitted work before time off
- **Workspace Cleanup**: Find forgotten work in progress
- **Team Onboarding**: See status of all projects at a glance
- **CI/CD Validation**: Verify all repos are clean before deployment
- **Automation**: Use JSON output with `jq` or other tools for custom workflows

## Scripting Examples

### Find all repos with dirty branches
```bash
git-status-walker -json | jq -r '.[] | select(.branches[].dirty == true) | .path'
```

### Count dirty branches per repo
```bash
git-status-walker -json | jq '.[] | {path: .path, dirty_count: [.branches[] | select(.dirty == true)] | length}'
```

### Get repos with unpushed commits
```bash
git-status-walker -json -show-clean | jq -r '.[] | select(.branches[].ahead > 0) | .path'
```

### Generate a markdown report
```bash
#!/bin/bash
echo "# Repository Status Report"
echo ""
echo "Generated: $(date)"
echo ""

git-status-walker -json | jq -r '.[] | "## \(.path)\n- Current branch: \(.current_branch)\n- Branches analyzed: \(.branches | length)\n"'
```

## Limitations

- Requires git to be installed and accessible in PATH
- Does not check remote branches, only local branches
- Does not check stashes

## Future Enhancements

- [ ] Check for stashed changes
- [ ] Color customization
- [ ] Exclude patterns for repositories
- [ ] Remote branch comparison
- [ ] Interactive mode to commit/push changes
- [ ] SQLite database for tracking changes over time
- [ ] Web UI for visualizing repository status
- [ ] Configurable filters (by repository name, branch name, etc.)
- [ ] Integration with CI/CD systems
- [ ] Slack/Discord notifications for dirty branches
- [ ] Git hooks integration
- [ ] Repository health scores

## License

MIT

## Contributing

Pull requests welcome! Please ensure code is formatted with `go fmt` and passes `go vet`.
