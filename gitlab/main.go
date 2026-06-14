package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xanzy/go-gitlab"
)

// GitlabHelper holds the GitLab client.
type GitlabHelper struct {
	client *gitlab.Client
}

// getAuthToken retrieves the GitLab token and base URL from env variables or from 'glab auth status -t'.
func getAuthToken() (string, string, error) {
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		return token, os.Getenv("GITLAB_BASE_URL"), nil
	}
	if token := os.Getenv("GL_TOKEN"); token != "" {
		baseURL := os.Getenv("GL_TOKEN_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("GL_BASE_URL")
		}
		return token, baseURL, nil
	}

	// Try to get token via glab CLI
	cmd := exec.Command("glab", "auth", "status", "-t")
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		output := stdout.String()
		if output == "" {
			output = stderr.String()
		}

		lines := strings.Split(output, "\n")
		var token string
		var baseURL string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "Token:") {
				parts := strings.Split(line, "Token:")
				if len(parts) > 1 {
					token = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "REST API Endpoint:") {
				parts := strings.Split(line, "REST API Endpoint:")
				if len(parts) > 1 {
					baseURL = strings.TrimSpace(parts[1])
				}
			}
		}
		if token != "" {
			return token, baseURL, nil
		}
	}

	return "", "", fmt.Errorf("GitLab authentication token not found in GITLAB_TOKEN/GL_TOKEN env vars or via 'glab auth status -t'")
}

