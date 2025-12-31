# Chunk CLI Documentation

## Overview

Chunk is a universal CLI tool for deploying and managing modded Minecraft servers. It makes deploying modpack servers as simple as running a single command.

## Quick Start

```bash
# Search for modpacks in local recipe benches
chunk search "all the mods"

# Install a modpack from local recipes
chunk install atm9

# Install from a specific bench
chunk install usechunk/recipes::atm9

# Install from GitHub (shorthand)
chunk install alexinslc/my-cool-mod

# Install from local file
chunk install ./mymodpack.mrpack

# Upgrade an existing installation
chunk upgrade vault-hunters

# Compare versions
chunk diff ./old-server ./new-modpack

# Manage recipe benches
chunk bench add usechunk/recipes
chunk bench list
chunk bench update usechunk/recipes

# Create a new recipe interactively
chunk recipe create
```

## Installation

### macOS/Linux
```bash
curl -sSL https://chunkhub.io/install.sh | bash
```

### Windows
```powershell
irm https://chunkhub.io/install.ps1 | iex
```

### From Source
```bash
git clone https://github.com/alexinslc/chunk
cd chunk
make install
```

## Commands

### `chunk install <modpack>`

Install a modpack server to the current directory or specified location.

**Arguments:**
- `modpack` - Modpack identifier:
  - Recipe name: `atm9` (searches all installed benches)
  - Explicit bench: `usechunk/recipes::atm9`
  - GitHub repo: `alexinslc/my-modpack`
  - Modrinth: `modrinth:modpack-slug`
  - Local file: `./modpack.mrpack`

**Flags:**
- `--dir <path>` - Installation directory (default: ./server)
- `--skip-verify` - Skip checksum verification (not recommended)

**Examples:**
```bash
# Install from local recipe bench
chunk install atm9 --dir /opt/minecraft

# Install from specific bench
chunk install usechunk/recipes::atm9

# Install from GitHub
chunk install alexinslc/my-modpack

# Install from local file
chunk install ./modpack.mrpack

# Install without checksum verification
chunk install atm9 --skip-verify
```

**Recipe Installation:**

When installing from recipes, chunk will:
1. Search all installed benches for the recipe
2. Download the modpack from the recipe's download URL
3. Verify the SHA-256 checksum
4. Extract files to the installation directory
5. Create `.chunk-recipe.json` to track the source
6. Install the mod loader and generate start scripts

### `chunk search [query]`

Search for modpacks in local recipe benches.

**Flags:**
- `--bench <name>` - Limit search to a specific bench

**Examples:**
```bash
# Search all benches
chunk search "all the mods"

# Search specific bench
chunk search atm --bench usechunk/recipes
```

### `chunk upgrade [modpack]`

Upgrade an existing modpack server installation to the latest version while preserving world data and configurations.

**Arguments:**
- `modpack` - (Optional) Modpack identifier to upgrade to. If omitted, attempts to detect from installed.json

**Flags:**
- `-d, --dir <path>` - Server directory to upgrade (default: ./server)
- `--dry-run` - Preview changes without upgrading
- `--skip-backup` - Skip backup creation (not recommended)
- `--verify` - Verify checksums of downloaded files (default: true)

**Examples:**
```bash
# Upgrade from tracked installation
chunk upgrade --dir /opt/minecraft

# Upgrade specific modpack
chunk upgrade atm9

# Preview upgrade changes
chunk upgrade atm9 --dry-run

# Upgrade without backup (not recommended)
chunk upgrade atm9 --skip-backup

# Upgrade with custom directory
chunk upgrade atm9 --dir /opt/minecraft
```

**Upgrade Process:**

The upgrade command performs the following steps:

1. **Version Detection:**
   - Reads current version from `.chunk-recipe.json` in server directory
   - Queries benches for the latest version of the modpack
   - Displays version comparison and changes

2. **Backup Creation:**
   - Creates backup in `.chunk-backup` directory within server
   - Backs up critical files:
     - `world/`, `world_nether/`, `world_the_end/` - World data
     - `server.properties` - Server configuration
     - `whitelist.json`, `ops.json`, `banned-players.json`, `banned-ips.json` - Player data

3. **Installation:**
   - Downloads new modpack version
   - Installs new mods and mod loader
   - Updates configuration files
   - Generates new start scripts

