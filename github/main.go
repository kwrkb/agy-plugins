package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GithubHelper holds the GitHub client.
type GithubHelper struct {
	client *github.Client
}

// getAuthToken retrieves the GitHub token from env variables or from 'gh auth token'.
func getAuthToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token, nil
	}

	// Try to get token via gh CLI
	cmd := exec.Command("gh", "auth", "token")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		token := strings.TrimSpace(stdout.String())
		if token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("GitHub authentication token not found in GITHUB_TOKEN/GH_TOKEN env vars or via 'gh auth token'")
}

func main() {
	token, err := getAuthToken()
	if err != nil {
		// Log warning to Stderr (safe as MCP communicates on Stdin/Stdout)
		log.Printf("Warning: %v. Public repositories may still be accessible.", err)
	}

	ctx := context.Background()
	var client *github.Client
	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	helper := &GithubHelper{client: client}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "github-mcp",
		Version: "0.1.0",
	}, nil)

	// 1. get_repo_info
	type getRepoInfoArgs struct {
		Owner string `json:"owner" jsonschema:"Repository owner (username or organization)"`
		Repo  string `json:"repo" jsonschema:"Repository name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_repo_info",
		Description: "Get basic information about a GitHub repository",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getRepoInfoArgs) (*mcp.CallToolResult, any, error) {
		repo, _, err := helper.client.Repositories.Get(ctx, args.Owner, args.Repo)
		if err != nil {
			return nil, nil, err
		}

		info := fmt.Sprintf(
			"Repository: %s/%s\nDescription: %s\nStars: %d\nForks: %d\nOpen Issues: %d\nDefault Branch: %s\nURL: %s",
			repo.GetOwner().GetLogin(), repo.GetName(), repo.GetDescription(),
			repo.GetStargazersCount(), repo.GetForksCount(), repo.GetOpenIssuesCount(),
			repo.GetDefaultBranch(), repo.GetHTMLURL(),
		)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: info},
			},
		}, nil, nil
	})

	// 2. list_issues
	type listIssuesArgs struct {
		Owner   string  `json:"owner" jsonschema:"Repository owner"`
		Repo    string  `json:"repo" jsonschema:"Repository name"`
		State   *string `json:"state,omitempty" jsonschema:"Issue state: open, closed, or all (default: open)"`
		PerPage *int    `json:"per_page,omitempty" jsonschema:"Number of issues to return per page (default: 30)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_issues",
		Description: "List issues in a GitHub repository",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listIssuesArgs) (*mcp.CallToolResult, any, error) {
		opts := &github.IssueListByRepoOptions{}
		if args.State != nil {
			opts.State = *args.State
		}
		if args.PerPage != nil {
			opts.ListOptions.PerPage = *args.PerPage
		}

		issues, _, err := helper.client.Issues.ListByRepo(ctx, args.Owner, args.Repo, opts)
		if err != nil {
			return nil, nil, err
		}

		var sb strings.Builder
		if len(issues) == 0 {
			sb.WriteString("No issues found.")
		} else {
			for _, issue := range issues {
				prMark := ""
				if issue.IsPullRequest() {
					prMark = "[PR] "
				}
				sb.WriteString(fmt.Sprintf(
					"#%d: %s%s (State: %s, Author: %s)\nURL: %s\n\n",
					issue.GetNumber(), prMark, issue.GetTitle(), issue.GetState(),
					issue.GetUser().GetLogin(), issue.GetHTMLURL(),
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
		Owner string `json:"owner" jsonschema:"Repository owner"`
		Repo  string `json:"repo" jsonschema:"Repository name"`
		Title string `json:"title" jsonschema:"Issue title"`
		Body  string `json:"body" jsonschema:"Issue body description"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new issue in a GitHub repository",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createIssueArgs) (*mcp.CallToolResult, any, error) {
		issueReq := &github.IssueRequest{
			Title: &args.Title,
			Body:  &args.Body,
		}
		issue, _, err := helper.client.Issues.Create(ctx, args.Owner, args.Repo, issueReq)
		if err != nil {
			return nil, nil, err
		}
		result := fmt.Sprintf("Successfully created issue #%d: %s\nURL: %s", issue.GetNumber(), issue.GetTitle(), issue.GetHTMLURL())
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 4. get_issue
	type getIssueArgs struct {
		Owner       string `json:"owner" jsonschema:"Repository owner"`
		Repo        string `json:"repo" jsonschema:"Repository name"`
		IssueNumber int    `json:"issue_number" jsonschema:"The issue number"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_issue",
		Description: "Get detailed information about a specific issue (including its description and comments)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getIssueArgs) (*mcp.CallToolResult, any, error) {
		issue, _, err := helper.client.Issues.Get(ctx, args.Owner, args.Repo, args.IssueNumber)
		if err != nil {
			return nil, nil, err
		}

		comments, _, _ := helper.client.Issues.ListComments(ctx, args.Owner, args.Repo, args.IssueNumber, nil)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(
			"Issue #%d: %s\nState: %s\nAuthor: %s\nCreated: %s\nURL: %s\n\nDescription:\n%s\n",
			issue.GetNumber(), issue.GetTitle(), issue.GetState(),
			issue.GetUser().GetLogin(), issue.GetCreatedAt().String(), issue.GetHTMLURL(),
			issue.GetBody(),
		))

		if len(comments) > 0 {
			sb.WriteString("\nComments:\n")
			for _, comment := range comments {
				sb.WriteString(fmt.Sprintf(
					"--- \nComment by %s on %s:\n%s\n",
					comment.GetUser().GetLogin(), comment.GetCreatedAt().String(),
					comment.GetBody(),
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
		Owner       string `json:"owner" jsonschema:"Repository owner"`
		Repo        string `json:"repo" jsonschema:"Repository name"`
		IssueNumber int    `json:"issue_number" jsonschema:"The issue or PR number to comment on"`
		Body        string `json:"body" jsonschema:"Comment body text"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_issue_comment",
		Description: "Add a comment to an existing issue or pull request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createIssueCommentArgs) (*mcp.CallToolResult, any, error) {
		commentReq := &github.IssueComment{
			Body: &args.Body,
		}
		comment, _, err := helper.client.Issues.CreateComment(ctx, args.Owner, args.Repo, args.IssueNumber, commentReq)
		if err != nil {
			return nil, nil, err
		}
		result := fmt.Sprintf("Successfully added comment to issue/PR #%d\nComment URL: %s", args.IssueNumber, comment.GetHTMLURL())
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 6. list_prs
	type listPrsArgs struct {
		Owner   string  `json:"owner" jsonschema:"Repository owner"`
		Repo    string  `json:"repo" jsonschema:"Repository name"`
		State   *string `json:"state,omitempty" jsonschema:"PR state: open, closed, or all (default: open)"`
		PerPage *int    `json:"per_page,omitempty" jsonschema:"Number of PRs to return per page (default: 30)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_prs",
		Description: "List pull requests in a GitHub repository",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listPrsArgs) (*mcp.CallToolResult, any, error) {
		opts := &github.PullRequestListOptions{}
		if args.State != nil {
			opts.State = *args.State
		}
		if args.PerPage != nil {
			opts.ListOptions.PerPage = *args.PerPage
		}

		prs, _, err := helper.client.PullRequests.List(ctx, args.Owner, args.Repo, opts)
		if err != nil {
			return nil, nil, err
		}

		var sb strings.Builder
		if len(prs) == 0 {
			sb.WriteString("No pull requests found.")
		} else {
			for _, pr := range prs {
				sb.WriteString(fmt.Sprintf(
					"#%d: %s (State: %s, Author: %s)\nBranch: %s -> %s\nURL: %s\n\n",
					pr.GetNumber(), pr.GetTitle(), pr.GetState(),
					pr.GetUser().GetLogin(), pr.GetHead().GetRef(), pr.GetBase().GetRef(),
					pr.GetHTMLURL(),
				))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: sb.String()},
			},
		}, nil, nil
	})

	// 7. create_pr
	type createPrArgs struct {
		Owner string `json:"owner" jsonschema:"Repository owner"`
		Repo  string `json:"repo" jsonschema:"Repository name"`
		Title string `json:"title" jsonschema:"Pull request title"`
		Body  string `json:"body" jsonschema:"Pull request description"`
		Head  string `json:"head" jsonschema:"The name of the branch where your changes are implemented"`
		Base  string `json:"base" jsonschema:"The name of the branch you want to merge into (e.g. main or master)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_pr",
		Description: "Create a new pull request in a GitHub repository",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createPrArgs) (*mcp.CallToolResult, any, error) {
		prReq := &github.NewPullRequest{
			Title: &args.Title,
			Body:  &args.Body,
			Head:  &args.Head,
			Base:  &args.Base,
		}
		pr, _, err := helper.client.PullRequests.Create(ctx, args.Owner, args.Repo, prReq)
		if err != nil {
			return nil, nil, err
		}
		result := fmt.Sprintf("Successfully created Pull Request #%d: %s\nURL: %s", pr.GetNumber(), pr.GetTitle(), pr.GetHTMLURL())
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 8. get_pr
	type getPrArgs struct {
		Owner    string `json:"owner" jsonschema:"Repository owner"`
		Repo     string `json:"repo" jsonschema:"Repository name"`
		PrNumber int    `json:"pr_number" jsonschema:"The pull request number"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_pr",
		Description: "Get detailed information about a specific pull request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getPrArgs) (*mcp.CallToolResult, any, error) {
		pr, _, err := helper.client.PullRequests.Get(ctx, args.Owner, args.Repo, args.PrNumber)
		if err != nil {
			return nil, nil, err
		}

		result := fmt.Sprintf(
			"PR #%d: %s\nState: %s\nAuthor: %s\nBranch: %s -> %s\nMergeable: %v\nMerged: %t\nURL: %s\n\nDescription:\n%s",
			pr.GetNumber(), pr.GetTitle(), pr.GetState(),
			pr.GetUser().GetLogin(), pr.GetHead().GetRef(), pr.GetBase().GetRef(),
			pr.GetMergeable(), pr.GetMerged(), pr.GetHTMLURL(),
			pr.GetBody(),
		)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})

	// 9. merge_pr
	type mergePrArgs struct {
		Owner       string  `json:"owner" jsonschema:"Repository owner"`
		Repo        string  `json:"repo" jsonschema:"Repository name"`
		PrNumber    int     `json:"pr_number" jsonschema:"The pull request number"`
		CommitTitle *string `json:"commit_title,omitempty" jsonschema:"Extra detail for the merge commit message"`
		MergeMethod *string `json:"merge_method,omitempty" jsonschema:"Merge method to use: merge, squash, or rebase (default: merge)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "merge_pr",
		Description: "Merge a pull request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mergePrArgs) (*mcp.CallToolResult, any, error) {
		opts := &github.PullRequestOptions{}
		if args.MergeMethod != nil {
			opts.MergeMethod = *args.MergeMethod
		}
		commitMsg := ""
		if args.CommitTitle != nil {
			commitMsg = *args.CommitTitle
		}

		mergeResult, _, err := helper.client.PullRequests.Merge(ctx, args.Owner, args.Repo, args.PrNumber, commitMsg, opts)
		if err != nil {
			return nil, nil, err
		}

		result := fmt.Sprintf("PR #%d merged status: %t, Message: %s, SHA: %s", args.PrNumber, mergeResult.GetMerged(), mergeResult.GetMessage(), mergeResult.GetSHA())
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
