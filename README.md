# Chunk - Modpack Server Toolkit

> **Part of the [Chunk ecosystem](https://github.com/usechunk/chunk-docs) - see [Architecture](https://github.com/usechunk/chunk-docs/blob/main/ARCHITECTURE.md)**

A universal CLI for deploying modded Minecraft servers in seconds, not hours.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](go.mod)
[![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python)](api/pyproject.toml)

Chunk is a Homebrew-style tool that transforms modded Minecraft server deployment from a tedious multi-hour process into a single command. No more manual mod downloads, no more config headaches, no more version mismatches.

```bash
chunk install atm9
# That's it. Your server is ready to start.
```

## 🎯 Why Chunk?

**The Problem:**
Setting up modded Minecraft servers is painful. Hours of downloading mods individually, figuring out which are server-side, dealing with version incompatibilities, configuring memory settings, and repeating this nightmare for every update.

**The Solution:**
Chunk handles all of it. One command to install, one command to upgrade, automatic Java detection, automatic mod filtering, automatic everything.

## ✨ Features

### 🚀 Multiple Installation Sources
- **ChunkHub Registry** - Curated, verified modpacks
- **GitHub Repos** - `chunk install alexinslc/my-modpack`
- **Modrinth** - Direct .mrpack file support
- **Local Files** - Import existing modpacks

### 🎮 Universal Loader Support
Automatically installs and configures:
- Forge
- Fabric
- NeoForge

### ☕ Smart Java Management
- Detects existing Java installations
- Validates version compatibility
- Guides you through installation if needed
- Supports Java 8, 11, 17, and 21

### 💾 Data Preservation
Never lose your worlds again:
- Automatic backups before upgrades
- Preserves world data, configs, and player data
- Rollback support if something goes wrong
- Merge strategies for config conflicts

### 🔍 Version Management
- Compare versions before upgrading: `chunk diff atm9`
- See exactly what mods changed
- Get warnings about breaking changes
- View known working versions

### 🛡️ Validation & Safety
- Smoke tests after installation
- File structure validation
- Permission checks
- Auto-fix common issues

## 📦 Installation

### Quick Install (macOS/Linux)
```bash
curl -sSL https://chunkhub.io/install.sh | bash
```

### Quick Install (Windows)
```powershell
irm https://chunkhub.io/install.ps1 | iex
```

### From Source
```bash
git clone https://github.com/alexinslc/chunk.git
cd chunk
make install
```

### Using Go
```bash
go install github.com/alexinslc/chunk/cmd/chunk@latest
```

## 🚀 Quick Start

### Install a Modpack

```bash
# From ChunkHub registry
chunk install atm9

# From GitHub (shorthand)
chunk install alexinslc/my-cool-modpack

# From Modrinth
chunk install ./modpack.mrpack

# To specific directory
chunk install atm9 --dir /opt/minecraft
```

### Search for Modpacks

```bash
# Search all modpacks
chunk search

# Search with query
chunk search "all the mods"
```

### Upgrade Existing Server

```bash
# Upgrade to latest version
chunk upgrade atm9

# Preview changes first
chunk diff atm9

# Upgrade specific directory
chunk upgrade atm9 --dir /opt/minecraft
```

### Complete Example

```bash
# 1. Install a modpack
chunk install atm9 --dir ./my-server

# 2. Server is ready - just start it
cd my-server
./start.sh

# 3. Later, upgrade to new version
chunk upgrade atm9 --dir ./my-server

# 4. Check what changed
chunk diff atm9
```

## 📖 Documentation