4. **Data Restoration:**
   - Restores world data from backup
   - Restores server configuration files
   - Preserves custom player permissions and bans

5. **Rollback on Failure:**
   - If upgrade fails, automatically restores from backup
   - Returns server to previous working state

**Output Example:**
```
🔄 Chunk Modpack Upgrader

ℹ Current version: 0.3.1
ℹ Checking for updates...
ℹ Available version: 0.3.2

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Upgrade Summary

   Modpack:       All the Mods 9
   Current:       0.3.1
   New:           0.3.2
   Minecraft:     1.20.1
   Loader:        forge 47.2.0
   Server Mods:   342
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💾 Data to preserve:
   • world
   • server.properties
   • whitelist.json
   • ops.json

✓ Backup created: .chunk-backup
ℹ Downloading and installing new version...
✓ Data restored

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Upgrade Complete!

   Modpack:   All the Mods 9
   Version:   0.3.2
   Location:  /opt/minecraft
   Backup:    /opt/minecraft/.chunk-backup

To start the server:
   cd /opt/minecraft
   ./start.sh (Linux/Mac) or start.bat (Windows)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Safety Features:**

- **Automatic Backup:** Creates backup before any changes
- **Dry Run Mode:** Preview changes with `--dry-run` flag
- **Version Comparison:** Shows what will change before upgrading
- **Automatic Rollback:** Restores previous version if upgrade fails
- **Data Preservation:** World and player data are never deleted
- **Checksum Verification:** Validates downloaded files for integrity

**When to Use:**

- Updating to a new modpack version
- Applying bug fixes or mod updates
- Upgrading Minecraft or loader versions
- Migrating between compatible modpack versions

**Warning:**

Major version changes (e.g., Minecraft 1.19 → 1.20) may not be compatible with existing worlds. Always test in a backup world first.

### `chunk uninstall <modpack>`

Uninstall a modpack server installation and optionally preserve world data.

**Arguments:**
- `modpack` - Modpack identifier to uninstall

**Flags:**
- `--dir <path>` - Server directory to uninstall from (default: ./server)
- `--keep-worlds` - Preserve world and player data
- `--force` - Skip confirmation prompts (respects --keep-worlds)

**Examples:**
```bash
# Interactive uninstall with prompt for world preservation
chunk uninstall atm9

# Keep world data without prompt
chunk uninstall atm9 --keep-worlds

# Force uninstall without confirmation prompts
chunk uninstall atm9 --force

# Uninstall from custom directory
chunk uninstall atm9 --dir /opt/minecraft

# Force uninstall, keeping worlds (no prompts, preserves world data)
chunk uninstall atm9 --force --keep-worlds
```

**Behavior:**

By default, the uninstall command will:
1. Prompt you to keep or delete world and player data
2. Show what will be removed and preserved
3. Ask for final confirmation before proceeding
4. Remove modpack files (mods, configs, libraries, scripts)
5. Optionally preserve world data, server properties, and player files
6. Update installation tracking in `~/.chunk/installed.json`

**Removed Files:**
- `mods/` - All mod files
- `config/` - Configuration files
- `libraries/` - Mod loader libraries
- `defaultconfigs/` - Default configurations
- `kubejs/`, `scripts/` - Custom scripts
- `resourcepacks/`, `shaderpacks/` - Resource packs
- Start scripts (`start.sh`, `start.bat`)
- Loader installers and JAR files
- `eula.txt`

**Preserved Files (with --keep-worlds):**
- `world/`, `world_nether/`, `world_the_end/` - World data
- `server.properties` - Server configuration
- `whitelist.json`, `ops.json` - Player permissions
- `banned-players.json`, `banned-ips.json` - Ban lists
- `usercache.json` - Player cache

### `chunk bench`

Manage recipe benches (repositories containing modpack recipes).

**Subcommands:**
- `add <name> [url]` - Add a new bench
- `remove <name>` - Remove a bench
- `list` - List all installed benches
- `update [name]` - Update bench(es) to latest recipes
- `info <name>` - Show detailed bench information

**Examples:**
```bash
# Add the core recipes bench
chunk bench add usechunk/recipes

# Add a custom bench
chunk bench add my-bench https://github.com/user/recipes

# List all benches
chunk bench list

