Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Ensure current directory is inside a Git repository.
.DESCRIPTION
Throws an error when Git metadata is not available for the current path.
.EXAMPLE
_assert_git_repo
#>
function _assert_git_repo {
    _assert_command_available -Name git
    git rev-parse --is-inside-work-tree 1>$null 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Current directory is not a Git repository."
    }
}

<#
.SYNOPSIS
Show repository status.
.DESCRIPTION
Runs `git status`.
.EXAMPLE
dm g_status
#>
function g_status {
    _assert_git_repo
    git status
}

<#
.SYNOPSIS
Show current branch name.
.DESCRIPTION
Prints the checked-out branch.
.EXAMPLE
dm g_branch_current
#>
function g_branch_current {
    _assert_git_repo
    git branch --show-current
}

<#
.SYNOPSIS
List local and remote branches.
.DESCRIPTION
Runs `git branch -a`.
.EXAMPLE
dm g_branch_list
#>
function g_branch_list {
    _assert_git_repo
    git branch -a
}

<#
.SYNOPSIS
Fetch updates from remotes.
.DESCRIPTION
Runs `git fetch --all --prune`.
.EXAMPLE
dm g_fetch
#>
function g_fetch {
    _assert_git_repo
    git fetch --all --prune
}

<#
.SYNOPSIS
Pull updates for current branch.
.DESCRIPTION
Runs `git pull`.
.EXAMPLE
dm g_pull
#>
function g_pull {
    _assert_git_repo
    git pull
}

<#
.SYNOPSIS
Push current branch to remote.
.DESCRIPTION
Runs `git push`.
.EXAMPLE
dm g_push
#>
function g_push {
    _assert_git_repo
    git push
}

<#
.SYNOPSIS
Stage all changes.
.DESCRIPTION
Runs `git add .`.
.EXAMPLE
dm g_add_all
#>
function g_add_all {
    _assert_git_repo
    git add .
}

<#
.SYNOPSIS
Commit staged changes.
.DESCRIPTION
Runs `git commit -m <message>`.
.PARAMETER Message
Commit message.
.EXAMPLE
dm g_commit -Message "Fix parser"
#>
function g_commit {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    _assert_git_repo
    git commit -m "$Message"
}

<#
.SYNOPSIS
Stage all changes and commit.
.DESCRIPTION
Runs `git add .` then `git commit -m <message>`.
.PARAMETER Message
Commit message.
.EXAMPLE
dm g_add_commit -Message "Update docs"
#>
function g_add_commit {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    _assert_git_repo
    git add .
    git commit -m "$Message"
}

<#
.SYNOPSIS
Show recent commit history.
.DESCRIPTION
Runs `git log --decorate --stat -n 20`.
.EXAMPLE
dm g_log
#>
function g_log {
    _assert_git_repo
    git log --decorate --stat -n 20
}

<#
.SYNOPSIS
Show compact graph log.
.DESCRIPTION
Runs `git log --oneline --graph --decorate --all`.
.EXAMPLE
dm g_log_graph_oneline
#>
function g_log_graph_oneline {
    _assert_git_repo
    git log --oneline --graph --decorate --all
}

<#
.SYNOPSIS
Show unstaged and staged diff.
.DESCRIPTION
Runs `git diff` and `git diff --cached`.
.EXAMPLE
dm g_diff
#>
function g_diff {
    _assert_git_repo
    git diff
    git diff --cached
}

<#
.SYNOPSIS
Restore one file from HEAD.
.DESCRIPTION
Runs `git restore <path>`.
.PARAMETER Path
File path to restore.
.PARAMETER Confirm
Skip interactive prompt when provided.
.EXAMPLE
dm g_restore_file -Path "README.md" -Confirm
#>
function g_restore_file {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [switch]$Confirm
    )

    _assert_git_repo
    if (-not $Confirm) {
        $answer = Read-Host "Restore '$Path' from HEAD? (y/N)"
        if ($answer -notin @("y", "Y", "yes", "YES")) {
            Write-Host "Canceled."
            return
        }
    }
    git restore -- "$Path"
}

