# Example Output

## Scenario 1: Basic usage (only dirty branches shown)

```bash
$ ./git-status-walker -dir ~/workspace
```

```
Found 4 git repositories:

📁 /home/user/workspace/api-gateway
   ⚠️  feature/rate-limiting [↑3] - 4 modified, 2 untracked
   ⚠️  hotfix/cors - 1 modified

📁 /home/user/workspace/frontend
   ⚠️  develop [↑1 ↓2] - 8 modified, 3 added, 1 deleted, 5 untracked

📁 /home/user/workspace/mobile-app
   ✓ All branches clean

📁 /home/user/workspace/infrastructure
   ⚠️  main - 2 modified
```

## Scenario 2: Show all branches (clean and dirty)

```bash
$ ./git-status-walker -show-clean
```

```
Found 4 git repositories:

📁 /home/user/workspace/api-gateway
   ⚠️  feature/rate-limiting [↑3] - 4 modified, 2 untracked
   ⚠️  hotfix/cors - 1 modified
   ✓ main - Clean
   ✓ develop [↓1] - Clean

📁 /home/user/workspace/frontend
   ⚠️  develop [↑1 ↓2] - 8 modified, 3 added, 1 deleted, 5 untracked
   ✓ main - Clean

📁 /home/user/workspace/mobile-app
   ✓ main - Clean
   ✓ develop - Clean
   ✓ feature/push-notifications - Clean

📁 /home/user/workspace/infrastructure
   ⚠️  main - 2 modified
   ✓ staging - Clean
   ✓ production - Clean
```

## Scenario 3: Verbose mode

```bash
$ ./git-status-walker -dir ~/workspace -verbose
```

```
Scanning directory: /home/user/workspace
Show clean branches: false

Found repository: /home/user/workspace/api-gateway
Found repository: /home/user/workspace/frontend
Found repository: /home/user/workspace/mobile-app
Found repository: /home/user/workspace/infrastructure

Found 4 git repositories:

📁 /home/user/workspace/api-gateway
   ⚠️  feature/rate-limiting [↑3] - 4 modified, 2 untracked
   ⚠️  hotfix/cors - 1 modified

📁 /home/user/workspace/frontend
   ⚠️  develop [↑1 ↓2] - 8 modified, 3 added, 1 deleted, 5 untracked

📁 /home/user/workspace/mobile-app
   ✓ All branches clean

📁 /home/user/workspace/infrastructure
   ⚠️  main - 2 modified
```

## Scenario 4: No repositories found

```bash
$ ./git-status-walker -dir ~/empty-folder
```

```
No git repositories found.
```

## Scenario 5: Limited depth search

```bash
$ ./git-status-walker -dir ~/deep-workspace -max-depth 3
```

```
Found 2 git repositories:

📁 /home/user/deep-workspace/level1/project-a
   ⚠️  develop - 3 modified

📁 /home/user/deep-workspace/level1/level2/project-b
   ✓ All branches clean
```

## Understanding the Output

### Status Indicators
- `⚠️` - Branch has uncommitted changes
- `✓` - Branch is clean (or repository has no dirty branches)

### Tracking Status
- `[↑3]` - Branch is 3 commits ahead of upstream
- `[↓2]` - Branch is 2 commits behind upstream
- `[↑1 ↓2]` - Branch is 1 ahead and 2 behind upstream

### Change Details
- `X modified` - Files with changes
- `X added` - New files staged for commit
- `X deleted` - Files deleted
- `X untracked` - New files not yet added to git
