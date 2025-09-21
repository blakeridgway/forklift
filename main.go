package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "forklift/internal/harvester"
    "forklift/internal/forge"

    "github.com/spf13/cobra"
)

var (
    token       string
    destination string
    recursive   bool
    useHTTPS    bool
    filterLang  string
)

func main() {
    var rootCmd = &cobra.Command{
        Use:   "forklift [username/organization]",
        Short: "Harvest GitHub repositories from the open source commons",
        Long: `Forklift - A tool to harvest and organize open source repositories.
Select which repositories to collect and where to cultivate them locally.
Uses SSH by default for easy development workflow.`,
        Args: cobra.ExactArgs(1),
        Run:  runForklift,
    }

    rootCmd.Flags().StringVarP(&token, "token", "t", "", 
        "GitHub personal access token (optional for public repos)")
    rootCmd.Flags().StringVarP(&destination, "dest", "d", "", 
        "Destination directory (will prompt if not provided)")
    rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", true, 
        "Harvest repositories recursively with submodules")
    rootCmd.Flags().BoolVar(&useHTTPS, "https", false, 
        "Use HTTPS instead of SSH for cloning (useful for read-only access)")
    rootCmd.Flags().StringVarP(&filterLang, "language", "l", "", 
        "Filter repositories by programming language (e.g., Go, Python, JavaScript)")

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func runForklift(cmd *cobra.Command, args []string) {
    owner := args[0]

    // Check SSH setup if not using HTTPS
    if !useHTTPS {
        checkSSHSetup()
    }

    // Initialize forge client
    client := forge.NewClient(token)

    // Discover repositories
    fmt.Printf("Discovering repositories for %s...\n", owner)
    repos, err := client.DiscoverRepositories(context.Background(), owner)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error discovering repositories: %v\n", err)
        os.Exit(1)
    }

    if len(repos) == 0 {
        fmt.Println("No repositories found in the commons.")
        return
    }

    // Filter by language if specified
    if filterLang != "" {
        repos = filterByLanguage(repos, filterLang)
        if len(repos) == 0 {
            fmt.Printf("No repositories found for language: %s\n", filterLang)
            return
        }
    }

    // Let user select repositories
    selectedRepos := selectRepositories(repos)
    if len(selectedRepos) == 0 {
        fmt.Println("No repositories selected for harvest.")
        return
    }

    // Get destination directory
    destDir := getDestinationDirectory()

    // Initialize harvester
    h := harvester.New(recursive)

    // Harvest selected repositories
    fmt.Printf("\nHarvesting %d repositories to %s...\n", 
        len(selectedRepos), destDir)
    
    if useHTTPS {
        fmt.Println("Using HTTPS for cloning (read-only friendly)")
    } else {
        fmt.Println("Using SSH for cloning (development ready)")
    }
    
    for i, repo := range selectedRepos {
        fmt.Printf("\n[%d/%d] Harvesting %s (%s)...\n", 
            i+1, len(selectedRepos), repo.Name, repo.Language)
        
        repoPath := filepath.Join(destDir, repo.Name)
        
        // Choose URL based on protocol preference
        cloneURL := repo.SSHURL
        if useHTTPS {
            cloneURL = repo.CloneURL
        }
        
        if err := h.HarvestRepository(cloneURL, repoPath); err != nil {
            fmt.Printf("  FAILED to harvest %s: %v\n", repo.Name, err)
            
            // If SSH fails, offer to retry with HTTPS
            if !useHTTPS && strings.Contains(err.Error(), "ssh") {
                fmt.Printf("  SSH failed, trying HTTPS for %s...\n", repo.Name)
                if retryErr := h.HarvestRepository(repo.CloneURL, repoPath); retryErr != nil {
                    fmt.Printf("  HTTPS also failed for %s: %v\n", repo.Name, retryErr)
                } else {
                    fmt.Printf("  Successfully harvested %s via HTTPS\n", repo.Name)
                }
            }
            continue
        }
        fmt.Printf("  Successfully harvested %s\n", repo.Name)
    }

    fmt.Println("\nHarvest completed! Your open source garden is ready.")
    if !useHTTPS {
        fmt.Println("NOTE: Repositories are cloned via SSH - ready for development and contributions!")
    }
}

