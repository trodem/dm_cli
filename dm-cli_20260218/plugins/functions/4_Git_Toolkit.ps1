# =============================================================================
# DM GIT TOOLKIT – Local Git Operational Layer
# Production-safe Git helpers for local development environments
# Non-destructive defaults, deterministic behavior, no admin requirements
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
Invoke git_status.
.DESCRIPTION
Helper/command function for git_status.
.EXAMPLE
dm git_status
#>
function git_status { _assert_git_repo; git status }
<#
.SYNOPSIS
Invoke git_branch_current.
.DESCRIPTION
Helper/command function for git_branch_current.
.EXAMPLE
dm git_branch_current
#>
function git_branch_current { _assert_git_repo; git branch --show-current }
<#
.SYNOPSIS
Invoke git_branch_list.
.DESCRIPTION
Helper/command function for git_branch_list.
.EXAMPLE
dm git_branch_list
#>
function git_branch_list { _assert_git_repo; git branch -a }
<#
.SYNOPSIS
Invoke git_fetch.
.DESCRIPTION
Helper/command function for git_fetch.
.EXAMPLE
dm git_fetch
#>
function git_fetch { _assert_git_repo; git fetch --all --prune }
<#
.SYNOPSIS
Invoke git_pull.
.DESCRIPTION
Helper/command function for git_pull.
.EXAMPLE
dm git_pull
#>
function git_pull { _assert_git_repo; git pull }
<#
.SYNOPSIS
Invoke git_pull_rebase.
.DESCRIPTION
Helper/command function for git_pull_rebase.
.EXAMPLE
dm git_pull_rebase
#>
function git_pull_rebase { _assert_git_repo; git pull --rebase }
<#
.SYNOPSIS
Invoke git_push.
.DESCRIPTION
Helper/command function for git_push.
.EXAMPLE
dm git_push
#>
function git_push { _assert_git_repo; git push }
<#
.SYNOPSIS
Invoke git_push_force_with_lease.
.DESCRIPTION
Helper/command function for git_push_force_with_lease.
.EXAMPLE
dm git_push_force_with_lease
#>
function git_push_force_with_lease { _assert_git_repo; git push --force-with-lease }
<#
.SYNOPSIS
Invoke git_add_all.
.DESCRIPTION
Helper/command function for git_add_all.
.EXAMPLE
dm git_add_all
#>
function git_add_all { _assert_git_repo; git add . }

<#
.SYNOPSIS
Invoke git_commit.
.DESCRIPTION
Helper/command function for git_commit.
.EXAMPLE
dm git_commit
#>
function git_commit {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git commit -m "$Message"
}

<#
.SYNOPSIS
Invoke git_add_commit.
.DESCRIPTION
Helper/command function for git_add_commit.
.EXAMPLE
dm git_add_commit
#>
function git_add_commit {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git add .
    git commit -m "$Message"
}

<#
.SYNOPSIS
Invoke git_commit_amend.
.DESCRIPTION
Helper/command function for git_commit_amend.
.EXAMPLE
dm git_commit_amend
#>
function git_commit_amend {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git commit --amend -m "$Message"
}

<#
.SYNOPSIS
Invoke git_commit_amend_noedit.
.DESCRIPTION
Helper/command function for git_commit_amend_noedit.
.EXAMPLE
dm git_commit_amend_noedit
#>
function git_commit_amend_noedit { _assert_git_repo; git commit --amend --no-edit }
<#
.SYNOPSIS
Invoke git_log.
.DESCRIPTION
Helper/command function for git_log.
.EXAMPLE
dm git_log
#>
function git_log { _assert_git_repo; git log --decorate --stat -n 20 }
<#
.SYNOPSIS
Invoke git_log_graph_oneline.
.DESCRIPTION
Helper/command function for git_log_graph_oneline.
.EXAMPLE
dm git_log_graph_oneline
#>
function git_log_graph_oneline { _assert_git_repo; git log --oneline --graph --decorate --all }
<#
.SYNOPSIS
Invoke git_diff.
.DESCRIPTION
Helper/command function for git_diff.
.EXAMPLE
dm git_diff
#>
function git_diff { _assert_git_repo; git diff; git diff --cached }

<#
.SYNOPSIS
Invoke git_diff_file.
.DESCRIPTION
Helper/command function for git_diff_file.
.EXAMPLE
dm git_diff_file
#>
function git_diff_file {
    param([Parameter(Mandatory=$true)][string]$Path)
    _assert_git_repo
    git diff -- "$Path"
    git diff --cached -- "$Path"
}

