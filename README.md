# Jira AI Issue Solver

A Go application that automatically processes Jira tickets labeled with "good-for-ai" by using Claude CLI to generate code changes and create pull requests.

## Features

- **Periodic Ticket Scanning**: Automatically scans for tickets with the "good-for-ai" label at configurable intervals
- **AI-Powered Code Generation**: Uses Claude CLI to analyze tickets and generate code changes
- **GitHub Integration**: Creates forks, branches, and pull requests automatically
- **Jira Integration**: Updates ticket status and adds comments with PR links
- **Label Management**: Automatically manages labels to prevent duplicate processing

## How It Works

### Periodic Scanning

The service runs a periodic scanner that:

1. Searches for Jira tickets with the "good-for-ai" label but without the "ai-in-progress" label
2. Processes each ticket by adding the "ai-in-progress" label and updating status to "In Progress"
3. Forks the repository associated with the ticket to the bot's GitHub account
4. Clones the forked repository and creates a new branch
5. Uses Claude CLI to generate code changes based on the ticket description and comments
6. Commits the changes and pushes the branch to the forked repository
7. Creates a Pull Request from the bot's fork to the original repository
8. Adds a comment to the ticket with a link to the PR and updates the ticket status to "In Review"

### Configuration

The scanner interval can be configured using the `SCANNER_INTERVAL_SECONDS` environment variable (default: 300 seconds).

## Installation

### Prerequisites

- Go 1.21 or later
- Claude CLI installed and configured
- GitHub App with appropriate permissions
- Jira API access

### Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd jira-ai-issue-solver
```

2. Install dependencies:
```bash
go mod download
```

3. Create a configuration file (optional) or set environment variables:
```bash
cp config.example.env .env
# Edit .env with your configuration
```

4. Build the application:
```bash
go build -o jira-ai-solver
```

## Configuration

The application uses a YAML configuration file. Copy `config.example.yaml` to `config.yaml` and update the values:

### Configuration File

Create a `config.yaml` file with the following structure:

```yaml
# Server Configuration
server:
  port: 8080

# Jira Configuration
jira:
  base_url: https://your-domain.atlassian.net
  username: your-username
  api_token: your-jira-api-token

# GitHub Configuration
github:
  personal_access_token: your-personal-access-token-here
  bot_username: your-org-ai-bot
  bot_email: ai-bot@your-org.com

# Claude CLI Configuration
claude:
  cli_path: claude-cli
  timeout: 300
  dangerously_skip_permissions: true
  allowed_tools: "Bash Edit"
  disallowed_tools: "Python"

# Scanner Configuration
scanner:
  interval_seconds: 300

# Component to Repository Mapping
component_to_repo:
  frontend: https://github.com/your-org/frontend.git
  backend: https://github.com/your-org/backend.git
  api: https://github.com/your-org/api.git

# Temporary Directory
temp_dir: /tmp/jira-ai-issue-solver

# Jira Configuration
jira_config:
  disable_error_comments: false
```

### Jira Configuration

The `jira_config` section contains additional Jira-specific settings:

- `disable_error_comments`: When set to `true`, prevents the application from adding error comments to Jira tickets when processing fails. Useful for testing or to avoid spamming tickets with error messages.

### Component Mapping

The application uses a component-to-repository mapping to determine which repository to use for each ticket:

```yaml
component_to_repo:
  frontend: https://github.com/your-org/frontend.git
  backend: https://github.com/your-org/backend.git
  api: https://github.com/your-org/api.git
```

The application will:
1. Look at the first component assigned to a Jira ticket
2. Use the component name to find the corresponding repository URL
3. Process the ticket using that repository

### Running the Application

```bash
# Run with default config.yaml
go run main.go

# Run with custom config file
go run main.go -config /path/to/config.yaml

# Build and run
go build -o jira-ai-solver
./jira-ai-solver -config config.yaml
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

## Usage

### Setting Up Jira Tickets

To have a ticket processed by the AI:

1. Add the "good-for-ai" label to the ticket
2. Ensure the ticket's project has the "ai.bot.github.repo" property set to the repository URL
3. The scanner will automatically pick up the ticket and process it

### Ticket Processing Flow

1. **Scanning**: The scanner periodically searches for tickets with "good-for-ai" label
2. **Processing**: When found, the ticket is marked with "ai-in-progress" label
3. **Code Generation**: Claude CLI analyzes the ticket and generates code changes
4. **Pull Request**: A PR is created with the changes
5. **Completion**: The ticket is updated with "ai-pr-created" label and status changed to "In Review"

### Labels Used

- `good-for-ai`: Marks tickets for AI processing
- `ai-in-progress`: Prevents duplicate processing
- `ai-pr-created`: Indicates a PR has been created
- `ai-failed`: Indicates processing failed

### Status Transitions

- **Open** → **In Progress** (when processing starts)
- **In Progress** → **In Review** (when PR is created)
- **In Progress** → **Open** (if processing fails)

## Architecture

The application is built with a clean architecture pattern:

- **Services**: Handle external API interactions (Jira, GitHub, Claude CLI)
- **Handlers**: Process incoming requests and coordinate between services
- **Models**: Define data structures and configuration
- **Scanner**: Periodically searches for and processes tickets

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