- **[Architecture](https://github.com/usechunk/chunk-docs/blob/main/ARCHITECTURE.md)** - Understanding the Chunk platform
- **[CLI Usage Guide](docs/CLI_USAGE.md)** - Complete command reference
- **[API Reference](https://github.com/usechunk/chunk-docs/blob/main/API.md)** - ChunkHub API documentation
- **[.chunk.json Spec](docs/chunk-json-spec.md)** - Modpack manifest format

## 🔗 Related Projects

- [chunk-docs](https://github.com/usechunk/chunk-docs) - Central documentation
- [chunk-api](https://github.com/usechunk/chunk-api) - Registry backend
- [chunk-app](https://github.com/usechunk/chunk-app) - Web interface

## 🎨 For Modpack Creators

Make your modpack installable with Chunk by adding a `.chunk.json` to your repository:

```json
{
  "schema_version": "1.0.0",
  "name": "My Awesome Modpack",
  "version": "1.0.0",
  "mc_version": "1.20.1",
  "loader": "forge",
  "loader_version": "47.2.0",
  "java_version": 17,
  "recommended_ram_gb": 8,
  "mods": [
    {
      "id": "jei",
      "name": "Just Enough Items",
      "version": "15.2.0.27",
      "side": "both"
    }
  ]
}
```

Users can then install with:
```bash
chunk install yourusername/your-modpack
```

See the **[.chunk.json specification](docs/chunk-json-spec.md)** for complete documentation.

## 🏗️ Architecture

### Tech Stack

**CLI:**
- **Language:** Go 1.21+
- **CLI Framework:** Cobra
- **HTTP Client:** Native net/http

**Registry API:**
- **Framework:** FastAPI (Python 3.11+)
- **ORM:** SQLAlchemy
- **Database:** PostgreSQL (production), SQLite (dev)
- **Auth:** JWT tokens
- **Package Manager:** uv

## 🛠️ Development

### Prerequisites

- **Go 1.21+** for CLI development
- **Python 3.11+** for API development
- **uv** - Python package manager ([install](https://docs.astral.sh/uv/))
- **Make** - Build automation
- **Docker** (optional) - For API deployment

### Setup

```bash
# Clone repository
git clone https://github.com/alexinslc/chunk.git
cd chunk

# Install all dependencies
make dev

# Build CLI
make build

# Run tests
make test
```

### Available Make Commands

```bash
make help       # Show all available commands
make build      # Build CLI binary to bin/chunk
make test       # Run Go tests
make lint       # Run linters
make run-api    # Start FastAPI server (dev mode)
make run-cli    # Run CLI with ARGS='install atm9'
make clean      # Remove build artifacts
make install    # Install chunk CLI to $GOPATH/bin
make dev        # Full dev environment setup
```

### Running the CLI (Development)

```bash
# Build and run
make build
./bin/chunk install atm9

# Or use make run-cli
make run-cli ARGS="search modpack"
```

### Running the API (Development)

```bash
# Start development server with auto-reload
make run-api

# API will be available at:
# - Main API: http://localhost:8000
# - Swagger Docs: http://localhost:8000/docs
# - ReDoc: http://localhost:8000/redoc
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/sources/...

# Run with coverage
go test -cover ./...
```

## 🚢 Deployment

### CLI Distribution

Build for multiple platforms:

```bash
# Build for current platform
make build

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bin/chunk-linux ./cmd/chunk

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o bin/chunk-macos ./cmd/chunk

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o bin/chunk.exe ./cmd/chunk
```

### API Deployment (Docker)

```bash
cd api

# Development
docker-compose up -d

# Production
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Variables

Create `api/.env`:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/chunkhub
SECRET_KEY=your-secret-key-here
ACCESS_TOKEN_EXPIRE_MINUTES=30
UPLOAD_DIR=./uploads
MAX_FILE_SIZE=524288000
```

## 📋 Roadmap

### ✅ Completed (v1.0)

- [x] Project structure and development environment
- [x] Core CLI framework with Cobra
- [x] Modpack source integrations (ChunkHub, GitHub, Modrinth, local)
- [x] Conversion engine (Forge/Fabric/NeoForge)
- [x] Registry API backend (FastAPI + PostgreSQL)
- [x] Java detection and validation
- [x] Data preservation and upgrade system
- [x] Validation, smoke testing, and error handling
- [x] Complete documentation

### 🚀 Planned (v1.1+)

- [ ] CurseForge modpack support
- [ ] Fleet mode (manage multiple servers)
- [ ] Web UI for ChunkHub registry
- [ ] Auto-update checking for CLI
- [ ] Plugin system for custom sources
- [ ] Pterodactyl panel integration
- [ ] Windows GUI wrapper
- [ ] Automated mod compatibility checking

See [tasks/tasks-prd-modpack-server-toolkit.md](tasks/tasks-prd-modpack-server-toolkit.md) for detailed progress.

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** and add tests
4. **Run tests**: `make test`
5. **Commit**: `git commit -m 'Add amazing feature'`
6. **Push**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**

### Contribution Ideas

- Add support for new modpack sources
- Improve error messages and UX
- Write tests
- Improve documentation
- Report bugs
- Suggest features

## 💬 Support & Community

- **Documentation:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/alexinslc/chunk/issues)
- **Discussions:** [GitHub Discussions](https://github.com/alexinslc/chunk/discussions)
- **Discord:** [Join our Discord](https://discord.gg/chunk) (coming soon)

## 🙏 Acknowledgments

- Inspired by Homebrew's simplicity
- Built for the Minecraft modding community
- Thanks to all modpack creators who make this ecosystem amazing

## 📊 Project Status

**Current Version:** v1.0.0 (MVP Complete)

- ✅ Core functionality implemented
- ✅ Multiple source integrations working
- ✅ Data preservation system complete
- ✅ Comprehensive documentation
- 🚧 Community testing phase
- 🚧 Building ChunkHub modpack registry

