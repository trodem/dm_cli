# =============================================================================
# GIT TOOLKIT – Local Git operational layer (standalone)
# Git helpers for local development environments.
# Safety: Non-destructive defaults. Remote delete requires -Force or confirmation.
# Entry point: git_*
#
# FUNCTIONS
#   git_status
#   git_branch_current
#   git_branch_list
#   git_fetch
#   git_pull
#   git_pull_rebase
#   git_push
#   git_push_force_with_lease
#   git_add_all
#   git_commit
#   git_add_commit
#   git_commit_amend
#   git_commit_amend_noedit
#   git_log
#   git_log_graph_oneline
#   git_diff
#   git_diff_file
#   git_log_file
#   git_grep
#   git_show
#   git_remote_list
#   git_tag_list
#   git_tag_create
#   git_tag_push
#   git_tag_push_all
#   git_switch
#   git_checkout_new
#   git_branch_delete_local
#   git_branch_delete_remote
#   git_rebase_continue
#   git_rebase_abort
#   git_merge_abort
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name git
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ask for yes/no confirmation before a risky action.
.PARAMETER Prompt
Message shown to the user.
.EXAMPLE
if (-not (_confirm_action -Prompt "Continue?")) { return }
#>
function _confirm_action {
    param([Parameter(Mandatory = $true)][string]$Prompt)
    $answer = Read-Host "$Prompt [y/N]"
    if ([string]::IsNullOrWhiteSpace($answer)) { return $false }
    return $answer.Trim().ToLowerInvariant() -in @("y", "yes")
}

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

# -----------------------------------------------------------------------------
# Read operations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show working tree status.
.DESCRIPTION
Displays staged, unstaged and untracked files.
.EXAMPLE
git_status
#>
function git_status { _assert_git_repo; git status }

<#
.SYNOPSIS
Show current branch name.
.DESCRIPTION
Prints the name of the currently checked-out branch.
.EXAMPLE
git_branch_current
#>
function git_branch_current { _assert_git_repo; git branch --show-current }

<#
.SYNOPSIS
List all local and remote branches.
.DESCRIPTION
Shows every branch including remotes.
.EXAMPLE
git_branch_list
#>
function git_branch_list { _assert_git_repo; git branch -a }

<#
.SYNOPSIS
Show recent commit log with stats.
.DESCRIPTION
Displays last 20 commits with file change statistics.
.EXAMPLE
git_log
#>
function git_log { _assert_git_repo; git log --decorate --stat -n 20 }

<#
.SYNOPSIS
Show commit graph as one-line-per-commit.
.DESCRIPTION
Displays all branches as an ASCII graph with abbreviated commits.
.EXAMPLE
git_log_graph_oneline
#>
function git_log_graph_oneline { _assert_git_repo; git log --oneline --graph --decorate --all }

<#
.SYNOPSIS
Show unstaged and staged diffs.
.DESCRIPTION
Displays both working directory changes and staged changes.
.EXAMPLE
git_diff
#>
function git_diff { _assert_git_repo; git diff; git diff --cached }

<#
.SYNOPSIS
Show diff for a specific file.
.DESCRIPTION
Displays unstaged and staged changes for the given file path.
.PARAMETER Path
File path to diff.
.EXAMPLE
git_diff_file -Path src/main.go
#>
function git_diff_file {
    param([Parameter(Mandatory = $true)][string]$Path)
    _assert_git_repo
    git diff -- "$Path"
    git diff --cached -- "$Path"
}

<#
.SYNOPSIS
Show commit history for a specific file.
.DESCRIPTION
Lists one-line commits that touched the given file.
.PARAMETER Path
File path to inspect.
.EXAMPLE
git_log_file -Path src/main.go
#>
function git_log_file {
    param([Parameter(Mandatory = $true)][string]$Path)
    _assert_git_repo
    git log --oneline -- "$Path"
}

<#
.SYNOPSIS
Search tracked files for a text pattern.
.DESCRIPTION
Runs git grep to find matching lines in the repository.
.PARAMETER Pattern
Text or regex pattern to search for.
.EXAMPLE
git_grep -Pattern "TODO"
#>
function git_grep {
    param([Parameter(Mandatory = $true)][string]$Pattern)
    _assert_git_repo
    git grep -- "$Pattern"
}