func checkSSHSetup() {
    // Check if SSH key exists
    homeDir, _ := os.UserHomeDir()
    sshDir := filepath.Join(homeDir, ".ssh")
    
    keyFiles := []string{"id_rsa", "id_ed25519", "id_ecdsa"}
    hasKey := false
    
    for _, keyFile := range keyFiles {
        if _, err := os.Stat(filepath.Join(sshDir, keyFile)); err == nil {
            hasKey = true
            break
        }
    }
    
    if !hasKey {
        fmt.Println("WARNING: No SSH keys found in ~/.ssh/")
        fmt.Println("Consider setting up SSH keys for GitHub:")
        fmt.Println("   https://docs.github.com/en/authentication/connecting-to-github-with-ssh")
        fmt.Println("   Or use --https flag for read-only access")
        fmt.Print("\nContinue anyway? (y/N): ")
        
        reader := bufio.NewReader(os.Stdin)
        response, _ := reader.ReadString('\n')
        response = strings.TrimSpace(strings.ToLower(response))
        
        if response != "y" && response != "yes" {
            fmt.Println("Exiting. Set up SSH keys or use --https flag.")
            os.Exit(1)
        }
    } else {
        // Test SSH connection to GitHub
        if testSSHConnection() {
            fmt.Println("SSH connection to GitHub verified!")
        } else {
            fmt.Println("WARNING: SSH connection to GitHub failed.")
            fmt.Println("You may need to add your SSH key to GitHub or use --https flag")
            fmt.Print("\nContinue anyway? (y/N): ")
            
            reader := bufio.NewReader(os.Stdin)
            response, _ := reader.ReadString('\n')
            response = strings.TrimSpace(strings.ToLower(response))
            
            if response != "y" && response != "yes" {
                fmt.Println("Exiting. Fix SSH setup or use --https flag.")
                os.Exit(1)
            }
        }
    }
}

func testSSHConnection() bool {
    cmd := exec.Command("ssh", "-T", "git@github.com")
    err := cmd.Run()
    // SSH to GitHub returns exit code 1 on successful auth (by design)
    // Exit code 255 usually means connection failed
    if exitError, ok := err.(*exec.ExitError); ok {
        return exitError.ExitCode() == 1
    }
    return err == nil
}

func filterByLanguage(repos []forge.Repository, language string) []forge.Repository {
    var filtered []forge.Repository
    targetLang := strings.ToLower(language)
    
    for _, repo := range repos {
        if strings.ToLower(repo.Language) == targetLang {
            filtered = append(filtered, repo)
        }
    }
    
    fmt.Printf("Filtered to %d repositories using %s\n", len(filtered), language)
    return filtered
}

func selectRepositories(repos []forge.Repository) []forge.Repository {
    // Show language distribution
    showLanguageStats(repos)
    
    fmt.Printf("\nFound %d repositories in the commons:\n\n", len(repos))
    
    // Format the display with language info
    for i, repo := range repos {
        description := repo.Description
        if description == "" {
            description = "No description available"
        }
        
        // Truncate description if too long
        if len(description) > 50 {
            description = description[:47] + "..."
        }
        
        // Format stars
        stars := ""
        if repo.Stars > 0 {
            if repo.Stars >= 1000 {
                stars = fmt.Sprintf("\u2605%.1fk", float64(repo.Stars)/1000)
            } else {
                stars = fmt.Sprintf("\u2605%d", repo.Stars)
            }
        }
        
        // Display language
        langDisplay := getLanguageDisplay(repo.Language)
        
        fmt.Printf("%2d. %-25s %-12s %-8s %s\n", 
            i+1, repo.Name, langDisplay, stars, description)
    }

    fmt.Println("\nHarvest selection options:")
    fmt.Println("  - Enter numbers separated by commas (e.g., 1,3,5)")
    fmt.Println("  - Enter ranges (e.g., 1-5)")
    fmt.Println("  - Enter 'all' to harvest all repositories")
    fmt.Println("  - Enter 'dev' to select only development repositories")
    fmt.Println("  - Enter a language name (e.g., 'go', 'python') to filter by language")
    fmt.Println("  - Press Enter to finish selection")

    reader := bufio.NewReader(os.Stdin)
    var selected []forge.Repository

    for {
        fmt.Print("\nEnter your harvest selection: ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        if input == "" {
            break
        }

        if input == "all" {
            return repos
        }

        if input == "dev" {
            return selectDevRepositories(repos)
        }

        // Check if input is a language filter
        if langFiltered := tryLanguageFilter(repos, input); langFiltered != nil {
            return selectRepositories(langFiltered)
        }

        indices := parseSelection(input, len(repos))
        if len(indices) > 0 {
            for _, idx := range indices {
                if idx >= 0 && idx < len(repos) {
                    // Check if already selected
                    alreadySelected := false
                    for _, s := range selected {
                        if s.Name == repos[idx].Name {
                            alreadySelected = true
                            break
                        }
                    }
                    if !alreadySelected {
                        selected = append(selected, repos[idx])
                        fmt.Printf("  Added to harvest: %s (%s)\n", 
                            repos[idx].Name, repos[idx].Language)
                    }
                }
            }
        }

        if len(selected) > 0 {
            fmt.Printf("\nCurrently selected %d repositories for harvest.\n", 
                len(selected))
        }
    }

    return selected
}

