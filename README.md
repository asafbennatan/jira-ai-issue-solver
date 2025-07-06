# Jira AI Issue Solver

The Jira AI Issue Solver is a service that automatically creates Pull Requests for Jira tickets marked with the "good-for-ai" label, and fixes PRs based on human comments.

## Features

- Automatically processes Jira tickets with the "good-for-ai" label
- Uses a fork+PR workflow for better isolation and security
  - Forks the repository to a dedicated bot account
  - Makes changes in the fork rather than directly in the main repository
  - Creates pull requests from the fork to the original repository
- Uses Claude CLI to generate code changes based on the ticket description
- Updates Jira ticket status and adds comments with links to the PR

## Architecture

The system consists of a single, long-running Go service that receives webhooks from Jira. It interacts with:

- Jira API: To fetch ticket details and update ticket status and labels
- GitHub API: To create PRs and interact with repositories
- Claude CLI: To generate code changes based on ticket descriptions

## Prerequisites

- Go 1.24 or later
- Git command-line interface
- Claude CLI installed and configured
- Access to Jira and GitHub APIs

## Configuration

The application is configured using environment variables:

### Server Configuration
- `SERVER_PORT`: The port to listen on (default: 8080)
- `SERVER_URL`: The publicly accessible URL of the server (e.g., https://your-server.com). Required for automatic Jira webhook registration.

### Jira Configuration
- `JIRA_BASE_URL`: The base URL of your Jira instance (default: https://your-domain.atlassian.net)
- `JIRA_USERNAME`: Your Jira username
- `JIRA_API_TOKEN`: Your Jira API token
- `JIRA_WEBHOOK_SECRET`: The secret for validating Jira webhooks

### GitHub Configuration
- `GITHUB_API_TOKEN`: Your GitHub API token (for main account)
- `GITHUB_USERNAME`: Your GitHub username (main account)
- `GITHUB_EMAIL`: Your GitHub email (main account)
- `GITHUB_BOT_USERNAME`: The username of the GitHub bot account that will own the forks
- `GITHUB_BOT_TOKEN`: The API token for the GitHub bot account
- `GITHUB_BOT_EMAIL`: The email address of the GitHub bot account

### Claude CLI Configuration
- `CLAUDE_CLI_PATH`: The path to the Claude CLI executable (default: claude-cli)
- `CLAUDE_TIMEOUT`: The timeout for Claude CLI operations in seconds (default: 300)
- `CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS`: Whether to use the `--dangerously-skip-permissions` flag (default: false)
- `CLAUDE_ALLOWED_TOOLS`: Comma or space-separated list of tool names to allow (e.g., "Bash(git:*) Edit")
- `CLAUDE_DISALLOWED_TOOLS`: Comma or space-separated list of tool names to deny (e.g., "Bash(git:*) Edit")

### Temporary Directory
- `TEMP_DIR`: The directory to use for temporary files (default: /tmp/jira-ai-issue-solver)

## Setup

### Jira Setup

The application can automatically register and refresh the Jira webhook on startup if the `SERVER_URL` environment variable is set. If you prefer to manually set up the webhook, follow these steps:

1. Create a webhook in Jira with the following configuration:
   - URL: `https://your-server/webhook/jira`
   - Events: `issue_updated`
   - JQL Filter: `labels = "good-for-ai" AND labels != "ai-in-progress"`

2. Add a custom property to your Jira project:
   - Key: `ai.bot.github.repo`
   - Value: The URL of the GitHub repository (e.g., `https://github.com/username/repo.git`)

### GitHub Setup

#### 1. Create a GitHub Bot Account

1. Create a new GitHub account that will serve as your bot account (e.g., `your-org-ai-bot`)
2. This account will own the forks and create pull requests on behalf of the AI

#### 2. Generate GitHub Personal Access Token for Bot Account

1. Log in to the bot's GitHub account
2. Go to **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
3. Click **Generate new token (classic)**
4. Give it a descriptive name like "Jira AI Issue Solver Bot"
5. Set the expiration as needed (or choose "No expiration" for long-term use)
6. Select the following scopes:
   - `repo` (Full control of private repositories)
   - `workflow` (Update GitHub Action workflows)
7. Click **Generate token**
8. **Copy the token immediately** - you won't be able to see it again
9. This token will be used as `GITHUB_BOT_TOKEN`

#### 3. Generate GitHub Personal Access Token for Main Account

1. Log in to your main GitHub account (the one that owns the repositories)
2. Go to **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
3. Click **Generate new token (classic)**
4. Give it a descriptive name like "Jira AI Issue Solver"
5. Set the expiration as needed
6. Select the following scopes:
   - `repo` (Full control of private repositories)
   - `admin:org` (Full control of organizations and teams)
   - `admin:repo_hook` (Full control of repository hooks)
7. Click **Generate token**
8. **Copy the token immediately** - you won't be able to see it again
9. This token will be used as `GITHUB_API_TOKEN`

#### 4. Environment Variables

Set these environment variables for the GitHub integration:

```bash
# Main account token (for webhook management and repository access)
export GITHUB_API_TOKEN="ghp_your_main_account_token_here"

# Bot account credentials
export GITHUB_BOT_USERNAME="your-org-ai-bot"
export GITHUB_BOT_TOKEN="ghp_your_bot_account_token_here"
export GITHUB_BOT_EMAIL="ai-bot@your-org.com"

# Your main account details (for git configuration)
export GITHUB_USERNAME="your-main-username"
export GITHUB_EMAIL="your-email@your-org.com"
```


## Running the Application

```bash
# Set environment variables
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_USERNAME="your-username"
export JIRA_API_TOKEN="your-api-token"
export GITHUB_API_TOKEN="your-github-token"
export GITHUB_USERNAME="your-github-username"
export GITHUB_EMAIL="your-github-email"
export GITHUB_BOT_USERNAME="your-bot-username"
export GITHUB_BOT_TOKEN="your-bot-token"
export GITHUB_BOT_EMAIL="your-bot-email"
export SERVER_URL="https://your-server.com"  # Required for automatic Jira webhook registration

# Claude CLI configuration
export CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=true  # Use with caution
export CLAUDE_ALLOWED_TOOLS="Bash Edit"  # Optional: Comma or space-separated list of allowed tools
export CLAUDE_DISALLOWED_TOOLS="Python"  # Optional: Comma or space-separated list of disallowed tools

# Run the application
go run main.go

# Or build and run
go build -o jira-ai-solver
./jira-ai-solver
```

## Testing

The project includes comprehensive unit tests for all components. Run the tests using:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./handlers
go test ./services
```

## How It Works

### Webhook Registration

On startup, if the `SERVER_URL` environment variable is set, the service will automatically register or refresh the Jira webhook:

1. The service checks for existing webhooks in Jira.
2. If a webhook with the same URL already exists, it is deleted.
3. A new webhook is registered with the appropriate configuration (events, JQL filter, etc.).
4. This ensures that the webhook is always up-to-date and properly configured.

### Processing New Tickets

1. When a ticket is updated with the "good-for-ai" label, Jira sends a webhook to the service.
2. The service adds the "ai-in-progress" label to the ticket and updates its status to "In Progress".
3. The service forks the repository associated with the ticket to the bot's GitHub account.
4. The service clones the forked repository.
5. The service creates a new branch with the format `feature/<JIRA-KEY>-<summary>`.
6. The service generates a prompt for Claude CLI based on the ticket description and comments.
7. Claude CLI generates code changes based on the prompt.
8. The service commits the changes and pushes the branch to the forked repository.
9. The service creates a Pull Request from the bot's fork to the original repository.
10. The service adds a comment to the ticket with a link to the PR and updates the ticket status to "In Review".

## Labels

The service uses the following labels to track the status of tickets:

- `good-for-ai`: Indicates that the ticket should be processed by the AI.
- `ai-in-progress`: Indicates that the AI is currently processing the ticket.
- `ai-pr-created`: Indicates that the AI has created a PR for the ticket.
- `ai-failed`: Indicates that the AI failed to process the ticket.


## Limitations

- The service does not handle merge conflicts.
- The service does not handle complex code changes that require deep understanding of the codebase.
- The service does not handle tickets that require changes to multiple repositories.
- The service does not handle PR review feedback or change requests (this will be handled by a separate complementary project).

## Future Enhancements

A complementary project is planned to handle GitHub webhook events for pull request reviews and change requests. This separate service will:

- Process PR review comments and feedback
- Automatically make requested changes to existing PRs
- Handle iterative improvements based on human feedback
- Maintain the same fork+PR workflow for consistency

This separation allows for better modularity and focused responsibility between initial ticket processing and ongoing PR refinement.
