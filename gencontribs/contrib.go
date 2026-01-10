package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"log"
	"slices"
	"strconv"
	"strings"
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
	{"wiki", "golang/wiki"},
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

// gitlabRepos holds my contributions to repos located at https://gitlab.com.
var gitlabRepos = []string{
	"gitlab-org/gitlab",
	"gitlab-org/api/client-go",
	"gitlab-org/cli",
	"bosi/decorder",
}

type Repository struct {
	OwnerName string
	StarCount int

	GoModule        string
	PlaceInGoTop100 int
}

type ContribPlatform interface {
	PullRequests(ctx context.Context) ([]EdgePullRequest, error)
	RepositoryStarsCount(ctx context.Context, ownerName string) (int, error)
}

type GoTop100 interface {
	Place(moduleName string) int
}

func Contributions(ctx context.Context, platform ContribPlatform, top100 GoTop100) ([]Repository, error) {
	allPullRequests, err := platform.PullRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("get merged pull requests: %w", err)
	}
	log.Printf("Total pull request: %d\n", len(allPullRequests))

	type starredRepo struct {
		starCount int
		goModule  string
	}

	starredRepositories := map[string]starredRepo{}
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

		starredRepositories[ownerName] = starredRepo{
			starCount: int(pr.Node.Repository.StargazerCount),
			goModule:  extractGoModuleName(string(pr.Node.Repository.GoModContent)),
		}
	}

	getStarsCount := func(ownerName string) starredRepo {
		starsCount, err := platform.RepositoryStarsCount(ctx, ownerName)
		if err != nil {
			log.Printf("Failed to get repository %q stars: %v", ownerName, err)
			starsCount = 1000
		}
		return starredRepo{
			starCount: starsCount,
		}
	}

	fillGoogleStarsCount := func(googleRepos []googleSourceGitHub) {
		for _, googleGithub := range googleRepos {
			ownerName := googleGithub.GitHubOwnerName
			getStarsCount(ownerName)
			starredRepositories[ownerName] = getStarsCount(ownerName)
		}
	}

	fillGoogleStarsCount(googleGoGitHubRepos)
	fillGoogleStarsCount(googleCodeGitHubRepos)

	for _, ownerName := range additionalGitHubRepos {
		starredRepositories[ownerName] = getStarsCount(ownerName)
	}

	repositories := make([]Repository, 0, len(starredRepositories))
	for ownerName, repo := range starredRepositories {
		repositories = append(repositories, Repository{
			OwnerName:       ownerName,
			StarCount:       repo.starCount,
			GoModule:        repo.goModule,
			PlaceInGoTop100: top100.Place(repo.goModule),
		})
	}

	return slices.SortedStableFunc(slices.Values(repositories), func(a, b Repository) int {
		return cmp.Or(
			-cmp.Compare(a.StarCount, b.StarCount),
			cmp.Compare(a.OwnerName, b.OwnerName),
		)
	}), nil
}

// ownRepo returns true if merged to my github.com/alexandear account.
func ownRepo(ownerName string) bool {
	return strings.HasPrefix(ownerName, "alexandear/") || strings.HasPrefix(ownerName, "alexandear-org/")
}

// extractGoModuleName extracts the module name from go.mod content.
func extractGoModuleName(goModContent string) string {
	for line := range strings.Lines(goModContent) {
		module, ok := strings.CutPrefix(strings.TrimSpace(line), "module")
		if ok {
			module = strings.TrimSpace(module)
			if module == "github.com/YOUR-USER-OR-ORG-NAME/YOUR-REPO-NAME" {
				// special case for https://github.com/golang-standards/projec-layout/blob/HEAD/go.mod#L1
				return ""
			}
			return module
		}
	}
	return ""
}

// ContributionReport writes the list of repositories to out.
// Assuming WriteString never fails.
func ContributionReport(out io.StringWriter, repositories []Repository) {
	out.WriteString(`<!---
Code generated by gencontribs; DO NOT EDIT.

To update the doc:
GITHUB_TOKEN=<YOUR_TOKEN> go run .
-->

# Open Source Projects I've Ever Contributed

See also [APPRECIATIONS](./APPRECIATIONS.md) for testimonials about these contributions.
`)

	out.WriteString(`
## Google Go Git Repositories

_links pointed to a log with my contributions_

`)
	for _, repo := range googleGoGitHubRepos {
		line := fmt.Sprintf("* [%[1]s](https://go.googlesource.com/%[1]s/+log?author=Oleksandr%%20Redko)\n", repo.GoogleSourceRepo)
		out.WriteString(line)
	}

	out.WriteString(`
## Google Code Git Repositories

_links pointed to a log with my contributions_

`)
	for _, repo := range googleCodeGitHubRepos {
		line := fmt.Sprintf("* [%[1]s](https://code.googlesource.com/%[1]s/+log?author=Oleksandr%%20Redko)\n", repo.GoogleSourceRepo)
		out.WriteString(line)
	}

	out.WriteString(`
## GitHub Projects

| Project | Stars | Go Module | [Place in Go Top 100](https://blog.thibaut-rousseau.com/blog/the-most-popular-go-dependency-is/) |
|---------|-------|-----------|--------------------------------------------------------------------------------------------------|
`)
	for _, repo := range repositories {
		var place string
		if repo.PlaceInGoTop100 > 0 {
			place = strconv.Itoa(repo.PlaceInGoTop100)
		}
		line := fmt.Sprintf("| [%[1]s](https://github.com/%[1]s/commits?author=alexandear) | %d | [%[3]s](https://pkg.go.dev/%[3]s) | %s |\n",
			repo.OwnerName, repo.StarCount, repo.GoModule, place)
		out.WriteString(line)
	}

	out.WriteString(`
## GitLab Projects

_links pointed to a log with my merge requests_

`)
	for _, repo := range gitlabRepos {
		line := fmt.Sprintf("* [%[1]s](https://gitlab.com/%[1]s/-/merge_requests/?sort=updated_desc&state=all&author_username=alexandear)\n", repo)
		out.WriteString(line)
	}
}