func showLanguageStats(repos []forge.Repository) {
    langCount := make(map[string]int)
    
    for _, repo := range repos {
        lang := repo.Language
        if lang == "" {
            lang = "Unknown"
        }
        langCount[lang]++
    }
    
    // Convert to slice for sorting
    type langStat struct {
        name  string
        count int
    }
    
    var stats []langStat
    for lang, count := range langCount {
        stats = append(stats, langStat{lang, count})
    }
    
    // Sort by count descending
    sort.Slice(stats, func(i, j int) bool {
        return stats[i].count > stats[j].count
    })
    
    fmt.Printf("\nLanguage distribution:\n")
    for _, stat := range stats {
        if stat.count == 1 {
            fmt.Printf("   %s: %d repository\n", stat.name, stat.count)
        } else {
            fmt.Printf("   %s: %d repositories\n", stat.name, stat.count)
        }
    }
}

func getLanguageDisplay(language string) string {
    if language == "" || language == "Unknown" {
        return "Unknown"
    }
    
    // Just return the language name without icons
    return language
}

func tryLanguageFilter(repos []forge.Repository, input string) []forge.Repository {
    inputLower := strings.ToLower(input)
    
    // Check if any repositories match this language
    var matches []forge.Repository
    for _, repo := range repos {
        if strings.ToLower(repo.Language) == inputLower {
            matches = append(matches, repo)
        }
    }
    
    if len(matches) > 0 {
        fmt.Printf("Filtered to %d %s repositories\n", len(matches), input)
        return matches
    }
    
    return nil
}

func selectDevRepositories(repos []forge.Repository) []forge.Repository {
    var devRepos []forge.Repository
    
    // Filter for repositories that are likely development projects
    excludeKeywords := []string{"docs", "documentation", "website", "blog", "awesome-"}
    // Also exclude certain languages that are typically not development projects
    excludeLanguages := []string{"html", "css"}
    
    for _, repo := range repos {
        isDevRepo := true
        lowerName := strings.ToLower(repo.Name)
        lowerDesc := strings.ToLower(repo.Description)
        lowerLang := strings.ToLower(repo.Language)
        
        // Skip if it matches exclude patterns
        for _, keyword := range excludeKeywords {
            if strings.Contains(lowerName, keyword) || strings.Contains(lowerDesc, keyword) {
                isDevRepo = false
                break
            }
        }
        
        // Skip if it's an excluded language
        for _, excludeLang := range excludeLanguages {
            if lowerLang == excludeLang {
                isDevRepo = false
                break
            }
        }
        
        if isDevRepo {
            devRepos = append(devRepos, repo)
        }
    }
    
    fmt.Printf("Selected %d repositories that appear to be development projects\n", len(devRepos))
    return devRepos
}

func parseSelection(input string, maxLen int) []int {
    var indices []int
    parts := strings.Split(input, ",")

    for _, part := range parts {
        part = strings.TrimSpace(part)
        
        if strings.Contains(part, "-") {
            // Handle range
            rangeParts := strings.Split(part, "-")
            if len(rangeParts) == 2 {
                start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
                end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
                
                if err1 == nil && err2 == nil && 
                   start > 0 && end > 0 && 
                   start <= maxLen && end <= maxLen && 
                   start <= end {
                    for i := start; i <= end; i++ {
                        indices = append(indices, i-1) // Convert to 0-based
                    }
                }
            }
        } else {
            // Handle single number
            if num, err := strconv.Atoi(part); err == nil && 
               num > 0 && num <= maxLen {
                indices = append(indices, num-1) // Convert to 0-based
            }
        }
    }

    return indices
}

func getDestinationDirectory() string {
    if destination != "" {
        if err := os.MkdirAll(destination, 0755); err != nil {
            fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", 
                destination, err)
            os.Exit(1)
        }
        return destination
    }

    reader := bufio.NewReader(os.Stdin)
    
    for {
        fmt.Print("\nEnter destination directory for your harvest (or press Enter for current directory): ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        if input == "" {
            cwd, _ := os.Getwd()
            return cwd
        }

        // Expand home directory
        if strings.HasPrefix(input, "~/") {
            home, _ := os.UserHomeDir()
            input = filepath.Join(home, input[2:])
        }

        // Convert to absolute path
        absPath, err := filepath.Abs(input)
        if err != nil {
            fmt.Printf("Invalid path: %v\n", err)
            continue
        }

        // Create directory if it doesn't exist
        if err := os.MkdirAll(absPath, 0755); err != nil {
            fmt.Printf("Error creating directory: %v\n", err)
            continue
        }

        return absPath
    }
}