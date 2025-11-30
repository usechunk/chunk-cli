# Chunk - Modpack Server Toolkit

A universal, open-source CLI and registry system that simplifies modded Minecraft server deployment. Deploy modded Minecraft servers with a single command—no more hours of manual mod installation.

## Overview

Chunk is a Homebrew-style tool that makes deploying modded Minecraft servers as simple as:

```bash
chunk install <modpack>
```

It solves the time investment and technical complexity of modded server setup by:
- Automatically detecting and installing the correct mod loader (Forge/Fabric/NeoForge)
- Downloading all required server-side mods
- Excluding client-only mods
- Generating configurations and start scripts
- Preserving world data and configs during upgrades

## Features

- **Multiple Modpack Sources**: ChunkHub registry, GitHub repositories, Modrinth, or local files
- **Universal Compatibility**: Works on any Linux environment (VMs, cloud, home servers)
- **Smart Java Detection**: Detects existing Java installations and validates versions
- **Data Preservation**: Safely upgrades modpacks while preserving worlds and player data
- **Creator-Friendly**: Simple `.chunk.json` specification for modpack creators
- **Beginner-Friendly**: Clear error messages and optional guided installation

## Quick Start

### Prerequisites

- Go 1.21+ (for CLI development)
- Python 3.11+ with uv (for API development)
- Linux environment (for deployment)

### Installation

```bash
# Clone the repository
git clone https://github.com/alexinslc/chunk.git
cd chunk

# Set up development environment
make dev

# Build the CLI
make build
```

### Usage

```bash
# Search for modpacks
chunk search <query>

# Install a modpack from GitHub
chunk install alexinslc/my-cool-mod

# Install from ChunkHub registry
chunk install atm9

# Install to specific directory
chunk install <modpack> --dir /path/to/server

# Upgrade existing installation
chunk upgrade <modpack>

# Compare versions
chunk diff <modpack>
```

## Development

### Project Structure

```
chunk/
├── cmd/chunk/          # CLI entry point and commands
├── internal/           # Internal packages
│   ├── sources/        # Modpack source integrations
│   ├── converter/      # Modpack-to-server conversion
│   ├── java/           # Java detection and validation
│   ├── upgrade/        # Data preservation and upgrades
│   └── validation/     # Smoke tests and validation
├── api/                # FastAPI registry backend
│   ├── routers/        # API endpoints
│   ├── models/         # Data models
│   └── main.py         # FastAPI app
├── docs/               # Documentation
└── tasks/              # PRD and task lists
```

### Makefile Commands

```bash
make help       # Show available commands
make build      # Build CLI binary
make test       # Run all tests
make run-api    # Start FastAPI server
make run-cli    # Run CLI (use ARGS='...')
make clean      # Clean build artifacts
make install    # Install dependencies
make dev        # Set up development environment
```

### Running the API

```bash
make run-api
# API will be available at http://localhost:8000
# API docs at http://localhost:8000/docs
```

### Running the CLI

```bash
# During development
make run-cli ARGS="search modpack"

# After building
./bin/chunk search modpack
```

## Architecture

- **CLI**: Go-based, using Cobra framework for command routing
- **Registry API**: Python FastAPI backend with SQLAlchemy ORM
- **Database**: SQLite for development, PostgreSQL for production
- **Package Manager**: uv for Python dependencies

## For Modpack Creators

Add a `.chunk.json` file to your repository root:

```json
{
  "name": "My Cool Modpack",
  "mc_version": "1.20.1",
  "loader": "forge",
  "recommended_ram_gb": 8
}
```

See [.chunk.json specification](docs/chunk-json-spec.md) for full documentation.

## Roadmap

- [x] Project structure and setup
- [ ] Core CLI framework
- [ ] Modpack source integrations
- [ ] Conversion engine
- [ ] Registry API backend
- [ ] Java management
- [ ] Data preservation system
- [ ] Validation and testing

See [tasks](tasks/tasks-prd-modpack-server-toolkit.md) for detailed progress.

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

## License

MIT License - see LICENSE file for details.

## Support

- Issues: [GitHub Issues](https://github.com/alexinslc/chunk/issues)
- Documentation: [docs/](docs/)

---

*Chunk: Because Minecraft's world is built from chunks, and so is your server deployment.*