func main() {
	token, baseURL, err := getAuthToken()
	if err != nil {
		// Log warning to Stderr (safe as MCP communicates on Stdin/Stdout)
		log.Printf("Warning: %v. Public repositories may still be accessible.", err)
	}

	ctx := context.Background()
	var client *gitlab.Client
	if token != "" {
		var err error
		if baseURL != "" {
			client, err = gitlab.NewOAuthClient(token, gitlab.WithBaseURL(baseURL))
		} else {
			client, err = gitlab.NewOAuthClient(token)
		}
		if err != nil {
			log.Fatalf("Failed to create GitLab client: %v", err)
		}
	} else {
		// Public access client (without token)
		client, err = gitlab.NewClient("")
		if err != nil {
			log.Fatalf("Failed to create GitLab client: %v", err)
		}
	}

	helper := &GitlabHelper{client: client}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gitlab-mcp",
		Version: "0.1.0",
	}, nil)

	// 1. get_project_info
	type getProjectInfoArgs struct {
		Owner string `json:"owner" jsonschema:"Project owner (username or group/organization)"`
		Repo  string `json:"repo" jsonschema:"Project name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_project_info",
		Description: "Get basic information about a GitLab project (repository)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getProjectInfoArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		project, _, err := helper.client.Projects.GetProject(projectPath, nil)
		if err != nil {
			return nil, nil, err
		}

		info := fmt.Sprintf(
			"Project: %s\nDescription: %s\nStars: %d\nForks: %d\nOpen Issues: %d\nDefault Branch: %s\nURL: %s",
			project.PathWithNamespace, project.Description,
			project.StarCount, project.ForksCount, project.OpenIssuesCount,
			project.DefaultBranch, project.WebURL,
		)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: info},
			},
		}, nil, nil
	})

	// 2. list_issues
	type listIssuesArgs struct {
		Owner   string  `json:"owner" jsonschema:"Project owner"`
		Repo    string  `json:"repo" jsonschema:"Project name"`
		State   *string `json:"state,omitempty" jsonschema:"Issue state: opened, closed, or all (default: opened)"`
		PerPage *int    `json:"per_page,omitempty" jsonschema:"Number of issues to return per page (default: 30)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_issues",
		Description: "List issues in a GitLab project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listIssuesArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		opts := &gitlab.ListProjectIssuesOptions{}
		if args.State != nil {
			state := *args.State
			// Map common states if user inputs GitHub style 'open'
			if state == "open" {
				state = "opened"
			}
			opts.State = &state
		}
		if args.PerPage != nil {
			opts.PerPage = *args.PerPage
		}

		issues, _, err := helper.client.Issues.ListProjectIssues(projectPath, opts)
		if err != nil {
			return nil, nil, err
		}

		var sb strings.Builder
		if len(issues) == 0 {
			sb.WriteString("No issues found.")
		} else {
			for _, issue := range issues {
				sb.WriteString(fmt.Sprintf(
					"#%d: %s (State: %s, Author: %s)\nURL: %s\n\n",
					issue.IID, issue.Title, issue.State,
					issue.Author.Username, issue.WebURL,
				))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: sb.String()},
			},
		}, nil, nil
	})

	// 3. create_issue
	type createIssueArgs struct {
		Owner string `json:"owner" jsonschema:"Project owner"`
		Repo  string `json:"repo" jsonschema:"Project name"`
		Title string `json:"title" jsonschema:"Issue title"`
		Body  string `json:"body" jsonschema:"Issue body description"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new issue in a GitLab project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createIssueArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		opts := &gitlab.CreateIssueOptions{
			Title:       &args.Title,
			Description: &args.Body,
		}
		issue, _, err := helper.client.Issues.CreateIssue(projectPath, opts)
		if err != nil {
			return nil, nil, err
		}
		result := fmt.Sprintf("Successfully created issue #%d: %s\nURL: %s", issue.IID, issue.Title, issue.WebURL)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 4. get_issue
	type getIssueArgs struct {
		Owner       string `json:"owner" jsonschema:"Project owner"`
		Repo        string `json:"repo" jsonschema:"Project name"`
		IssueNumber int    `json:"issue_number" jsonschema:"The issue IID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_issue",
		Description: "Get detailed information about a specific issue (including its description and comments)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getIssueArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		issue, _, err := helper.client.Issues.GetIssue(projectPath, args.IssueNumber)
		if err != nil {
			return nil, nil, err
		}

		notesOpts := &gitlab.ListIssueNotesOptions{}
		notes, _, _ := helper.client.Notes.ListIssueNotes(projectPath, args.IssueNumber, notesOpts)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(
			"Issue #%d: %s\nState: %s\nAuthor: %s\nCreated: %s\nURL: %s\n\nDescription:\n%s\n",
			issue.IID, issue.Title, issue.State,
			issue.Author.Username, issue.CreatedAt.String(), issue.WebURL,
			issue.Description,
		))

		if len(notes) > 0 {
			sb.WriteString("\nComments:\n")
			for _, note := range notes {
				if note.System {
					continue
				}
				sb.WriteString(fmt.Sprintf(
					"--- \nComment by %s on %s:\n%s\n",
					note.Author.Username, note.CreatedAt.String(),
					note.Body,
				))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: sb.String()},
			},
		}, nil, nil
	})

	// 5. create_issue_comment
	type createIssueCommentArgs struct {
		Owner       string `json:"owner" jsonschema:"Project owner"`
		Repo        string `json:"repo" jsonschema:"Project name"`
		IssueNumber int    `json:"issue_number" jsonschema:"The issue or MR IID to comment on"`
		Body        string `json:"body" jsonschema:"Comment body text"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_issue_comment",
		Description: "Add a comment to an existing issue or merge request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createIssueCommentArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		
		// First try as an issue comment (note)
		issueOpts := &gitlab.CreateIssueNoteOptions{
			Body: &args.Body,
		}
		note, _, err := helper.client.Notes.CreateIssueNote(projectPath, args.IssueNumber, issueOpts)
		
		if err != nil {
			// If it fails, try as a merge request comment (note)
			mrOpts := &gitlab.CreateMergeRequestNoteOptions{
				Body: &args.Body,
			}
			var mrErr error
			note, _, mrErr = helper.client.Notes.CreateMergeRequestNote(projectPath, args.IssueNumber, mrOpts)
			if mrErr != nil {
				return nil, nil, fmt.Errorf("failed to add comment to issue (err: %v) and merge request (err: %v)", err, mrErr)
			}
		}

		result := fmt.Sprintf("Successfully added comment to issue/MR #%d\nComment ID: %d", args.IssueNumber, note.ID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 6. list_mrs
	type listMrsArgs struct {
		Owner   string  `json:"owner" jsonschema:"Project owner"`
		Repo    string  `json:"repo" jsonschema:"Project name"`
		State   *string `json:"state,omitempty" jsonschema:"MR state: opened, closed, locked, merged, or all (default: opened)"`
		PerPage *int    `json:"per_page,omitempty" jsonschema:"Number of MRs to return per page (default: 30)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_mrs",
		Description: "List merge requests in a GitLab project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listMrsArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		opts := &gitlab.ListProjectMergeRequestsOptions{}
		if args.State != nil {
			state := *args.State
			if state == "open" {
				state = "opened"
			}
			opts.State = &state
		}
		if args.PerPage != nil {
			opts.PerPage = *args.PerPage
		}

		mrs, _, err := helper.client.MergeRequests.ListProjectMergeRequests(projectPath, opts)
		if err != nil {
			return nil, nil, err
		}

		var sb strings.Builder
		if len(mrs) == 0 {
			sb.WriteString("No merge requests found.")
		} else {
			for _, mr := range mrs {
				sb.WriteString(fmt.Sprintf(
					"#%d: %s (State: %s, Author: %s)\nBranch: %s -> %s\nURL: %s\n\n",
					mr.IID, mr.Title, mr.State,
					mr.Author.Username, mr.SourceBranch, mr.TargetBranch,
					mr.WebURL,
				))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: sb.String()},
			},
		}, nil, nil
	})

	// 7. create_mr
	type createMrArgs struct {
		Owner string `json:"owner" jsonschema:"Project owner"`
		Repo  string `json:"repo" jsonschema:"Project name"`
		Title string `json:"title" jsonschema:"Merge request title"`
		Body  string `json:"body" jsonschema:"Merge request description"`
		Head  string `json:"head" jsonschema:"The name of the branch where your changes are implemented"`
		Base  string `json:"base" jsonschema:"The name of the branch you want to merge into (e.g. main or master)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_mr",
		Description: "Create a new merge request in a GitLab project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createMrArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		opts := &gitlab.CreateMergeRequestOptions{
			Title:        &args.Title,
			Description:  &args.Body,
			SourceBranch: &args.Head,
			TargetBranch: &args.Base,
		}
		mr, _, err := helper.client.MergeRequests.CreateMergeRequest(projectPath, opts)
		if err != nil {
			return nil, nil, err
		}
		result := fmt.Sprintf("Successfully created Merge Request #%d: %s\nURL: %s", mr.IID, mr.Title, mr.WebURL)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 8. get_mr
	type getMrArgs struct {
		Owner    string `json:"owner" jsonschema:"Project owner"`
		Repo     string `json:"repo" jsonschema:"Project name"`
		PrNumber int    `json:"pr_number" jsonschema:"The merge request IID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_mr",
		Description: "Get detailed information about a specific merge request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getMrArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		mr, _, err := helper.client.MergeRequests.GetMergeRequest(projectPath, args.PrNumber, nil)
		if err != nil {
			return nil, nil, err
		}

		result := fmt.Sprintf(
			"MR #%d: %s\nState: %s\nAuthor: %s\nBranch: %s -> %s\nHas Conflicts: %t\nURL: %s\n\nDescription:\n%s",
			mr.IID, mr.Title, mr.State,
			mr.Author.Username, mr.SourceBranch, mr.TargetBranch,
			mr.HasConflicts, mr.WebURL,
			mr.Description,
		)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 9. merge_mr
	type mergeMrArgs struct {
		Owner       string  `json:"owner" jsonschema:"Project owner"`
		Repo        string  `json:"repo" jsonschema:"Project name"`
		PrNumber    int     `json:"pr_number" jsonschema:"The merge request IID"`
		CommitTitle *string `json:"commit_title,omitempty" jsonschema:"Extra detail for the merge commit message"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "merge_mr",
		Description: "Merge a merge request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mergeMrArgs) (*mcp.CallToolResult, any, error) {
		projectPath := fmt.Sprintf("%s/%s", args.Owner, args.Repo)
		opts := &gitlab.AcceptMergeRequestOptions{}
		if args.CommitTitle != nil {
			opts.MergeCommitMessage = args.CommitTitle
		}

		mr, _, err := helper.client.MergeRequests.AcceptMergeRequest(projectPath, args.PrNumber, opts)
		if err != nil {
			return nil, nil, err
		}

		result := fmt.Sprintf("MR #%d merged status: %s, Title: %s, SHA: %s", args.PrNumber, mr.State, mr.Title, mr.SHA)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 10. commit_and_push
	type commitAndPushArgs struct {
		RepoPath      string  `json:"repo_path" jsonschema:"Absolute path to the local git repository"`
		CommitMessage string  `json:"commit_message" jsonschema:"Commit message"`
		Branch        *string `json:"branch,omitempty" jsonschema:"Branch to push to (default: current branch)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "commit_and_push",
		Description: "Stage all changes, commit, and push to the remote repository using local git CLI",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args commitAndPushArgs) (*mcp.CallToolResult, any, error) {
		cmdAdd := exec.CommandContext(ctx, "git", "add", ".")
		cmdAdd.Dir = args.RepoPath
		if out, err := cmdAdd.CombinedOutput(); err != nil {
			return nil, nil, fmt.Errorf("git add failed: %v\nOutput: %s", err, string(out))
		}

		cmdCommit := exec.CommandContext(ctx, "git", "commit", "-m", args.CommitMessage)
		cmdCommit.Dir = args.RepoPath
		out, err := cmdCommit.CombinedOutput()
		commitOutput := string(out)
		if err != nil && !strings.Contains(commitOutput, "nothing to commit") {
			return nil, nil, fmt.Errorf("git commit failed: %v\nOutput: %s", err, commitOutput)
		}

		var cmdPush *exec.Cmd
		if args.Branch != nil && *args.Branch != "" {
			cmdPush = exec.CommandContext(ctx, "git", "push", "origin", *args.Branch)
		} else {
			cmdPush = exec.CommandContext(ctx, "git", "push")
		}
		cmdPush.Dir = args.RepoPath
		if out, err := cmdPush.CombinedOutput(); err != nil {
			return nil, nil, fmt.Errorf("git push failed: %v\nOutput: %s", err, string(out))
		}

		result := fmt.Sprintf("Successfully committed and pushed changes in %s", args.RepoPath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// Run the server on Stdio transport
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
