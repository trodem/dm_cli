package config

import (
	"fmt"
	"sort"
	"strings"
)

type Issue struct {
	Level   string
	Message string
}

func Validate(cfg Config) []Issue {
	var issues []Issue

	if strings.TrimSpace(cfg.Search.Knowledge) == "" {
		issues = append(issues, Issue{Level: "warn", Message: "search.knowledge is empty"})
	}

	for k, v := range cfg.Jump {
		if strings.TrimSpace(k) == "" {
			issues = append(issues, Issue{Level: "error", Message: "jump key is empty"})
		}
		if strings.TrimSpace(v) == "" {
			issues = append(issues, Issue{Level: "error", Message: fmt.Sprintf("jump '%s' has empty path", k)})
		}
	}

	for k, v := range cfg.Run {
		if strings.TrimSpace(k) == "" {
			issues = append(issues, Issue{Level: "error", Message: "run key is empty"})
		}
		if strings.TrimSpace(v) == "" {
			issues = append(issues, Issue{Level: "error", Message: fmt.Sprintf("run '%s' has empty command", k)})
		}
	}

	for name, p := range cfg.Projects {
		if strings.TrimSpace(name) == "" {
			issues = append(issues, Issue{Level: "error", Message: "project name is empty"})
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, Issue{Level: "error", Message: fmt.Sprintf("project '%s' has empty path", name)})
		}
		for ck, cv := range p.Commands {
			if strings.TrimSpace(ck) == "" {
				issues = append(issues, Issue{Level: "error", Message: fmt.Sprintf("project '%s' has empty command name", name)})
			}
			if strings.TrimSpace(cv) == "" {
				issues = append(issues, Issue{Level: "error", Message: fmt.Sprintf("project '%s' action '%s' is empty", name, ck)})
			}
		}
		if len(p.Commands) == 0 {
			issues = append(issues, Issue{Level: "warn", Message: fmt.Sprintf("project '%s' has no commands", name)})
		}
	}

	for _, issue := range detectNameCollisions(cfg) {
		issues = append(issues, issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Level == issues[j].Level {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Level < issues[j].Level
	})

	return issues
}

func detectNameCollisions(cfg Config) []Issue {
	var issues []Issue
	seen := map[string]string{}

	for k := range cfg.Jump {
		seen[k] = "jump"
	}
	for k := range cfg.Run {
		if prev, ok := seen[k]; ok {
			issues = append(issues, Issue{Level: "warn", Message: fmt.Sprintf("name '%s' exists in %s and run", k, prev)})
		} else {
			seen[k] = "run"
		}
	}
	for k := range cfg.Projects {
		if prev, ok := seen[k]; ok {
			issues = append(issues, Issue{Level: "warn", Message: fmt.Sprintf("name '%s' exists in %s and projects", k, prev)})
		} else {
			seen[k] = "projects"
		}
	}

	return issues
}