<#
.SYNOPSIS
Create a stash entry.
.DESCRIPTION
Runs `git stash push` with optional message.
.PARAMETER Message
Optional stash message.
.EXAMPLE
dm g_stash_push -Message "WIP api"
#>
function g_stash_push {
    param(
        [string]$Message
    )

    _assert_git_repo
    if ([string]::IsNullOrWhiteSpace($Message)) {
        git stash push
        return
    }
    git stash push -m "$Message"
}

<#
.SYNOPSIS
List stashes.
.DESCRIPTION
Runs `git stash list`.
.EXAMPLE
dm g_stash_list
#>
function g_stash_list {
    _assert_git_repo
    git stash list
}

<#
.SYNOPSIS
Apply and drop top stash.
.DESCRIPTION
Runs `git stash pop`.
.EXAMPLE
dm g_stash_pop
#>
function g_stash_pop {
    _assert_git_repo
    git stash pop
}

<#
.SYNOPSIS
Switch to an existing branch.
.DESCRIPTION
Runs `git switch <branch>`.
.PARAMETER Branch
Branch name.
.EXAMPLE
dm g_switch -Branch feature/login
#>
function g_switch {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Branch
    )

    _assert_git_repo
    git switch "$Branch"
}

<#
.SYNOPSIS
Create and switch to a new branch.
.DESCRIPTION
Runs `git checkout -b <branch>`.
.PARAMETER Branch
New branch name.
.EXAMPLE
dm g_checkout_new -Branch feature/login
#>
function g_checkout_new {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Branch
    )

    _assert_git_repo
    git checkout -b "$Branch"
}

<#
.SYNOPSIS
Continue an in-progress rebase.
.DESCRIPTION
Runs `git rebase --continue`.
.EXAMPLE
dm g_rebase_continue
#>
function g_rebase_continue {
    _assert_git_repo
    git rebase --continue
}

<#
.SYNOPSIS
Abort an in-progress rebase.
.DESCRIPTION
Runs `git rebase --abort`.
.EXAMPLE
dm g_rebase_abort
#>
function g_rebase_abort {
    _assert_git_repo
    git rebase --abort
}

<#
.SYNOPSIS
Abort an in-progress merge.
.DESCRIPTION
Runs `git merge --abort`.
.EXAMPLE
dm g_merge_abort
#>
function g_merge_abort {
    _assert_git_repo
    git merge --abort
}

<#
.SYNOPSIS
Move HEAD back by one commit (soft).
.DESCRIPTION
Runs `git reset --soft HEAD~1`.
.EXAMPLE
dm g_reset_soft_head1
#>
function g_reset_soft_head1 {
    _assert_git_repo
    git reset --soft HEAD~1
}

<#
.SYNOPSIS
Move HEAD back by one commit (mixed).
.DESCRIPTION
Runs `git reset --mixed HEAD~1`.
.EXAMPLE
dm g_reset_mixed_head1
#>
function g_reset_mixed_head1 {
    _assert_git_repo
    git reset --mixed HEAD~1
}

<#
.SYNOPSIS
Unstage all files.
.DESCRIPTION
Runs `git restore --staged .`.
.EXAMPLE
dm g_unstage_all
#>
function g_unstage_all {
    _assert_git_repo
    git restore --staged .
}

<#
.SYNOPSIS
Unstage one file.
.DESCRIPTION
Runs `git restore --staged <path>`.
.PARAMETER Path
File path to unstage.
.EXAMPLE
dm g_unstage_file -Path "README.md"
#>
function g_unstage_file {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    _assert_git_repo
    git restore --staged -- "$Path"
}

