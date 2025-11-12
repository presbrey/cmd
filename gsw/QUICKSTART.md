# Quick Start Guide

## Installation

### Option 1: Build from source

```bash
# Clone the repository
git clone <repository-url>
cd git-status-walker

# Build the binary
make build

# The binary will be in ./bin/git-status-walker
```

### Option 2: Install to GOPATH

```bash
# Clone and install
git clone <repository-url>
cd git-status-walker
make install

# Now you can run from anywhere
git-status-walker
```

## Common Usage Patterns

### Check your current workspace
```bash
cd ~/my-projects
git-status-walker
```

### Check a specific directory
```bash
git-status-walker -dir ~/code/work-projects
```

### See everything (including clean branches)
```bash
git-status-walker -show-clean
```

### Get detailed information
```bash
git-status-walker -verbose -show-clean
```

## Quick Tips

1. **Add to your daily routine**: Run this at the start of each day to see what work is in progress
2. **Use before standup**: Quick summary of what you've been working on
3. **Pre-vacation check**: Make sure nothing is left uncommitted
4. **Create an alias**: Add to your `.bashrc` or `.zshrc`:
   ```bash
   alias gitscan='git-status-walker'
   alias gitscan-all='git-status-walker -show-clean'
   ```

## Understanding the Output

When you run the tool, you'll see:

```
Found 3 git repositories:

📁 /path/to/repo1
   ⚠️  my-feature [↑2] - 3 modified, 1 untracked
   
📁 /path/to/repo2
   ✓ All branches clean
   
📁 /path/to/repo3
   ⚠️  main [↓1] - 1 modified
```

**What this means:**
- `📁` marks each repository found
- `⚠️` indicates a dirty branch (has uncommitted changes)
- `✓` indicates everything is clean
- `[↑2]` means the branch is 2 commits ahead of its upstream
- `[↓1]` means the branch is 1 commit behind its upstream
- The change summary shows what files are affected

## Next Steps

- Check out [EXAMPLES.md](EXAMPLES.md) for more detailed output examples
- Read [README.md](README.md) for complete documentation
- Customize with flags to fit your workflow

## Troubleshooting

**"No git repositories found"**
- Make sure you're in the right directory
- Try increasing `-max-depth` (default is 10)
- Use `-verbose` to see what's being scanned

**"Error checking out branch"**
- The tool needs to checkout each branch to analyze it
- Make sure you don't have important uncommitted work
- Or run with `-verbose` to see which repo/branch is causing issues

**Tool runs slowly**
- Reduce `-max-depth` to limit how deep it searches
- Exclude large directories like `node_modules` (already skipped by default)
- Future versions may add parallel processing for speed
