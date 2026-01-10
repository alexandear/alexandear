package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

type Querier interface {
	Query(ctx context.Context, q any, variables map[string]any) error
}

type GitHub struct {
	client Querier
}

type EdgePullRequest struct {
	Node struct {
		Repository struct {
			NameWithOwner  githubv4.String
			StargazerCount githubv4.Int
			GoModContent   githubv4.String
		}
		Merged githubv4.Boolean
	}
}

func (gh *GitHub) PullRequests(ctx context.Context) ([]EdgePullRequest, error) {
	var pullRequests []EdgePullRequest
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
					Edges      []struct {
						Node struct {
							Repository struct {
								NameWithOwner  githubv4.String
								StargazerCount githubv4.Int
								Object         struct {
									Blob struct {
										Text githubv4.String
									} `graphql:"... on Blob"`
								} `graphql:"object(expression: \"HEAD:go.mod\")"`
							}
							Merged githubv4.Boolean
						}
					}
				} `graphql:"pullRequests(states: [MERGED, CLOSED], orderBy:{field: CREATED_AT, direction: ASC}, first:100, after: $after)"`
			}
		}

		if err := gh.client.Query(ctx, &queryPullRequest, variables); err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}

		for _, edge := range queryPullRequest.Viewer.PullRequests.Edges {
			pullRequests = append(pullRequests, EdgePullRequest{
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
						NameWithOwner:  edge.Node.Repository.NameWithOwner,
						StargazerCount: edge.Node.Repository.StargazerCount,
						GoModContent:   edge.Node.Repository.Object.Blob.Text,
					},
					Merged: edge.Node.Merged,
				},
			})
		}

		if !queryPullRequest.Viewer.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["after"] = queryPullRequest.Viewer.PullRequests.PageInfo.EndCursor
	}

	return pullRequests, nil
}

func (gh *GitHub) RepositoryStarsCount(ctx context.Context, ownerName string) (int, error) {
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

	if err := gh.client.Query(ctx, &queryRepository, variables); err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}

	return int(queryRepository.Repository.StargazerCount), nil
}
