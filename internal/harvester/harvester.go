package harvester

import (
    "fmt"
    "os"

    "github.com/go-git/go-git/v5"
)

type Harvester struct {
    recursive bool
}

func New(recursive bool) *Harvester {
    return &Harvester{recursive: recursive}
}

func (h *Harvester) HarvestRepository(url, path string) error {
    // Check if directory already exists
    if _, err := os.Stat(path); err == nil {
        return fmt.Errorf("directory %s already exists", path)
    }

    // Clone options
    cloneOptions := &git.CloneOptions{
        URL:      url,
        Progress: os.Stdout,
    }

    // Harvest the repository
    repo, err := git.PlainClone(path, false, cloneOptions)
    if err != nil {
        return fmt.Errorf("failed to harvest: %w", err)
    }

    // Handle submodules if recursive is enabled
    if h.recursive {
        if err := h.cultivateSubmodules(repo, path); err != nil {
            fmt.Printf("  WARNING: failed to cultivate submodules: %v\n", err)
        }
    }

    return nil
}

func (h *Harvester) cultivateSubmodules(repo *git.Repository, repoPath string) error {
    worktree, err := repo.Worktree()
    if err != nil {
        return err
    }

    // Get submodules
    submodules, err := worktree.Submodules()
    if err != nil {
        return err
    }

    if len(submodules) == 0 {
        return nil // No submodules
    }

    fmt.Printf("  Cultivating %d submodule(s)...\n", len(submodules))

    for _, submodule := range submodules {
        fmt.Printf("    - %s\n", submodule.Config().Name)
        if err := submodule.Update(&git.SubmoduleUpdateOptions{
            Init: true,
            RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
        }); err != nil {
            return fmt.Errorf("failed to cultivate submodule %s: %w", 
                submodule.Config().Name, err)
        }
    }

    return nil
}