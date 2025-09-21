package forge

import (
    "context"

    "github.com/google/go-github/v58/github"
    "golang.org/x/oauth2"
)

type Repository struct {
    Name        string
    Description string
    CloneURL    string
    SSHURL      string
    Language    string
    Stars       int
    Size        int  // in KB
}

type Client struct {
    client *github.Client
}

func NewClient(token string) *Client {
    var client *github.Client
    
    if token != "" {
        ts := oauth2.StaticTokenSource(
            &oauth2.Token{AccessToken: token},
        )
        tc := oauth2.NewClient(context.Background(), ts)
        client = github.NewClient(tc)
    } else {
        client = github.NewClient(nil)
    }

    return &Client{client: client}
}

func (c *Client) DiscoverRepositories(ctx context.Context, 
                                     owner string) ([]Repository, error) {
    opt := &github.RepositoryListOptions{
        ListOptions: github.ListOptions{PerPage: 100},
        Sort:        "updated", // Sort by most recently updated
        Direction:   "desc",
    }

    var allRepos []Repository

    for {
        repos, resp, err := c.client.Repositories.List(ctx, owner, opt)
        if err != nil {
            return nil, err
        }

        for _, repo := range repos {
            if repo.GetFork() {
                continue // Skip forks by default
            }

            language := repo.GetLanguage()
            if language == "" {
                language = "Unknown"
            }

            allRepos = append(allRepos, Repository{
                Name:        repo.GetName(),
                Description: repo.GetDescription(),
                CloneURL:    repo.GetCloneURL(),
                SSHURL:      repo.GetSSHURL(),
                Language:    language,
                Stars:       repo.GetStargazersCount(),
                Size:        repo.GetSize(),
            })
        }

        if resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }

    return allRepos, nil
}