# Update all benches
chunk bench update

# Update specific bench
chunk bench update usechunk/recipes

# Show bench info
chunk bench info usechunk/recipes

# Remove a bench
chunk bench remove my-bench
```

**About Benches:**

Benches are Git repositories containing recipe files in a `Recipes/` directory. Each recipe is a JSON or YAML file that describes:
- Modpack metadata (name, version, description)
- Minecraft and mod loader versions
- Download URL for the modpack archive
- SHA-256 checksum for verification
- System requirements (RAM, disk space, Java version)

The core bench (`usechunk/recipes`) is automatically added on first run unless `CHUNK_NO_AUTO_BENCH=1` is set.

### `chunk diff <old> <new>`

Compare two modpack versions and show differences.

**Arguments:**
- `old` - Current modpack or server directory
- `new` - New modpack to compare against

**Examples:**
```bash
chunk diff ./my-server atm9:latest
```

## Configuration

Chunk uses a `.chunk.json` manifest file for modpack specifications:

```json
{
  "name": "My Modpack",
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
      "version": "15.2.0",
      "url": "https://modrinth.com/mod/jei"
    }
  ]
}
```

## Java Requirements

Chunk automatically detects Java installations and validates compatibility:

| Minecraft Version | Required Java |
|------------------|---------------|
| 1.21+            | Java 21+      |
| 1.20.5+          | Java 21+      |
| 1.20.x           | Java 17+      |
| 1.18-1.19        | Java 17+      |
| 1.16-1.17        | Java 8+       |

If Java is not installed or incompatible, Chunk provides installation instructions.

## Data Preservation

When upgrading servers, Chunk automatically preserves:
- World data (overworld, nether, end)
- Server configuration (server.properties)
- Player data (whitelist, ops, bans)
- Custom configurations

A backup is created before upgrades and can be restored if issues occur.

## Recipe Management

### `chunk recipe create`

Create a new recipe JSON file through an interactive wizard.

**Flags:**
- `--template <recipe>` - Start from an existing recipe (name, slug, or file path)
- `--output <dir>` - Output directory for the recipe file

**Examples:**
```bash
# Create a new recipe interactively
chunk recipe create

# Start from an existing recipe
chunk recipe create --template atm9

# Save to a specific directory
chunk recipe create --output ./my-recipes
```

**Interactive Flow:**

The wizard will prompt you for:
1. **Name** - Modpack name (e.g., "My Custom Modpack")
2. **Slug** - Auto-generated from name (e.g., "my-custom-modpack")
3. **Description** - Brief description of the modpack
4. **Minecraft version** - Target MC version (e.g., "1.20.1")
5. **Loader** - Mod loader type (forge/fabric/neoforge)
6. **Loader version** - Specific loader version (e.g., "47.3.0")
7. **Download URL** - Direct download link to modpack
8. **RAM** - Recommended RAM in GB
9. **Disk space** - Required disk space in GB
10. **License** - License type (MIT/GPL-3.0/ARR)
11. **Homepage** - Optional project homepage
12. **Author** - Optional author name

The command will:
- Download the file to calculate SHA-256 checksum
- Validate all inputs
- Generate a properly formatted JSON recipe file
- Display instructions for submitting to usechunk/recipes

**Security Notes:**
- Only HTTP/HTTPS URLs are allowed
- Files are limited to 2GB to prevent memory exhaustion
- Checksums are automatically calculated for integrity verification

**Submitting Recipes:**

To submit your recipe to the official repository:
1. Fork https://github.com/usechunk/recipes
2. Add your recipe JSON to the `Recipes/` directory
3. Open a pull request
4. Your recipe will be reviewed and made available to all chunk users

## Troubleshooting

### Java Not Found
```bash
# Check Java installation
java -version

# Chunk will guide you through installation if needed
chunk install <modpack>
```

### Permission Denied
```bash
# Make start script executable
chmod +x start.sh

# Or use auto-fix
chunk validate --fix
```

### Mod Compatibility Issues
```bash
# Check differences between versions
chunk diff old-version new-version

# Review breaking changes before upgrading
```

## Support

- Documentation: https://docs.chunkhub.io
- Issues: https://github.com/alexinslc/chunk/issues
- Discord: https://discord.gg/chunk

## License

MIT License - see LICENSE.md for details
