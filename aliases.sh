# Some good standards, which are not used if the user
# creates his/her own .bashrc/.bash_profile

# Git alias location --> C:\Program Files\Git\etc\profile.d
# "C:\Program Files\Git\etc\profile.d\aliases.sh"

# --show-control-chars: help showing Korean or accented characters
alias ls='ls -F --color=auto --show-control-chars'

# General Aliases
alias ll='ls -l'
alias la='ls -a'
alias sql='winpty mysql -u root'
alias gl='git log --oneline --graph --branches'
alias gs='git status'
alias gss='git status -s'
alias gcl='git clone'
alias gbr='git branch'
alias gco='git checkout'
alias grh='git reset --hard'
alias gst='git stash'
alias gsta='git stash apply'
alias gpl='git pull'
alias gps='git push'
alias gconf=' git config --list'
alias dev='cd E:/'
alias node='winpty node.exe'
alias dps='docker ps'
alias dstop='docker stop $(docker ps -q)'
alias drm='docker rm $(docker ps -aq)'
alias dlogs='docker logs -f'
alias doccomup='docker-compose up -d'
alias doccomdo='docker-compose down'
alias dbuild='docker build -t'
alias dexec='docker exec -it'

# Git Functions
gcheckout() {
    read -p "Enter the branch name: " branch
    if git rev-parse --verify "$branch" >/dev/null 2>&1; then
        git checkout "$branch"
        echo "Switched to branch '$branch'."
    else
        read -p "Branch '$branch' does not exist. Create it? (y/n): " create
        if [[ "$create" == "y" || "$create" == "Y" ]]; then
            git checkout -b "$branch"
            echo "Branch '$branch' has been created and checked out."
        else
            echo "No action taken."
        fi
    fi
}

gcommit() {
    if git diff --quiet && git diff --cached --quiet; then
        echo "No changes to commit. Operation canceled."
        return 1
    fi
    read -p "Enter the commit message: " commit_message
    if [ -z "$commit_message" ]; then
        echo "Error: Commit message cannot be empty."
        return 1
    fi
    git add .
    git commit -m "$commit_message"
    if [ $? -eq 0 ]; then
        echo "Commit successful: '$commit_message'."
    else
        echo "Error during commit."
    fi
}

gpush() {
    if git diff --quiet && git diff --cached --quiet; then
        echo "No changes to push. Operation canceled."
        return 1
    fi
    read -p "Enter the commit message: " commit_message
    if [ -z "$commit_message" ]; then
        echo "Error: Commit message cannot be empty."
        return 1
    fi
    read -p "Are you sure you want to push changes? (y/n): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Push canceled."
        return 1
    fi
    git add .
    git commit -m "$commit_message"
    if git push; then
        echo "Push completed successfully."
    else
        echo "Error during push."
    fi
}

gmerge() {
    read -p "Enter the branch to merge: " branch
    if [ -z "$branch" ]; then
        echo "Error: No branch specified."
        return 1
    fi
    if ! git rev-parse --verify "$branch" >/dev/null 2>&1; then
        echo "Error: Branch '$branch' does not exist."
        return 1
    fi
    read -p "Are you sure you want to merge '$branch'? (y/n): " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        git merge "$branch"
        if [ $? -eq 0 ]; then
            echo "Merge of '$branch' completed successfully."
        else
            echo "Error during merge of '$branch'."
        fi
    else
        echo "Merge canceled."
    fi
}


gstatus() {

    echo "Fetching the latest information from the repository..."
    git fetch --all --prune >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: Unable to fetch the latest information."
        return 1
    fi

    echo "Displaying the current status of the repository..."
    git status
    echo ""

    echo "🔹 Current branch:"
    git rev-parse --abbrev-ref HEAD

    echo "🔹 Last 3 commits on this branch:"
    git log -3 --oneline --decorate --graph

    echo "🔹 Untracked files:"
    git ls-files --others --exclude-standard
    echo ""

}



gdiscard() {
    read -p "Are you sure you want to discard all unstaged changes? (y/n): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Operation canceled."
        return 1
    fi
    git restore . --source=HEAD
    if [ $? -eq 0 ]; then
        echo "All unstaged changes have been discarded."
    else
        echo "Error: Unable to discard changes."
    fi
}



gdiscard_all() {
    read -p "Are you sure you want to discard all changes, including untracked files? (y/n): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Operation canceled."
        return 1
    fi
    git restore . --source=HEAD
    git clean -fd
    if [ $? -eq 0 ]; then
        echo "All changes, including untracked files, have been discarded."
    else
        echo "Error: Unable to discard changes."
    fi
}



ghelp() {
    echo "📋 List of custom commands and snippets with descriptions:"
    echo ""

    echo "🔹 Aliases:"
    cat <<EOF | column -t -s'|'
Alias           | Description
code            | Open Visual Studio Code
doc             | change to Dokumente

ll              | List files in long format
la              | List all files, including hidden ones
gl              | Git log with a simple graph
gs              | Git status
gss             | Git status -s
gconf           | Git config --list
gcl             | git clone
gbr             | git branch
gco             | git checkout
grh             | git reset --hard
gst             | git stash
gsta            | git stash apply
gpl             | git pull
gps             | git push
node            | winpty node.exe

dps             | Show running Docker containers
drm             | Remove all stopped Docker containers
dlogs           | docker logs -f
doccomup        | docker-compose up -d
doccomdo        | docker-compose down
dbuild          | docker build -t
dexec           | docker exec -it
dstop           | Stop all running Docker containers
sql             | Open MySQL with root user
EOF
    echo ""
    echo "🔹 Functions:"
    cat <<EOF | column -t -s'|'
Function        | Description
gcheckout       | Checkout or create a branch interactively
gcommit         | Add and commit changes with an interactive message
gpush           | Add, commit, and push changes with confirmation
gmerge          | Merge a branch into the current one with checks
gdiscard        | Discard all unstaged changes
gdiscard_all    | Discard all changes, including untracked files
sqlite          | Open SQLite3 interactively, specify a database or use in-memory
gstatus         | Display repository status, current branch, recent commits, and untracked files
EOF

    echo ""
    echo "Git alias location --> C:\Program Files\Git\etc\profile.d"
}


case "$TERM" in
xterm*)
	# The following programs are known to require a Win32 Console
	# for interactive usage, therefore let's launch them through winpty
	# when run inside `mintty`.
	for name in node ipython php php5 psql python2.7
	do
		case "$(type -p "$name".exe 2>/dev/null)" in
		''|/usr/bin/*) continue;;
		esac
		alias $name="winpty $name.exe"
	done
	;;
esac
