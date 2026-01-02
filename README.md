# Chunk - Modpack Server Toolkit

> **Part of the [Chunk ecosystem](https://github.com/usechunk/chunk-docs) - see [Documentation](https://github.com/usechunk/chunk-docs)**

A universal CLI for deploying modded Minecraft servers in seconds, not hours.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](go.mod)

```bash
chunk install atm9
# That's it. Your server is ready to start.
```

## 🎯 Why Chunk?

Setting up modded Minecraft servers is painful—hours of downloading mods, version mismatches, config headaches. **Chunk handles all of it.** One command to install, one to upgrade, with automatic Java detection and mod filtering.

## ✨ Features

- **Multiple Sources** - ChunkHub, GitHub, Modrinth, local files
- **Universal Loaders** - Forge, Fabric, NeoForge auto-configured
- **Smart Java** - Auto-detection and version validation
- **Data Preservation** - Backups, world preservation, rollback support
- **Version Management** - Compare changes with `chunk diff`
- **Validation** - Smoke tests and auto-fix for common issues

## 📦 Installation

**macOS/Linux:**
```bash
curl -sSL https://chunkhub.io/install.sh | bash
```

**Windows:**
```powershell
irm https://chunkhub.io/install.ps1 | iex
```

**Using Go:**
```bash
go install github.com/alexinslc/chunk/cmd/chunk@latest
```

## 🚀 Quick Start

```bash
# Install from ChunkHub registry
chunk install atm9

# Install from GitHub
chunk install alexinslc/my-cool-modpack

# Install from Modrinth (.mrpack)
chunk install ./modpack.mrpack

# Install to specific directory
chunk install atm9 --dir /opt/minecraft

# Search for modpacks
chunk search "all the mods"

# Upgrade existing server
chunk upgrade atm9

# Preview changes before upgrading
chunk diff atm9

# Uninstall a modpack
chunk uninstall atm9

# Uninstall keeping world data
chunk uninstall atm9 --keep-worlds
```

## 📖 Documentation

- **[Full Documentation](https://github.com/usechunk/chunk-docs)** - Comprehensive guides
- **[CLI Usage Guide](docs/CLI_USAGE.md)** - Complete command reference
- **[Installed Manifest Spec](docs/INSTALLED_MANIFEST.md)** - Server manifest format
- **[Architecture](https://github.com/usechunk/chunk-docs/blob/main/ARCHITECTURE.md)** - Platform design
- **[API Reference](https://github.com/usechunk/chunk-docs/blob/main/API.md)** - ChunkHub API

## 🔗 Related Projects

- [chunk-docs](https://github.com/usechunk/chunk-docs) - Central documentation
- [chunk-api](https://github.com/usechunk/chunk-api) - Registry backend
- [chunk-app](https://github.com/usechunk/chunk-app) - Web interface

## 🎨 For Modpack Creators

### Creating Recipes

The recommended way to distribute modpacks is through recipe benches. Create a recipe JSON file:

```json
{
  "slug": "my-modpack",
  "name": "My Awesome Modpack",
  "version": "1.0.0",
  "mc_version": "1.20.1",
  "loader": "forge",
  "loader_version": "47.2.0",
  "download_url": "https://example.com/modpack.zip",
  "sha256": "abc123..."
}
```

Submit your recipe to [usechunk/recipes](https://github.com/usechunk/recipes) to make it available to all Chunk users.

### Alternative: GitHub-Hosted Modpacks

You can also add an installed manifest template (`.chunk.json`) to your repository:

```json
{
  "schema_version": "1.0.0",
  "name": "My Awesome Modpack",
  "version": "1.0.0",
  "mc_version": "1.20.1",
  "loader": "forge",
  "loader_version": "47.2.0"
}
```

Then users can install with: `chunk install yourusername/your-modpack`

See the **[Installed Manifest specification](docs/INSTALLED_MANIFEST.md)** for the full format.

## 🛠️ Development

```bash
git clone https://github.com/alexinslc/chunk.git
cd chunk
make dev      # Install dependencies
make build    # Build CLI
make test     # Run tests
```

See **[chunk-docs](https://github.com/usechunk/chunk-docs)** for detailed development, deployment, and architecture guides.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make changes and run `make test`
4. Open a Pull Request

See **[Issues](https://github.com/usechunk/chunk-cli/issues)** for contribution ideas.

## 💬 Support

- **[Documentation](https://github.com/usechunk/chunk-docs)**
- **[GitHub Issues](https://github.com/usechunk/chunk-cli/issues)**
- **[Discussions](https://github.com/usechunk/chunk-cli/discussions)**

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

