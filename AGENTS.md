# AGENTS.md - Coding Guidelines for Bootconf

## Build, Test, and Lint Commands

### Building
```bash
make                  # Build bootconf binary
make clean           # Remove build artifacts
go build -o build/bin/bootconf cmd/bootconf/main.go  # Build directly
```

### Testing
```bash
make test            # Run all tests with verbose output: go test -v -race ./...
go test -v ./...     # Run all tests directly
go test -v -run TestConfig_Load ./internal/config/   # Run single test
```

## Code Style Guidelines

### Import Organization

- Standard library imports first (alphabetically sorted)
- Third-party imports second (alphabetically sorted)
- Internal imports last (alphabetically sorted)
- Empty line between groups
- No comments in import blocks

Example:
```go
import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"

    "github.com/offline-lab/bootconf/internal/config"
)
```

### Formatting
- Use `go fmt` conventions (tabs for indentation, no trailing whitespace)
- Struct fields: PascalCase
- Exported functions/types: PascalCase
- Unexported functions/variables: camelCase
- Constants: PascalCase with descriptive prefixes
- Package names: lowercase, single word

### Error Handling
- Always return errors, never panic in library code
- Use `fmt.Errorf` with `%w` verb for error wrapping
- Check errors immediately after operations
- Functions return `(result, error)` tuples
- Example: `return nil, fmt.Errorf("failed to read config: %w", err)`

### Naming Conventions
- Interfaces: Simple nouns (e.g., `Renderer`, `Applier`)
- Constructors: `NewTypeName` (e.g., `NewConfig`, `NewRenderer`)
- Booleans: Prefix with `Is`, `Has`, `Can` (e.g., `isValid`)
- Files: lowercase, single word or underscore-separated

### Testing
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Use `t.Fatal()` for test-ending failures
- Use `t.Errorf()` for non-fatal assertions
- Run single test: `go test -v -run TestName ./path/`

### Comments
- Exported types/functions should have comments explaining "why"
- No inline comments for obvious code
- No comments on unexported code unless explaining non-obvious logic

## Project Structure
```
cmd/bootconf/           - CLI entry point
  main.go               - Thin main calling commands.Execute()
  commands/root.go       - Cobra root command and subcommands
internal/               - Private packages
  version/version.go     - Build version variables (injected via LDFLAGS)
  config/                - YAML config loading and validation
```

## File Formatting
```bash
gofmt -s -w .    # Format all Go files (simplify code)
gofmt -d .       # Show formatting diff without making changes
```

## Key Dependencies
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML configuration parsing

## Important Notes

- Go 1.24.0
- Target platform: arm64 Linux (dev on arm64 macOS)
- Version info injected at build time via LDFLAGS into `internal/version`
- Use `go fmt` and run `make test` before committing
