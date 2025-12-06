# GitHub Copilot Instructions for Chunk CLI

## Project Context
Go CLI tool for installing and managing Minecraft mods/modpacks. Integrates with multiple sources (Modrinth, CurseForge, GitHub, ChunkHub).

## Code Style
- Go 1.25.4+
- Run `go fmt` on all code
- Use descriptive variable names
- Add comments for exported functions
- Keep functions small and focused

## Patterns
- Cobra for CLI framework
- Viper for configuration
- Standard library for HTTP (no external deps unless necessary)
- Structured error handling with wrapped errors
- Progress bars for long operations

## CLI Design
- Commands: install, upgrade, search, build, publish
- Consistent flag naming
- Short and long flags (-d, --dir)
- Colorized output for better UX
- Validate inputs early

## File Structure
- `cmd/` - CLI commands
- `internal/` - Core logic (not exported)
- `docs/` - User documentation

## Testing
- Table-driven tests
- Mock external HTTP calls
- Test CLI commands with cobra testing utils

## Don't
- No panics (return errors)
- No global mutable state
- No external deps for simple tasks
- No CLI output in library code

## How to Build and Test

### Building
```bash
make build              # Build the CLI binary to bin/chunk
go build -o bin/chunk ./cmd/chunk  # Alternative direct build
```

### Testing
```bash
make test               # Run all tests (Go + Python API tests)
go test ./...           # Run only Go tests
go test ./internal/sources -v  # Run specific package tests
go test -run TestInstall  # Run specific test
```

### Running
```bash
./bin/chunk install atm9
make run-cli ARGS="search minecraft"  # Run without building
```

### Linting
```bash
go fmt ./...            # Format all code
go vet ./...            # Run static analysis
```

## Dependencies and Tools

### Required
- **Go 1.25.4+** - Primary language
- **Cobra** - CLI framework (github.com/spf13/cobra)
- **Make** - Build automation

### Development
- Standard Go toolchain (no extra linters required)
- Git for version control

### Optional (API development)
- Python 3.x + uv for API development
- FastAPI (for backend API work)

## Common Tasks

### Adding a New Command
1. Create command file in `cmd/chunk/commands/`
2. Define command using Cobra pattern
3. Add command to `rootCmd` in `main.go`
4. Add tests in `*_test.go` file
5. Update README if user-facing

### Adding a New Source Type
1. Implement source interface in `internal/sources/`
2. Add source type to manager
3. Update `types.go` with new types
4. Add tests for new source
5. Update docs/CLI_USAGE.md

### Working with Modpack Manifests
- See `docs/chunk-json-spec.md` for `.chunk.json` format
- Converter logic is in `internal/converter/`
- Metadata handling in `internal/metadata/`

## Debugging and Troubleshooting

### Common Issues
- **Build fails**: Run `go mod download` to ensure dependencies
- **Test fails**: Some tests are placeholders (marked "not yet implemented")
- **Import errors**: Check module path is `github.com/alexinslc/chunk`

### Debugging Commands
```bash
go run ./cmd/chunk install atm9  # Run with Go for stack traces
./bin/chunk install atm9 --dir /tmp/test  # Test in isolated dir
```

### Test Validation
- Tests use Cobra test utilities for CLI testing
- External HTTP calls should be mocked
- Table-driven tests are preferred

## Environment Setup
- Works on macOS, Linux, and Windows
- Install script available at https://chunkhub.io/install.sh
- No special environment variables required
- Server directory defaults to `./server` but can be overridden with `--dir`

## Repository Structure
```
cmd/chunk/           # CLI entry point and commands
  commands/          # Individual command implementations
internal/            # Private packages (not importable)
  checksum/          # File integrity checking
  config/            # Configuration management
  converter/         # Modpack format conversion
  deps/              # Dependency resolution
  install/           # Installation logic
  java/              # Java detection and management
  metadata/          # Modpack metadata handling
  preserve/          # Data preservation for upgrades
  sources/           # Multiple source integrations (Modrinth, GitHub, etc.)
  telemetry/         # Usage analytics (opt-in)
  ui/                # CLI output formatting
  validation/        # Installation validation
docs/                # User documentation
  API_REFERENCE.md   # ChunkHub API documentation
  CLI_USAGE.md       # Command-line usage guide
  chunk-json-spec.md # Modpack manifest specification
```

## Project-Specific Conventions

### Error Handling
- Always return errors, never panic
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- User-facing errors should be descriptive

### Naming
- Commands: lowercase (install, upgrade, search)
- Packages: lowercase, single word when possible
- Exported functions: Start with verb (GetManifest, InstallMod)

### Output
- Use structured output for machine-readable data
- Colorize user-facing messages for better UX
- Progress bars for long operations
- Warnings use ⚠️, success uses ✓

### Configuration
- Prefer flags over environment variables
- Use Viper for configuration management
- Keep sensible defaults

## Resources
- [Full Documentation](https://github.com/usechunk/chunk-docs)
- [API Reference](docs/API_REFERENCE.md)
- [CLI Usage Guide](docs/CLI_USAGE.md)
- [.chunk.json Spec](docs/chunk-json-spec.md)