<#
.SYNOPSIS
Amend latest commit with a new message.
.DESCRIPTION
Runs `git commit --amend -m <message>`.
.PARAMETER Message
New commit message.
.EXAMPLE
dm g_commit_amend -Message "Refine parser"
#>
function g_commit_amend {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    _assert_git_repo
    git commit --amend -m "$Message"
}

<#
.SYNOPSIS
Amend latest commit without changing message.
.DESCRIPTION
Runs `git commit --amend --no-edit`.
.EXAMPLE
dm g_commit_amend_noedit
#>
function g_commit_amend_noedit {
    _assert_git_repo
    git commit --amend --no-edit
}

<#
.SYNOPSIS
Show one-line log for a single file.
.DESCRIPTION
Runs `git log --oneline -- <path>`.
.PARAMETER Path
File path to inspect.
.EXAMPLE
dm g_log_file -Path "internal/app/app.go"
#>
function g_log_file {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    _assert_git_repo
    git log --oneline -- "$Path"
}

<#
.SYNOPSIS
Show diff for a single file.
.DESCRIPTION
Runs `git diff -- <path>` and staged diff for same path.
.PARAMETER Path
File path to inspect.
.EXAMPLE
dm g_diff_file -Path "README.md"
#>
function g_diff_file {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    _assert_git_repo
    git diff -- "$Path"
    git diff --cached -- "$Path"
}

<#
.SYNOPSIS
Search content across tracked files.
.DESCRIPTION
Runs `git grep <pattern>`.
.PARAMETER Pattern
Search pattern.
.EXAMPLE
dm g_grep -Pattern "TODO"
#>
function g_grep {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Pattern
    )

    _assert_git_repo
    git grep -- "$Pattern"
}

<#
.SYNOPSIS
Show full details for one commit.
.DESCRIPTION
Runs `git show <ref>`.
.PARAMETER Ref
Commit reference (default HEAD).
.EXAMPLE
dm g_show -Ref HEAD~1
#>
function g_show {
    param(
        [string]$Ref = "HEAD"
    )

    _assert_git_repo
    git show "$Ref"
}

<#
.SYNOPSIS
Show remote configuration.
.DESCRIPTION
Runs `git remote -v`.
.EXAMPLE
dm g_remote_list
#>
function g_remote_list {
    _assert_git_repo
    git remote -v
}

<#
.SYNOPSIS
Show tags list.
.DESCRIPTION
Runs `git tag --list`.
.EXAMPLE
dm g_tag_list
#>
function g_tag_list {
    _assert_git_repo
    git tag --list
}

<#
.SYNOPSIS
Create an annotated tag.
.DESCRIPTION
Runs `git tag -a <name> -m <message>`.
.PARAMETER Name
Tag name.
.PARAMETER Message
Tag message.
.EXAMPLE
dm g_tag_create -Name v1.2.0 -Message "Release v1.2.0"
#>
function g_tag_create {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    _assert_git_repo
    git tag -a "$Name" -m "$Message"
}

<#
.SYNOPSIS
Push one tag to origin.
.DESCRIPTION
Runs `git push origin <tag>`.
.PARAMETER Name
Tag name to push.
.EXAMPLE
dm g_tag_push -Name v1.2.0
#>
function g_tag_push {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    _assert_git_repo
    git push origin "$Name"
}

<#
.SYNOPSIS
Push all tags to origin.
.DESCRIPTION
Runs `git push --tags`.
.EXAMPLE
dm g_tag_push_all
#>
function g_tag_push_all {
    _assert_git_repo
    git push --tags
}

<#
.SYNOPSIS
Apply stash without dropping it.
.DESCRIPTION
Runs `git stash apply` or `git stash apply <ref>`.
.PARAMETER Ref
Optional stash reference.
.EXAMPLE
dm g_stash_apply -Ref "stash@{1}"
#>
function g_stash_apply {
    param(
        [string]$Ref
    )

    _assert_git_repo
    if ([string]::IsNullOrWhiteSpace($Ref)) {
        git stash apply
        return
    }
    git stash apply "$Ref"
}

