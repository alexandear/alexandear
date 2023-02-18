package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

//go:generate go run gen.go

// additionalContribRepositories holds GitHub's 'owner/name' of contributed repositories that have mirrors on GitHub
// but accept pull requests in another way, e.g. through Gerrit Code Review.
var additionalContribRepositories = []string{
	"protocolbuffers/protobuf-go", // https://go.googlesource.com/protobuf
	"golang/build",                // https://go.googlesource.com/build
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
		ownerName := string(pr.Node.Repository.NameWithOwner)
		if ownRepo(ownerName) {
			log.Printf("Skipping own repo: %s\n", ownerName)
			continue
		}
		if !merged(pr) {
			log.Printf("Skipping not merged repo: %s\n", ownerName)
			continue
		}

		repositoryStars[ownerName] = int(pr.Node.Repository.StargazerCount)
	}

	for _, ownerName := range additionalContribRepositories {
		starsCount, err := RepositoryStarsCount(context.Background(), client, ownerName)
		if err != nil {
			log.Printf("Failed to get repository %q stars: %v", ownerName, err)
			starsCount = 1000
		}
		repositoryStars[ownerName] = starsCount
	}

	repositories := make([]repository, 0, len(repositoryStars))
	for ownerName, star := range repositoryStars {
		repositories = append(repositories, repository{
			OwnerName: ownerName,
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

	_, _ = contribFile.WriteString(`<!---
Code generated by gen.go; DO NOT EDIT.

To update the doc run:
GITHUB_TOKEN=<YOUR_TOKEN> go generate ./...
-->

# Open Source Projects I've Ever Contributed

`)
	_, _ = contribFile.WriteString(fmt.Sprintf(" _Updated %s_", time.Now().UTC().Format("_2 Jan 2006 15:04")))
	_, _ = contribFile.WriteString(`

The list of projects sorted by stars:

`)
	for _, repo := range repositories {
		line := fmt.Sprintf("* [%s](https://github.com/%s)\n", repo.OwnerName, repo.OwnerName)
		_, _ = contribFile.WriteString(line)
	}
}

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

func PullRequests(ctx context.Context, client *githubv4.Client) ([]edgePullRequest, error) {
	var pullRequests []edgePullRequest
	variables := map[string]any{
		"after": (*githubv4.String)(nil),
	}

	for {
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

func RepositoryStarsCount(ctx context.Context, client *githubv4.Client, ownerName string) (int, error) {
	spl := strings.Split(ownerName, "/")
	if len(spl) != 2 {
		return 0, fmt.Errorf("repo %s must have format 'owner/name'", ownerName)
	}
	owner, name := spl[0], spl[1]

	variables := map[string]any{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}

	var queryRepository struct {
		Repository struct {
			StargazerCount githubv4.Int
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	if err := client.Query(ctx, &queryRepository, variables); err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}

	return int(queryRepository.Repository.StargazerCount), nil
}

// ownRepo returns true if merged to my github.com/alexandear account.
func ownRepo(ownerName string) bool {
	return strings.HasPrefix(ownerName, "alexandear/")
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
