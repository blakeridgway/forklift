# Forklift

A command-line tool for harvesting open source repositories from GitHub with interactive selection and organized cultivation. Optimized for development workflows with SSH cloning by default.

## Features

- **SSH-first approach**: Clones via SSH by default for easy push/pull workflows
- **Language detection**: See the main programming language for each repository
- **Smart filtering**: Filter by language, development projects, or custom criteria
- Harvest repositories from any GitHub user or organization  
- Interactive repository selection with development-focused filtering
- Automatic SSH setup verification
- HTTPS fallback for read-only access
- Recursive harvesting with submodules

## Installation

```bash
git clone <this-repo>
cd forklift
go mod tidy
go build -o forklift
```

## SSH Setup (Recommended)

For the best development experience, set up SSH keys with GitHub:

1. Generate SSH key: `ssh-keygen -t ed25519 -C "your_email@example.com"`
2. Add to SSH agent: `ssh-add ~/.ssh/id_ed25519`
3. Add public key to GitHub: Copy `~/.ssh/id_ed25519.pub` to GitHub Settings > SSH Keys
4. Test connection: `ssh -T git@github.com`

## Usage

### Development Setup (SSH - Recommended)

```bash
# Harvest your own repos for development
./forklift yourusername

# Harvest specific dev tools to ~/code
./forklift -d ~/code microsoft

# Quick dev setup - auto-select development repos
./forklift yourusername
# Then type "dev" when prompted
```

### Read-only Setup (HTTPS)

```bash
# Use HTTPS for read-only access (no SSH required)
./forklift --https torvalds

# With GitHub token for private repos
./forklift --https -t YOUR_TOKEN yourusername
```

### Options

- `-t, --token`: GitHub personal access token
- `-d, --dest`: Destination directory  
- `-r, --recursive`: Recursive cloning with submodules (default: true)
- `--https`: Use HTTPS instead of SSH
- `-l, --language`: Filter by programming language

### Selection Options

When prompted for repository selection:
- `1,3,5` - Select specific repositories
- `1-5` - Select range of repositories  
- `all` - Harvest all repositories
- `dev` - Auto-select development repositories (skips docs, websites)
- `python` - Filter by language (e.g., python, go, javascript)
- Enter - Finish selection

## Perfect For

- **New machine setup**: Quickly clone all your development repos
- **Team onboarding**: Set up entire project ecosystems  
- **Open source exploration**: Harvest interesting repos for study
- **Backup workflows**: Local copies of important repositories

## Smart Filtering

The `dev` selection automatically filters out:
- Documentation repositories
- Websites and blogs
- "Awesome" lists
- Non-development projects

Perfect for setting up a clean development environment!

## Examples

```bash
# New machine setup - get all your development repos
./forklift -d ~/projects yourusername

# Explore a company's open source projects
./forklift microsoft
# Type "dev" to focus on actual development tools

# Filter by language from the command line
./forklift -l Go kubernetes

# Quick clone with HTTPS (no SSH setup needed)
./forklift --https -d ~/temp-code torvalds

# Harvest private repos with token
./forklift -t ghp_xxxxxxxxxxxx yourcompany
```

## Philosophy

Forklift treats your local development environment as a garden - helping you discover, select, and cultivate the repositories that matter to your work. With SSH-first approach, your harvested repos are immediately ready for contributions and development.