<#
.SYNOPSIS
Show details of a commit.
.DESCRIPTION
Displays commit metadata and diff for the given ref (default HEAD).
.PARAMETER Ref
Commit reference (default HEAD).
.EXAMPLE
git_show -Ref HEAD~1
#>
function git_show {
    param([string]$Ref = "HEAD")
    _assert_git_repo
    git show "$Ref"
}

<#
.SYNOPSIS
List configured remotes with URLs.
.DESCRIPTION
Shows all remotes and their fetch/push URLs.
.EXAMPLE
git_remote_list
#>
function git_remote_list { _assert_git_repo; git remote -v }

<#
.SYNOPSIS
List all tags.
.DESCRIPTION
Shows every tag in the repository.
.EXAMPLE
git_tag_list
#>
function git_tag_list { _assert_git_repo; git tag --list }

# -----------------------------------------------------------------------------
# Fetch / Pull / Push
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Fetch all remotes and prune stale branches.
.DESCRIPTION
Runs git fetch --all --prune.
.EXAMPLE
git_fetch
#>
function git_fetch { _assert_git_repo; git fetch --all --prune }

<#
.SYNOPSIS
Pull latest changes from remote.
.DESCRIPTION
Runs git pull with default merge strategy.
.EXAMPLE
git_pull
#>
function git_pull { _assert_git_repo; git pull }

<#
.SYNOPSIS
Pull with rebase from remote.
.DESCRIPTION
Runs git pull --rebase to replay local commits on top of upstream.
.EXAMPLE
git_pull_rebase
#>
function git_pull_rebase { _assert_git_repo; git pull --rebase }

<#
.SYNOPSIS
Push commits to remote.
.DESCRIPTION
Runs git push to the tracked upstream branch.
.EXAMPLE
git_push
#>
function git_push { _assert_git_repo; git push }

<#
.SYNOPSIS
Force-push with lease safety.
.DESCRIPTION
Pushes forcefully but aborts if remote has new commits since last fetch.
.EXAMPLE
git_push_force_with_lease
#>
function git_push_force_with_lease { _assert_git_repo; git push --force-with-lease }

# -----------------------------------------------------------------------------
# Stage / Commit
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Stage all changes in working directory.
.DESCRIPTION
Runs git add . to stage all modified, new and deleted files.
.EXAMPLE
git_add_all
#>
function git_add_all { _assert_git_repo; git add . }

<#
.SYNOPSIS
Commit staged changes with a message.
.DESCRIPTION
Runs git commit -m with the provided message.
.PARAMETER Message
Commit message text.
.EXAMPLE
git_commit -Message "Fix login bug"
#>
function git_commit {
    param([Parameter(Mandatory = $true)][string]$Message)
    _assert_git_repo
    git commit -m "$Message"
}

<#
.SYNOPSIS
Stage all changes and commit with a message.
.DESCRIPTION
Runs git add . followed by git commit -m.
.PARAMETER Message
Commit message text.
.EXAMPLE
git_add_commit -Message "Add new feature"
#>
function git_add_commit {
    param([Parameter(Mandatory = $true)][string]$Message)
    _assert_git_repo
    git add .
    git commit -m "$Message"
}

<#
.SYNOPSIS
Amend last commit with a new message.
.DESCRIPTION
Runs git commit --amend -m to replace the last commit message.
.PARAMETER Message
New commit message.
.EXAMPLE
git_commit_amend -Message "Corrected message"
#>
function git_commit_amend {
    param([Parameter(Mandatory = $true)][string]$Message)
    _assert_git_repo
    git commit --amend -m "$Message"
}

<#
.SYNOPSIS
Amend last commit keeping the same message.
.DESCRIPTION
Runs git commit --amend --no-edit to add staged changes to the last commit.
.EXAMPLE
git_commit_amend_noedit
#>
function git_commit_amend_noedit { _assert_git_repo; git commit --amend --no-edit }

