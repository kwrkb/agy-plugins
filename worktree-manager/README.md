# worktree-manager — Git Worktree Management Plugin

An MCP server providing safe, structured git worktree operations (`list`, `add`, `remove`, `prune`) for agy. Enables seamless parallel development, isolated subagent workspaces, and branch exploration without stashing uncommitted changes.

## Provided Tools

### `worktree_list`
Lists all git worktrees in the repository with details including branch, HEAD commit, lock status, and main worktree identification.

#### Parameters
- `repo_path` (string, optional): Target repository path (default: `.`).

### `worktree_add`
Creates a new worktree.

#### Parameters
- `path` (string, required): Directory path for the new worktree.
- `branch` (string, optional): Existing branch to check out.
- `new_branch` (string, optional): New branch name to create and check out (`-b <new_branch>`).
- `commit` (string, optional): Base commit, branch, or tag to branch off.
- `repo_path` (string, optional): Target repository path.

### `worktree_remove`
Safely removes a linked worktree. Refuses to delete the main working tree.

#### Parameters
- `path` (string, required): Path of the worktree to remove.
- `force` (boolean, optional, default: false): Force removal even if untracked/uncommitted changes exist.
- `repo_path` (string, optional): Target repository path.

### `worktree_prune`
Cleans up administrative metadata for removed or moved worktrees.

#### Parameters
- `dry_run` (boolean, optional, default: false): Only display what would be pruned without deleting.
- `repo_path` (string, optional): Target repository path.

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/worktree-manager
```

## Bundled Skill

Includes `skills/worktree-manager-gemini/SKILL.md` to guide agents in managing isolated worktrees during parallel workflows or subagent delegation.