<#
.SYNOPSIS
Drop one stash entry.
.DESCRIPTION
Runs `git stash drop <ref>`.
.PARAMETER Ref
Stash reference.
.EXAMPLE
dm g_stash_drop -Ref "stash@{0}"
#>
function g_stash_drop {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Ref
    )

    _assert_git_repo
    git stash drop "$Ref"
}

<#
.SYNOPSIS
Clear all stash entries.
.DESCRIPTION
Runs `git stash clear` after confirmation.
.PARAMETER Confirm
Skip interactive prompt when provided.
.EXAMPLE
dm g_stash_clear -Confirm
#>
function g_stash_clear {
    param(
        [switch]$Confirm
    )

    _assert_git_repo
    if (-not $Confirm) {
        $answer = Read-Host "Clear ALL stashes? (y/N)"
        if ($answer -notin @("y", "Y", "yes", "YES")) {
            Write-Host "Canceled."
            return
        }
    }
    git stash clear
}

<#
.SYNOPSIS
Delete a local branch.
.DESCRIPTION
Runs `git branch -d <branch>` or `-D` if forced.
.PARAMETER Branch
Branch name.
.PARAMETER Force
Use `-D` instead of `-d`.
.EXAMPLE
dm g_branch_delete_local -Branch old/feature -Force
#>
function g_branch_delete_local {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Branch,
        [switch]$Force
    )

    _assert_git_repo
    if ($Force) {
        git branch -D "$Branch"
        return
    }
    git branch -d "$Branch"
}

<#
.SYNOPSIS
Delete a remote branch on origin.
.DESCRIPTION
Runs `git push origin --delete <branch>`.
.PARAMETER Branch
Remote branch name.
.EXAMPLE
dm g_branch_delete_remote -Branch old/feature
#>
function g_branch_delete_remote {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Branch
    )

    _assert_git_repo
    git push origin --delete "$Branch"
}

<#
.SYNOPSIS
Rename current branch.
.DESCRIPTION
Runs `git branch -m <newName>`.
.PARAMETER NewName
New branch name.
.EXAMPLE
dm g_branch_rename -NewName feature/new-name
#>
function g_branch_rename {
    param(
        [Parameter(Mandatory = $true)]
        [string]$NewName
    )

    _assert_git_repo
    git branch -m "$NewName"
}

<#
.SYNOPSIS
Switch to main-like branch.
.DESCRIPTION
Tries `main`, then `master`.
.EXAMPLE
dm g_switch_main
#>
function g_switch_main {
    _assert_git_repo
    git show-ref --verify --quiet refs/heads/main
    if ($LASTEXITCODE -eq 0) {
        git switch main
        return
    }
    git show-ref --verify --quiet refs/heads/master
    if ($LASTEXITCODE -eq 0) {
        git switch master
        return
    }
    throw "Neither 'main' nor 'master' branch exists."
}

<#
.SYNOPSIS
Pull with rebase strategy.
.DESCRIPTION
Runs `git pull --rebase`.
.EXAMPLE
dm g_pull_rebase
#>
function g_pull_rebase {
    _assert_git_repo
    git pull --rebase
}

<#
.SYNOPSIS
Rebase current branch on main/master.
.DESCRIPTION
Fetches remotes and rebases on origin/main or origin/master.
.EXAMPLE
dm g_rebase_main
#>
function g_rebase_main {
    _assert_git_repo
    git fetch --all --prune
    git show-ref --verify --quiet refs/remotes/origin/main
    if ($LASTEXITCODE -eq 0) {
        git rebase origin/main
        return
    }
    git show-ref --verify --quiet refs/remotes/origin/master
    if ($LASTEXITCODE -eq 0) {
        git rebase origin/master
        return
    }
    throw "Neither 'origin/main' nor 'origin/master' was found."
}