# -----------------------------------------------------------------------------
# Branch management
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Switch to an existing branch.
.DESCRIPTION
Checks out the specified branch using git switch.
.PARAMETER Branch
Target branch name.
.EXAMPLE
git_switch -Branch develop
#>
function git_switch {
    param([Parameter(Mandatory = $true)][string]$Branch)
    _assert_git_repo
    git switch "$Branch"
}

<#
.SYNOPSIS
Create and switch to a new branch.
.DESCRIPTION
Runs git checkout -b to create a new branch from the current HEAD.
.PARAMETER Branch
New branch name.
.EXAMPLE
git_checkout_new -Branch feature/login
#>
function git_checkout_new {
    param([Parameter(Mandatory = $true)][string]$Branch)
    _assert_git_repo
    git checkout -b "$Branch"
}

<#
.SYNOPSIS
Delete a local branch.
.DESCRIPTION
Removes a local branch. Use -Force for unmerged branches.
.PARAMETER Branch
Branch name to delete.
.PARAMETER Force
Use -D instead of -d to force-delete unmerged branches.
.EXAMPLE
git_branch_delete_local -Branch old-feature
#>
function git_branch_delete_local {
    param(
        [Parameter(Mandatory = $true)][string]$Branch,
        [switch]$Force
    )
    _assert_git_repo
    if ($Force) { git branch -D "$Branch" } else { git branch -d "$Branch" }
}

<#
.SYNOPSIS
Delete a remote branch on origin.
.DESCRIPTION
Runs git push origin --delete to remove a branch from the remote.
Requires -Force or interactive confirmation.
.PARAMETER Branch
Remote branch name to delete.
.PARAMETER Force
Skip interactive confirmation.
.EXAMPLE
git_branch_delete_remote -Branch old-feature -Force
#>
function git_branch_delete_remote {
    param(
        [Parameter(Mandatory = $true)][string]$Branch,
        [switch]$Force
    )
    _assert_git_repo

    if (-not $Force) {
        if (-not (_confirm_action -Prompt "Delete remote branch '$Branch'?")) {
            return
        }
    }

    git push origin --delete "$Branch"
}

# -----------------------------------------------------------------------------
# Tags
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Create an annotated tag.
.DESCRIPTION
Creates a new annotated tag with a message at the current HEAD.
.PARAMETER Name
Tag name (e.g. v1.0.0).
.PARAMETER Message
Tag annotation message.
.EXAMPLE
git_tag_create -Name v1.0.0 -Message "Release 1.0.0"
#>
function git_tag_create {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Message
    )
    _assert_git_repo
    git tag -a "$Name" -m "$Message"
}

<#
.SYNOPSIS
Push a single tag to remote.
.DESCRIPTION
Pushes the specified tag to origin.
.PARAMETER Name
Tag name to push.
.EXAMPLE
git_tag_push -Name v1.0.0
#>
function git_tag_push {
    param([Parameter(Mandatory = $true)][string]$Name)
    _assert_git_repo
    git push origin "$Name"
}

<#
.SYNOPSIS
Push all local tags to remote.
.DESCRIPTION
Runs git push --tags to sync every local tag.
.EXAMPLE
git_tag_push_all
#>
function git_tag_push_all { _assert_git_repo; git push --tags }

# -----------------------------------------------------------------------------
# Rebase / Merge
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Continue an in-progress rebase.
.DESCRIPTION
Runs git rebase --continue after resolving conflicts.
.EXAMPLE
git_rebase_continue
#>
function git_rebase_continue { _assert_git_repo; git rebase --continue }

<#
.SYNOPSIS
Abort an in-progress rebase.
.DESCRIPTION
Cancels the current rebase and restores the previous state.
.EXAMPLE
git_rebase_abort
#>
function git_rebase_abort { _assert_git_repo; git rebase --abort }

<#
.SYNOPSIS
Abort an in-progress merge.
.DESCRIPTION
Cancels the current merge and restores the previous state.
.EXAMPLE
git_merge_abort
#>
function git_merge_abort { _assert_git_repo; git merge --abort }
