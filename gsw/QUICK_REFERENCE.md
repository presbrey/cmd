# Git Status Walker - Quick Reference Card

## Installation

```bash
cd git-status-walker
make build          # Creates ./bin/git-status-walker
make install        # Installs to $GOPATH/bin
```

## Most Common Commands

```bash
# Scan current directory
git-status-walker

# Scan specific directory
git-status-walker -dir ~/projects

# Show all branches (clean + dirty)
git-status-walker -show-clean

# Verbose output
git-status-walker -verbose

# Limit search depth
git-status-walker -max-depth 5
```

## Enhanced Version Only

```bash
# Build enhanced version
go build -o bin/git-status-walker main-enhanced.go

# Parallel processing (faster!)
git-status-walker -parallel

# JSON output for scripting
git-status-walker -json

# Combined
git-status-walker -parallel -json -show-clean
```

## Useful Aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias gs='git-status-walker'
alias gsa='git-status-walker -show-clean'
alias gsv='git-status-walker -verbose -show-clean'
alias gsj='git-status-walker -json'
```

## JSON Queries (using jq)

```bash
# Find dirty repos
git-status-walker -json | jq '.[] | select(.branches[].dirty)'

# List all repo paths
git-status-walker -json | jq -r '.[].path'

# Find unpushed commits
git-status-walker -json | jq '.[] | select(.branches[].ahead > 0)'

# Count branches per repo
git-status-walker -json | jq '.[] | {path, count: .branches|length}'
```

## Output Symbols

| Symbol | Meaning |
|--------|---------|
| 📁 | Repository |
| ⚠️ | Dirty branch |
| ✓ | Clean |
| [↑3] | 3 commits ahead |
| [↓2] | 2 commits behind |
| * | Current branch (enhanced) |

## Common Workflows

### Morning Routine
```bash
cd ~/work
git-status-walker
```

### Pre-Standup
```bash
git-status-walker -dir ~/projects -verbose
```

### Pre-Vacation
```bash
git-status-walker -show-clean > pre-vacation-status.txt
```

### Find All Dirty Repos
```bash
git-status-walker | grep ⚠️ -B1
```

## File Locations

| File | Purpose |
|------|---------|
| `main.go` | Basic version |
| `main-enhanced.go` | Advanced features |
| `README.md` | Full documentation |
| `QUICKSTART.md` | Getting started |
| `EXAMPLES.md` | Output examples |
| `ENHANCED.md` | Enhanced features |

## Build Targets

```bash
make build       # Build the tool
make install     # Install globally
make clean       # Remove artifacts
make run         # Build and run
make run-all     # Run with -show-clean
make fmt         # Format code
make vet         # Run go vet
make lint        # Format + vet
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| No repos found | Check directory, increase `-max-depth` |
| Slow scanning | Use `-parallel` (enhanced version) |
| Branch errors | Use enhanced version (preserves state) |

## Quick Examples

**Basic scan:**
```bash
git-status-walker
```

**Scan work directory:**
```bash
git-status-walker -dir ~/work
```

**Fast parallel scan:**
```bash
git-status-walker -parallel -dir ~/large-workspace
```

**JSON for scripting:**
```bash
git-status-walker -json | jq '.[] | select(.branches[].dirty == true) | .path'
```

**Complete analysis:**
```bash
git-status-walker -verbose -show-clean -max-depth 15
```

---

💡 **Tip:** Bookmark this file for quick access to common commands!
