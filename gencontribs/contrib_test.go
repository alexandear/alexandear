package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shurcooL/githubv4"
)

type mockPlatform struct {
	pullRequests []EdgePullRequest
	starsCount   map[string]int
}

func (m *mockPlatform) PullRequests(ctx context.Context) ([]EdgePullRequest, error) {
	return m.pullRequests, nil
}

func (m *mockPlatform) RepositoryStarsCount(ctx context.Context, ownerName string) (int, error) {
	return m.starsCount[ownerName], nil
}

func TestContributions(t *testing.T) {
	mockP := &mockPlatform{
		pullRequests: []EdgePullRequest{
			{
				Node: struct {
					Repository struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}
					Merged githubv4.Boolean
				}{
					Repository: struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}{
						NameWithOwner:  "golangci/golangci-lint",
						StargazerCount: 16_047,
						GoModContent: `module github.com/golangci/golangci-lint/v2

// The minimum Go version must always be latest-1.
// This version should never be changed outside of the PR to add the support of newer Go version.
// Only golangci-lint maintainers are allowed to change it.
go 1.24.0

require (
	4d63.com/gocheckcompilerdirectives v1.3.0
)
`,
					},
					Merged: true,
				},
			},
			{
				Node: struct {
					Repository struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}
					Merged githubv4.Boolean
				}{
					Repository: struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}{
						NameWithOwner:  "stretchr/testify",
						StargazerCount: 25_635,
						GoModContent: `module github.com/stretchr/testify

// This should match the minimum supported version that is tested in
// .github/workflows/main.yml
go 1.17

require (
	github.com/stretchr/objx v0.5.2 // To avoid a cycle the version of testify used by objx should be excluded below
	gopkg.in/yaml.v3 v3.0.1
)

// Break dependency cycle with objx.
// See https://github.com/stretchr/objx/pull/140
exclude github.com/stretchr/testify v1.8.4
`,
					},
					Merged: true,
				},
			},
			// Not merged repo
			{
				Node: struct {
					Repository struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}
					Merged githubv4.Boolean
				}{
					Repository: struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}{
						NameWithOwner:  "TortoiseGit/TortoiseGit",
						StargazerCount: 1_555,
						GoModContent:   "",
					},
					Merged: false,
				},
			},
			// Own owner
			{
				Node: struct {
					Repository struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}
					Merged githubv4.Boolean
				}{
					Repository: struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}{
						NameWithOwner:  "alexandear/alexandear",
						StargazerCount: 1,
					},
					Merged: true,
				},
			},
			// Own org
			{
				Node: struct {
					Repository struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}
					Merged githubv4.Boolean
				}{
					Repository: struct {
						NameWithOwner  githubv4.String
						StargazerCount githubv4.Int
						GoModContent   githubv4.String
					}{
						NameWithOwner:  "alexandear-org/swag",
						StargazerCount: 1,
					},
					Merged: true,
				},
			},
		},
		starsCount: map[string]int{
			"golangci/golangci-lint": 16_047,
			"stretchr/testify":       25_635,
		},
	}
	top100, err := NewGoTop100()
	if err != nil {
		t.Fatalf("NewGoTop100() error = %v", err)
	}

	repositories, err := Contributions(t.Context(), mockP, top100)
	if err != nil {
		t.Fatalf("Contributions() error = %v", err)
	}

	expected := []Repository{
		{OwnerName: "stretchr/testify", StarCount: 25_635, GoModule: "github.com/stretchr/testify", PlaceInGoTop100: 1},
		{OwnerName: "golangci/golangci-lint", StarCount: 16_047, GoModule: "github.com/golangci/golangci-lint/v2"},
		{OwnerName: "cognitedata/cognite-sdk-python", StarCount: 0},
		{OwnerName: "cue-lang/cue", StarCount: 0},
		{OwnerName: "golang/build", StarCount: 0},
		{OwnerName: "golang/go", StarCount: 0},
		{OwnerName: "golang/mod", StarCount: 0},
		{OwnerName: "golang/net", StarCount: 0},
		{OwnerName: "golang/text", StarCount: 0},
		{OwnerName: "golang/tools", StarCount: 0},
		{OwnerName: "golang/vulndb", StarCount: 0},
		{OwnerName: "golang/website", StarCount: 0},
		{OwnerName: "golang/wiki", StarCount: 0},
		{OwnerName: "google/re2", StarCount: 0},
		{OwnerName: "protocolbuffers/protobuf-go", StarCount: 0},
	}

	if diff := cmp.Diff(expected, repositories); diff != "" {
		t.Errorf("Contributions() mismatch (-want +got):\n%s", diff)
	}
}

