# Jira AI Issue Solver

The Jira AI Issue Solver is a service that automatically creates Pull Requests for Jira tickets marked with the "good-for-ai" label, and fixes PRs based on human comments.

## Features

- Automatically processes Jira tickets with the "good-for-ai" label
- Uses a fork+PR workflow for better isolation and security
  - Forks the repository to a dedicated bot account
  - Makes changes in the fork rather than directly in the main repository
  - Creates pull requests from the fork to the original repository
- Uses Claude CLI to generate code changes based on the ticket description
- Processes PR review comments and makes additional changes to address feedback
- Updates Jira ticket status and adds comments with links to the PR

## Architecture

The system consists of a single, long-running Go service that receives webhooks from Jira and GitHub. It interacts with:

- Jira API: To fetch ticket details and update ticket status and labels
- GitHub API: To create PRs and interact with repositories
- Claude CLI: To generate code changes based on ticket descriptions and PR feedback

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
- `GITHUB_API_TOKEN`: Your GitHub API token
- `GITHUB_WEBHOOK_SECRET`: The secret for validating GitHub webhooks
- `GITHUB_USERNAME`: Your GitHub username
- `GITHUB_EMAIL`: Your GitHub email
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

1. Create a webhook in your GitHub repository with the following configuration:
   - URL: `https://your-server/webhook/github`
   - Events: `Pull request reviews`
   - Content type: `application/json`
   - Secret: The same value as `GITHUB_WEBHOOK_SECRET`

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
```

## Testing the Application

The application includes a test mode that can be used to verify that the integrations are working correctly.

```bash
# Set test environment variables
export TEST_JIRA=true
export TEST_JIRA_TICKET_KEY="KEY-123"
export TEST_ADD_COMMENT=false
export TEST_REGISTER_WEBHOOK=false
export TEST_SERVER_URL="https://your-server.com"

export TEST_GITHUB=true
export TEST_GITHUB_REPO_URL="https://github.com/username/repo.git"
export TEST_GITHUB_BOT_USERNAME="your-bot-username"
export TEST_GITHUB_BOT_TOKEN="your-bot-token"
export TEST_GITHUB_BOT_EMAIL="your-bot-email"
export TEST_CREATE_BRANCH=false
export TEST_PUSH_CHANGES=false
export TEST_CREATE_PR=false
export TEST_FORK_REPO=false

export TEST_CLAUDE=true
export TEST_REPO_DIR="/path/to/repo"
export TEST_CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=true
export TEST_CLAUDE_ALLOWED_TOOLS="Bash Edit"
export TEST_CLAUDE_DISALLOWED_TOOLS="Python"

# Run the tests
go run main.go -test
```

You can control which tests to run and how they behave using the following environment variables:

### Jira Test Variables
- `TEST_JIRA`: Set to "true" to run Jira integration tests
- `TEST_JIRA_TICKET_KEY`: The key of a Jira ticket to use for testing
- `TEST_ADD_COMMENT`: Set to "true" to test adding a comment to the ticket
- `TEST_REGISTER_WEBHOOK`: Set to "true" to test webhook registration
- `TEST_SERVER_URL`: The URL of the server to use for webhook registration (required if `TEST_REGISTER_WEBHOOK` is "true")

### GitHub Test Variables
- `TEST_GITHUB`: Set to "true" to run GitHub integration tests
- `TEST_GITHUB_REPO_URL`: The URL of a GitHub repository to use for testing
- `TEST_GITHUB_BOT_USERNAME`: The username of the GitHub bot account to use for testing
- `TEST_GITHUB_BOT_TOKEN`: The API token for the GitHub bot account to use for testing
- `TEST_GITHUB_BOT_EMAIL`: The email address of the GitHub bot account to use for testing
- `TEST_FORK_REPO`: Set to "true" to test forking a repository
- `TEST_CREATE_BRANCH`: Set to "true" to test creating a branch
- `TEST_PUSH_CHANGES`: Set to "true" to test pushing changes
- `TEST_CREATE_PR`: Set to "true" to test creating a pull request

### Claude CLI Test Variables
- `TEST_CLAUDE`: Set to "true" to run Claude CLI integration tests
- `TEST_REPO_DIR`: The directory of a repository to use for testing (optional)
- `TEST_CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS`: Set to "true" to use the `--dangerously-skip-permissions` flag
- `TEST_CLAUDE_ALLOWED_TOOLS`: Comma or space-separated list of tool names to allow
- `TEST_CLAUDE_DISALLOWED_TOOLS`: Comma or space-separated list of tool names to deny

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

## Benefits of Fork+PR Workflow

The fork+PR workflow provides several benefits over the branch+PR workflow:

- **Better Isolation**: Changes are made in a separate fork, not directly in the main repository
- **Cleaner Repository**: The main repository doesn't get cluttered with numerous AI-generated branches
- **Improved Security**: The bot only needs push access to its own fork, not the main repository
- **Easier Cleanup**: Forks can be deleted when no longer needed without affecting the main repository

## Limitations

- The service does not handle merge conflicts.
- The service does not handle complex code changes that require deep understanding of the codebase.
- The service does not handle tickets that require changes to multiple repositories.
