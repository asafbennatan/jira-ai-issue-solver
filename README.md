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
- `GITHUB_APP_ID`: Your GitHub App ID
- `GITHUB_APP_PRIVATE_KEY`: Your GitHub App private key (PEM format)
- `GITHUB_INSTALLATION_ID`: The installation ID of your GitHub App
- `GITHUB_BOT_USERNAME`: The username that will be used for git commits (usually your organization's bot account)
- `GITHUB_BOT_EMAIL`: The email address for git commits

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

#### 1. Create a GitHub App

1. Go to your GitHub organization or user account
2. Navigate to **Settings** → **Developer settings** → **GitHub Apps**
3. Click **New GitHub App**
4. Fill in the app details:
   - **App name**: `jira-ai-issue-solver` (or your preferred name)
   - **Homepage URL**: `https://your-server.com`
   - **Webhook URL**: `https://your-server.com/webhook/github` (optional)
   - **Webhook secret**: Generate a secure secret
5. Set the following permissions:
   - **Repository permissions**:
     - `Contents`: Read and write
     - `Metadata`: Read-only
     - `Pull requests`: Read and write
     - `Workflows`: Read and write
   - **Organization permissions**:
     - `Members`: Read-only (if needed)
6. Click **Create GitHub App**

#### 2. Generate Private Key

1. After creating the app, go to the app's settings page
2. Scroll down to **Private keys** section
3. Click **Generate private key**
4. Download the private key file (`.pem` format)
5. **Copy the private key content** - you'll need this for the `GITHUB_APP_PRIVATE_KEY` environment variable

#### 3. Install the GitHub App

1. Go to your GitHub App's settings page
2. Click **Install App**
3. Choose the repositories or organizations where you want to install the app
4. Note the **Installation ID** from the URL (you'll need this for `GITHUB_INSTALLATION_ID`)

#### 4. Environment Variables

Set these environment variables for GitHub App authentication:

```bash
# GitHub App configuration
export GITHUB_APP_ID="1531890"
export GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"
export GITHUB_INSTALLATION_ID="12345678"

# Git commit configuration
export GITHUB_BOT_USERNAME="your-org-ai-bot"
export GITHUB_BOT_EMAIL="ai-bot@your-org.com"
```


## Running the Application

```bash
# Set environment variables
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_USERNAME="your-username"
export JIRA_API_TOKEN="your-api-token"
export SERVER_URL="https://your-server.com"  # Required for automatic Jira webhook registration

# GitHub App configuration
export GITHUB_APP_ID="1531890"
export GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"
export GITHUB_INSTALLATION_ID="12345678"
export GITHUB_BOT_USERNAME="your-org-ai-bot"
export GITHUB_BOT_EMAIL="ai-bot@your-org.com"

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
