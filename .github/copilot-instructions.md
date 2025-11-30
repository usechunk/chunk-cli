# GitHub Copilot Instructions for Chunk CLI

## Project Context
Go CLI tool for installing and managing Minecraft mods/modpacks. Integrates with multiple sources (Modrinth, CurseForge, GitHub, ChunkHub).

## Code Style
- Go 1.21+
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
- `pkg/` - Reusable packages (exported)

## Testing
- Table-driven tests
- Mock external HTTP calls
- Test CLI commands with cobra testing utils

## Don't
- No panics (return errors)
- No global mutable state
- No external deps for simple tasks
- No CLI output in library code
