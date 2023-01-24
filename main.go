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

type edgePullRequest struct {
	Node struct {
		Repository struct {
			NameWithOwner  githubv4.String
			StargazerCount githubv4.Int
		}
	}
}

var queryMergedPRs struct {
	Viewer struct {
		PullRequests struct {
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
			TotalCount githubv4.Int
			Edges      []edgePullRequest
		} `graphql:"pullRequests(states:MERGED, orderBy:{field: CREATED_AT, direction: ASC}, first:100, after: $after)"`
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

	var allPullRequests []edgePullRequest
	variables := map[string]any{
		"after": (*githubv4.String)(nil),
	}
	for {
		if err := client.Query(context.Background(), &queryMergedPRs, variables); err != nil {
			log.Fatalf("Executing query: %v", err)
		}
		allPullRequests = append(allPullRequests, queryMergedPRs.Viewer.PullRequests.Edges...)
		if !queryMergedPRs.Viewer.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["after"] = queryMergedPRs.Viewer.PullRequests.PageInfo.EndCursor
	}

	log.Printf("Total merged pull request: %d\n", len(allPullRequests))

	repositoryStars := map[string]int{}
	for _, pr := range allPullRequests {
		nameOwner := string(pr.Node.Repository.NameWithOwner)
		if strings.HasPrefix(nameOwner, "alexandear") {
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

The list of projects sorted desc by stars:

`)
	for _, repo := range repositories {
		_, _ = contribFile.WriteString(fmt.Sprintf("* [%s](https://github.com/%s)\n", repo.OwnerName, repo.OwnerName))
	}
}
