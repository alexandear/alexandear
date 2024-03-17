package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

//go:generate go run $GOFILE

const (
	genTimeout = 30 * time.Second
)

// googleSourceGitHub holds mapping of
// a Google Git repository name https://*.googlesource.com/<GoogleSourceRepo>
// to GitHub owner name https://github.com/<GitHubOwnerName>.
type googleSourceGitHub struct {
	GoogleSourceRepo string
	GitHubOwnerName  string
}

// googleGoGitHubRepos are Go Google Git repositories located at https://go.googlesource.com
// to which I have contributed.
var googleGoGitHubRepos = []googleSourceGitHub{
	{"build", "golang/build"},
	{"go", "golang/go"},
	{"net", "golang/net"},
	{"mod", "golang/mod"},
	{"protobuf", "protocolbuffers/protobuf-go"},
	{"tools", "golang/tools"},
	{"text", "golang/text"},
	{"vulndb", "golang/vulndb"},
	{"website", "golang/website"},
}

// googleCodeGitHubRepos are Code Google Git repositories located at https://code.googlesource.com/
// to which I have contributed.
var googleCodeGitHubRepos = []googleSourceGitHub{
	{"re2", "google/re2"},
}

// additionalGitHubRepos holds GitHub repositories to which I have contributed.
// Some of them are not hosted on GitHub, but on Gerrit, and GitHub is a mirror.
var additionalGitHubRepos = []string{
	"cue-lang/cue",                   // https://review.gerrithub.io/q/project:cue-lang%252Fcue
	"cognitedata/cognite-sdk-python", // https://github.com/cognitedata/cognite-sdk-python/pull/1400
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("environment variable 'GITHUB_TOKEN' must be non-empty and has permissions '[pull-requests: read]'")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	ctx, cancel := context.WithTimeout(context.Background(), genTimeout)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)

	allPullRequests, err := pullRequests(ctx, client)
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
		if !pr.Node.Merged {
			log.Printf("Skipping not merged repo: %s\n", ownerName)
			continue
		}

		repositoryStars[ownerName] = int(pr.Node.Repository.StargazerCount)
	}

	getStarsCount := func(ownerName string) int {
		starsCount, err := repositoryStarsCount(ctx, client, ownerName)
		if err != nil {
			log.Printf("Failed to get repository %q stars: %v", ownerName, err)
			return 1000
		}
		return starsCount
	}

	fillGoogleStarsCount := func(googleRepos []googleSourceGitHub) {
		for _, googleGithub := range googleRepos {
			ownerName := googleGithub.GitHubOwnerName
			getStarsCount(ownerName)
			repositoryStars[ownerName] = getStarsCount(ownerName)
		}
	}

	fillGoogleStarsCount(googleGoGitHubRepos)
	fillGoogleStarsCount(googleCodeGitHubRepos)

	for _, ownerName := range additionalGitHubRepos {
		repositoryStars[ownerName] = getStarsCount(ownerName)
	}

	type repository struct {
		OwnerName string
		StarCount int
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

	contribFile, err := os.Create(filepath.Join("..", "CONTRIBUTIONS.md"))
	if err != nil {
		log.Fatalf("Create file: %v", err)
	}
	defer func() {
		if err := contribFile.Close(); err != nil {
			log.Fatalf("Failed to close contrib file: %v\n", err)
		}
	}()
	_, _ = contribFile.WriteString(`<!---
Code generated by gen.go; DO NOT EDIT.

To update the doc run:
GITHUB_TOKEN=<YOUR_TOKEN> go generate ./...
-->

# Open Source Projects I've Ever Contributed
`)

	_, _ = contribFile.WriteString(`
## Google Go Git Repositories

_links pointed to a log with my contributions_

`)
	for _, repo := range googleGoGitHubRepos {
		line := fmt.Sprintf("* [%[1]s](https://go.googlesource.com/%[1]s/+log?author=Oleksandr%%20Redko)\n", repo.GoogleSourceRepo)
		_, _ = contribFile.WriteString(line)
	}

	_, _ = contribFile.WriteString(`
## Google Code Git Repositories

_links pointed to a log with my contributions_

`)
	for _, repo := range googleCodeGitHubRepos {
		line := fmt.Sprintf("* [%[1]s](https://code.googlesource.com/%[1]s/+log?author=Oleksandr%%20Redko)\n", repo.GoogleSourceRepo)
		_, _ = contribFile.WriteString(line)
	}

	_, _ = contribFile.WriteString(`
## GitHub Projects

_sorted by stars descending_

`)
	for _, repo := range repositories {
		line := fmt.Sprintf("* [%[1]s](https://github.com/%[1]s)\n", repo.OwnerName)
		_, _ = contribFile.WriteString(line)
	}
}

type edgePullRequest struct {
	Node struct {
		Repository struct {
			NameWithOwner  githubv4.String
			StargazerCount githubv4.Int
		}
		Merged githubv4.Boolean
	}
}

func pullRequests(ctx context.Context, client *githubv4.Client) ([]edgePullRequest, error) {
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

func repositoryStarsCount(ctx context.Context, client *githubv4.Client, ownerName string) (int, error) {
	owner, name, ok := strings.Cut(ownerName, "/")
	if !ok || owner == "" || name == "" {
		return 0, fmt.Errorf("repo %s must have format 'owner/name'", ownerName)
	}

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