func TestContributionReport(t *testing.T) {
	repositories := []Repository{
		{OwnerName: "stretchr/testify", StarCount: 25_635, GoModule: "github.com/stretchr/testify", PlaceInGoTop100: 1},
		{OwnerName: "golangci/golangci-lint", StarCount: 16_047, GoModule: "github.com/golangci/golangci-lint/v2"},
		{OwnerName: "lima-vm/lima", StarCount: 15_965, GoModule: "github.com/lima-vm/lima"},
	}

	var sb strings.Builder
	ContributionReport(&sb, repositories)

	output := sb.String()
	expected := `<!---
Code generated by gencontribs; DO NOT EDIT.

To update the doc:
GITHUB_TOKEN=<YOUR_TOKEN> go run .
-->

# Open Source Projects I've Ever Contributed

See also [APPRECIATIONS](./APPRECIATIONS.md) for testimonials about these contributions.

## Google Go Git Repositories

_links pointed to a log with my contributions_

* [build](https://go.googlesource.com/build/+log?author=Oleksandr%20Redko)
* [go](https://go.googlesource.com/go/+log?author=Oleksandr%20Redko)
* [net](https://go.googlesource.com/net/+log?author=Oleksandr%20Redko)
* [mod](https://go.googlesource.com/mod/+log?author=Oleksandr%20Redko)
* [protobuf](https://go.googlesource.com/protobuf/+log?author=Oleksandr%20Redko)
* [tools](https://go.googlesource.com/tools/+log?author=Oleksandr%20Redko)
* [text](https://go.googlesource.com/text/+log?author=Oleksandr%20Redko)
* [vulndb](https://go.googlesource.com/vulndb/+log?author=Oleksandr%20Redko)
* [website](https://go.googlesource.com/website/+log?author=Oleksandr%20Redko)
* [wiki](https://go.googlesource.com/wiki/+log?author=Oleksandr%20Redko)

## Google Code Git Repositories

_links pointed to a log with my contributions_

* [re2](https://code.googlesource.com/re2/+log?author=Oleksandr%20Redko)

## GitHub Projects

| Project | Stars | Go Module | [Place in Go Top 100](https://blog.thibaut-rousseau.com/blog/the-most-popular-go-dependency-is/) |
|---------|-------|-----------|--------------------------------------------------------------------------------------------------|
| [stretchr/testify](https://github.com/stretchr/testify/commits?author=alexandear) | 25635 | [github.com/stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify) | 1 |
| [golangci/golangci-lint](https://github.com/golangci/golangci-lint/commits?author=alexandear) | 16047 | [github.com/golangci/golangci-lint/v2](https://pkg.go.dev/github.com/golangci/golangci-lint/v2) |  |
| [lima-vm/lima](https://github.com/lima-vm/lima/commits?author=alexandear) | 15965 | [github.com/lima-vm/lima](https://pkg.go.dev/github.com/lima-vm/lima) |  |

## GitLab Projects

_links pointed to a log with my merge requests_

* [gitlab-org/gitlab](https://gitlab.com/gitlab-org/gitlab/-/merge_requests/?sort=updated_desc&state=all&author_username=alexandear)
* [gitlab-org/api/client-go](https://gitlab.com/gitlab-org/api/client-go/-/merge_requests/?sort=updated_desc&state=all&author_username=alexandear)
* [gitlab-org/cli](https://gitlab.com/gitlab-org/cli/-/merge_requests/?sort=updated_desc&state=all&author_username=alexandear)
* [bosi/decorder](https://gitlab.com/bosi/decorder/-/merge_requests/?sort=updated_desc&state=all&author_username=alexandear)
`

	if diff := cmp.Diff(expected, output); diff != "" {
		t.Errorf("ContributionReport() mismatch (-want +got):\n%s", diff)
	}
}
