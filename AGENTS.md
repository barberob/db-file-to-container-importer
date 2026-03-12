# dbimport - CLI Tool for Database Imports

## Overview
dbimport is an interactive CLI tool written in Go that simplifies importing SQL dumps into Docker database containers. It provides a user-friendly TUI (Terminal User Interface) for browsing files (local or S3), selecting Docker containers, and managing database credentials.

## Language & Runtime
- **Language:** Go (Golang)
- **No specific Go version requirement** - Uses standard library + external dependencies
- **Target platforms:** Linux, macOS, Windows (via WSL)

## Project Structure

```
.
├── main.go        # Entry point, orchestration, import flow with retry logic
├── docker.go      # Docker container operations (list, detect DB type, extract env vars)
├── config.go      # Configuration persistence (last directory, per-container credentials)
├── s3.go          # S3 client setup, file browser, download functionality
├── files.go       # File utilities
├── version.go     # Version info and automatic update checker
├── go.mod         # Go module dependencies
└── README.md      # User documentation
```

## Key Dependencies
- `github.com/charmbracelet/huh` - Interactive forms, selects, inputs
- `github.com/charmbracelet/huh/spinner` - Loading spinners
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/aws/aws-sdk-go-v2` - S3 operations (listing, downloading)
- `github.com/hashicorp/go-version` - Semantic version comparison

## Core Features

### 1. Source Selection
- **Local files:** Interactive file browser with directory navigation
  - Filters for `.sql`, `.sql.gz`, `.dump` files
  - Sorted by modification time (most recent first)
  - Remembers last accessed directory
- **S3 buckets:** Browse and download from S3-compatible storage
  - Supports custom endpoints (Scaleway, AWS, MinIO, etc.)
  - Remembers S3 configuration

### 2. Docker Integration
- **Auto-detects database containers** by image name (postgres, mysql, mariadb)
- **Auto-detects database type** (PostgreSQL vs MySQL)
- **Extracts credentials from container environment variables:**
  - PostgreSQL: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`
  - MySQL/MariaDB: `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`
  - Falls back to: `PGDATABASE`, `PGUSER`, `PGPASSWORD`
  - Falls back to root password if user password not set

### 3. Credential Management
- **Priority order:** Saved config > Container env vars > Default placeholders
- **Persistence:** Saves credentials per container in `~/.config/dbimport/containers/{container}.json`
- **Security:** Credential files have 0600 permissions (owner-only)
- **Smart defaults:** PostgreSQL defaults to `app`/`app`, MySQL to `root`/`password`

### 4. Import Process
- **Pre-import:** Optionally empty database (drop & recreate)
- **File handling:** 
 - Copies file to container
  - Auto-extracts `.gz` files
- **Database import:** Uses native tools (`psql` for PostgreSQL, `mysql` for MySQL)
- **Cleanup:** Removes temporary files from container
- **Post-import:** Saves successful credentials for next time

### 5. Error Handling & Retry
- **Retry mechanism:** If import fails (often wrong credentials):
  - Shows error message
  - Offers to "Modify credentials and retry" or "Cancel"
  - Re-opens credential form with current values pre-filled
  - Allows editing without re-entering everything
- **Graceful errors:** Returns errors instead of `log.Fatal()` during import attempts

### 6. Auto-Updates
- Checks for new releases on startup (GitHub releases)
- Non-blocking, silent on error
- Uses `github.com/hashicorp/go-version` for comparison

## Configuration Files

### Location
`~/.config/dbimport/`

### Files
- `lastdir` - Last accessed directory (for local file browsing)
- `s3config` - S3 endpoint, credentials, bucket, region
- `containers/{container_name}.json` - Per-container database credentials

## Supported Database Types

### PostgreSQL
- Uses `psql`, `dropdb`, `createdb` commands
- Environment variable: `PGPASSWORD`

### MySQL / MariaDB
- Uses `mysql` command
- Supports both `MYSQL_*` and `MARIADB_*` env vars

## Import Flow
1. Select source (local/S3)
2. Browse and select SQL file
3. Select Docker container
4. Auto-detect or select database type
5. Load credentials (saved → env vars → defaults)
6. Show credential form (pre-filled, editable)
7. Ask if should empty database
8. Import with spinner
9. On failure: offer retry with credential edit
10. On success: save credentials

## Code Patterns

### Forms (huh library)
```go
form := huh.NewForm(
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Title").
            Options(options...).
            Value(&variable).
            Height(10).
            Filtering(true),
    ),
)
form.Run()
```

### Docker commands
```go
// Execute command with error return (for retry mechanism)
func runWithError(name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### Credential loading priority
1. Load saved config if exists
2. Else extract from container env vars
3. Else use defaults

## Important Notes for AI Agents

### Git Workflow
- **NEVER commit changes unless the user explicitly asks you to**
- **It is VERY IMPORTANT to only commit when explicitly asked**
- **Always wait for explicit confirmation before running git commit, git push, or other destructive git operations**

### Code Style
- Use constructor property promotion where possible
- Keep controllers/agents simple, business logic in services
- Use specific return types, avoid `any`
- Prefer early returns over nested ifs
- Use `readonly` for injected dependencies

### Testing
- No test files currently exist in the project
- Build verification: `go build -o dbimport .`
- Release: Uses GoReleaser (`.goreleaser.yml`)

### Adding Features
- The tool is designed to be simple and focused
- Use the `huh` library for all interactive elements
- Maintain French language for user-facing text (already established pattern)
- Keep error messages informative and actionable

### Environment
- Works exclusively with Docker containers
- Requires Docker daemon to be running
- No database server installation needed locally