<#
.SYNOPSIS
Merge main/master into current branch.
.DESCRIPTION
Fetches remotes and merges origin/main or origin/master.
.EXAMPLE
dm g_merge_main
#>
function g_merge_main {
    _assert_git_repo
    git fetch --all --prune
    git show-ref --verify --quiet refs/remotes/origin/main
    if ($LASTEXITCODE -eq 0) {
        git merge origin/main
        return
    }
    git show-ref --verify --quiet refs/remotes/origin/master
    if ($LASTEXITCODE -eq 0) {
        git merge origin/master
        return
    }
    throw "Neither 'origin/main' nor 'origin/master' was found."
}

<#
.SYNOPSIS
Push with force-with-lease.
.DESCRIPTION
Runs `git push --force-with-lease`.
.EXAMPLE
dm g_push_force_with_lease
#>
function g_push_force_with_lease {
    _assert_git_repo
    git push --force-with-lease
}

<#
.SYNOPSIS
Cherry-pick one commit.
.DESCRIPTION
Runs `git cherry-pick <ref>`.
.PARAMETER Ref
Commit reference.
.EXAMPLE
dm g_cherry_pick -Ref abc1234
#>
function g_cherry_pick {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Ref
    )

    _assert_git_repo
    git cherry-pick "$Ref"
}

<#
.SYNOPSIS
Revert one commit.
.DESCRIPTION
Runs `git revert <ref>`.
.PARAMETER Ref
Commit reference.
.EXAMPLE
dm g_revert -Ref abc1234
#>
function g_revert {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Ref
    )

    _assert_git_repo
    git revert "$Ref"
}

<#
.SYNOPSIS
List git worktrees.
.DESCRIPTION
Runs `git worktree list`.
.EXAMPLE
dm g_worktree_list
#>
function g_worktree_list {
    _assert_git_repo
    git worktree list
}

<#
.SYNOPSIS
Add a git worktree for a branch.
.DESCRIPTION
Runs `git worktree add <path> <branch>`.
.PARAMETER Path
Target worktree path.
.PARAMETER Branch
Branch to checkout in the worktree.
.EXAMPLE
dm g_worktree_add -Path "../repo-hotfix" -Branch hotfix/urgent
#>
function g_worktree_add {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [Parameter(Mandatory = $true)]
        [string]$Branch
    )

    _assert_git_repo
    git worktree add "$Path" "$Branch"
}

<#
.SYNOPSIS
Remove a git worktree.
.DESCRIPTION
Runs `git worktree remove <path>`.
.PARAMETER Path
Worktree path to remove.
.PARAMETER Force
Use `--force` removal.
.EXAMPLE
dm g_worktree_remove -Path "../repo-hotfix" -Force
#>
function g_worktree_remove {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [switch]$Force
    )

    _assert_git_repo
    if ($Force) {
        git worktree remove --force "$Path"
        return
    }
    git worktree remove "$Path"
}

<#
.SYNOPSIS
Sync and init submodules.
.DESCRIPTION
Runs `git submodule update --init --recursive`.
.EXAMPLE
dm g_submodule_update
#>
function g_submodule_update {
    _assert_git_repo
    git submodule update --init --recursive
}

<#
.SYNOPSIS
Show Git cheat sheet in terminal.
.DESCRIPTION
Prints `docs/git-cheatsheet.md` from repository root.
.PARAMETER Paged
Show output through pager (`more`).
.EXAMPLE
dm g_cheatsheet
#>
function g_cheatsheet {
    param(
        [switch]$Paged
    )

    _assert_git_repo
    $repoRoot = (git rev-parse --show-toplevel).Trim()
    $cheatPath = Join-Path $repoRoot "docs\git-cheatsheet.md"
    _assert_path_exists -Path $cheatPath

    if ($Paged) {
        Get-Content $cheatPath | more
        return
    }
    Get-Content $cheatPath
}
