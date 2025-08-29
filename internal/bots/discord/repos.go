package discord

import (
	"context"
	"fmt"

	"github.com/Formula-SAE/discord/internal/db"
	"github.com/google/go-github/v74/github"
)

func (b *DiscordBot) updateRepositoriesInDB(ctx context.Context) ([]string, error) {
	githubRepos, _, err := b.gc.Repositories.ListByOrg(ctx, "ApexCorse", &github.RepositoryListByOrgOptions{
		Sort: "full_name",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories from GitHub organization 'ApexCorse': %w", err)
	}

	for _, repo := range githubRepos {
		dbRepo := &db.Repository{
			Name: repo.GetName(),
		}

		b.db.CreateRepository(dbRepo)
	}

	githubRepoNames := make([]string, len(githubRepos))
	for i, repo := range githubRepos {
		githubRepoNames[i] = repo.GetName()
	}

	return githubRepoNames, nil
}
