// gencontribs prints a list of open source projects I've ever contributed to CONTRIBUTING.md.
//
// Usage:
//
//	GITHUB_TOKEN=<YOUR_TOKEN> go run .
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	genTimeout = 2 * time.Minute
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Panic("environment variable 'GITHUB_TOKEN' must be non-empty and has permissions '[pull-requests: read]'")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	ctx, cancel := context.WithTimeout(context.Background(), genTimeout)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, src)
	client := githubv4.NewClient(httpClient)
	gh := &GitHub{client: client}

	top100, err := NewGoTop100()
	if err != nil {
		log.Panicf("Load Go top 100: %v\n", err)
	}

	repositories, err := Contributions(ctx, gh, top100)
	if err != nil {
		log.Panicf("Get contributions: %v\n", err)
	}

	log.Printf("Total contributed projects: %d\n", len(repositories))

	contribFile, err := os.Create(filepath.Join("..", "CONTRIBUTIONS.md"))
	if err != nil {
		log.Panicf("Create file: %v", err)
	}
	defer func() {
		if err := contribFile.Close(); err != nil {
			log.Printf("Failed to close contrib file: %v\n", err)
		}
	}()

	ContributionReport(contribFile, repositories)
}