<#
.SYNOPSIS
Invoke git_log_file.
.DESCRIPTION
Helper/command function for git_log_file.
.EXAMPLE
dm git_log_file
#>
function git_log_file {
    param([Parameter(Mandatory=$true)][string]$Path)
    _assert_git_repo
    git log --oneline -- "$Path"
}

<#
.SYNOPSIS
Invoke git_grep.
.DESCRIPTION
Helper/command function for git_grep.
.EXAMPLE
dm git_grep
#>
function git_grep {
    param([Parameter(Mandatory=$true)][string]$Pattern)
    _assert_git_repo
    git grep -- "$Pattern"
}

<#
.SYNOPSIS
Invoke git_show.
.DESCRIPTION
Helper/command function for git_show.
.EXAMPLE
dm git_show
#>
function git_show {
    param([string]$Ref="HEAD")
    _assert_git_repo
    git show "$Ref"
}

<#
.SYNOPSIS
Invoke git_remote_list.
.DESCRIPTION
Helper/command function for git_remote_list.
.EXAMPLE
dm git_remote_list
#>
function git_remote_list { _assert_git_repo; git remote -v }
<#
.SYNOPSIS
Invoke git_tag_list.
.DESCRIPTION
Helper/command function for git_tag_list.
.EXAMPLE
dm git_tag_list
#>
function git_tag_list { _assert_git_repo; git tag --list }

<#
.SYNOPSIS
Invoke git_tag_create.
.DESCRIPTION
Helper/command function for git_tag_create.
.EXAMPLE
dm git_tag_create
#>
function git_tag_create {
    param(
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Message
    )
    _assert_git_repo
    git tag -a "$Name" -m "$Message"
}

<#
.SYNOPSIS
Invoke git_tag_push.
.DESCRIPTION
Helper/command function for git_tag_push.
.EXAMPLE
dm git_tag_push
#>
function git_tag_push {
    param([Parameter(Mandatory=$true)][string]$Name)
    _assert_git_repo
    git push origin "$Name"
}

<#
.SYNOPSIS
Invoke git_tag_push_all.
.DESCRIPTION
Helper/command function for git_tag_push_all.
.EXAMPLE
dm git_tag_push_all
#>
function git_tag_push_all { _assert_git_repo; git push --tags }

<#
.SYNOPSIS
Invoke git_switch.
.DESCRIPTION
Helper/command function for git_switch.
.EXAMPLE
dm git_switch
#>
function git_switch {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git switch "$Branch"
}

<#
.SYNOPSIS
Invoke git_checkout_new.
.DESCRIPTION
Helper/command function for git_checkout_new.
.EXAMPLE
dm git_checkout_new
#>
function git_checkout_new {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git checkout -b "$Branch"
}

<#
.SYNOPSIS
Invoke git_branch_delete_local.
.DESCRIPTION
Helper/command function for git_branch_delete_local.
.EXAMPLE
dm git_branch_delete_local
#>
function git_branch_delete_local {
    param(
        [Parameter(Mandatory=$true)][string]$Branch,
        [switch]$Force
    )
    _assert_git_repo
    if ($Force) { git branch -D "$Branch" } else { git branch -d "$Branch" }
}

<#
.SYNOPSIS
Invoke git_branch_delete_remote.
.DESCRIPTION
Helper/command function for git_branch_delete_remote.
.EXAMPLE
dm git_branch_delete_remote
#>
function git_branch_delete_remote {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git push origin --delete "$Branch"
}

<#
.SYNOPSIS
Invoke git_rebase_continue.
.DESCRIPTION
Helper/command function for git_rebase_continue.
.EXAMPLE
dm git_rebase_continue
#>
function git_rebase_continue { _assert_git_repo; git rebase --continue }
<#
.SYNOPSIS
Invoke git_rebase_abort.
.DESCRIPTION
Helper/command function for git_rebase_abort.
.EXAMPLE
dm git_rebase_abort
#>
function git_rebase_abort { _assert_git_repo; git rebase --abort }
<#
.SYNOPSIS
Invoke git_merge_abort.
.DESCRIPTION
Helper/command function for git_merge_abort.
.EXAMPLE
dm git_merge_abort
#>
function git_merge_abort { _assert_git_repo; git merge --abort }
