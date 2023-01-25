package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type edgeComment struct {
	Node struct {
		Body githubv4.String
	}
}

type edgePullRequest struct {
	Node struct {
		Repository struct {
			NameWithOwner  githubv4.String
			StargazerCount githubv4.Int
		}
		Comments struct {
			Edges []edgeComment
		} `graphql:"comments(last:1)"`
		Merged githubv4.Boolean
		Closed githubv4.Boolean
	}
}

var queryPullRequest struct {
	Viewer struct {
		PullRequests struct {
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
			TotalCount githubv4.Int
			Edges      []edgePullRequest
		} `graphql:"pullRequests(states: [MERGED, CLOSED], orderBy:{field: CREATED_AT, direction: ASC}, first:100, after: $after)"`
	}
}

type repository struct {
	OwnerName string
	StarCount int
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("env variable 'GITHUB_TOKEN' must be non-empty")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	allPullRequests, err := PullRequests(context.Background(), client)
	if err != nil {
		log.Fatalf("Failed to get merged pull requests: %v\n", err)
	}
	log.Printf("Total pull request: %d\n", len(allPullRequests))

	repositoryStars := map[string]int{}
	for _, pr := range allPullRequests {
		nameOwner := string(pr.Node.Repository.NameWithOwner)
		if ownRepo(nameOwner) {
			log.Printf("Skipping own repo: %s\n", nameOwner)
			continue
		}
		if !merged(pr) {
			log.Printf("Skipping not merged repo: %s\n", nameOwner)
			continue
		}

		repositoryStars[nameOwner] = int(pr.Node.Repository.StargazerCount)
	}

	repositories := make([]repository, 0, len(repositoryStars))
	for repo, star := range repositoryStars {
		repositories = append(repositories, repository{
			OwnerName: repo,
			StarCount: star,
		})
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].StarCount > repositories[j].StarCount
	})

	log.Printf("Total contributed projects: %d\n", len(repositories))

	contribFile, err := os.Create("CONTRIBUTIONS.md")
	if err != nil {
		log.Fatalf("Create file: %v", err)
	}
	defer func() {
		if err := contribFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	_, _ = contribFile.WriteString(`# Open Source Projects I've Ever Contributed

The list of projects (with stars):

`)
	for _, repo := range repositories {
		_, _ = contribFile.WriteString(fmt.Sprintf("* [%s](https://github.com/%s) (%d)\n", repo.OwnerName, repo.OwnerName, repo.StarCount))
	}
}

func PullRequests(ctx context.Context, client *githubv4.Client) ([]edgePullRequest, error) {
	var pullRequests []edgePullRequest
	variables := map[string]any{
		"after": (*githubv4.String)(nil),
	}

	for {
		if err := client.Query(ctx, &queryPullRequest, variables); err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}
		pullRequests = append(pullRequests, queryPullRequest.Viewer.PullRequests.Edges...)
		if !queryPullRequest.Viewer.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["after"] = queryPullRequest.Viewer.PullRequests.PageInfo.EndCursor
	}

	return pullRequests, nil
}

// ownRepo returns true if merged to my github.com/alexandear account.
func ownRepo(nameOwner string) bool {
	return strings.HasPrefix(nameOwner, "alexandear/")
}

func merged(pr edgePullRequest) bool {
	return bool(pr.Node.Merged) || mergedToGolang(pr)
}

// mergedToGolang checks whether closed PR was merged to a repo in "golang" owner.
// A merged PR to is closed by gopherbot if with the comment
// "This PR is being closed because golang.org/cl/463098 has been merged."
func mergedToGolang(pr edgePullRequest) bool {
	if !strings.HasPrefix(string(pr.Node.Repository.NameWithOwner), "golang/") {
		return false
	}

	comments := pr.Node.Comments.Edges
	if len(comments) == 0 {
		return false
	}

	return strings.Contains(string(comments[0].Node.Body), "has been merged")
}
