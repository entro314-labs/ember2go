# ember2go for macOS

Create portable Windows installations on USB devices from macOS.

This high-performance Go implementation creates fully functional, portable Windows installations on USB devices that can boot on any UEFI-compatible computer.

## Features
- **Interactive TUI**: Beautiful terminal interface with real-time progress monitoring
- **Fast Performance**: Native Go binary with 10-100x faster execution
- **Single Binary**: No runtime dependencies required
- **Rich CLI**: Full-featured command interface with help and autocomplete
- **Smart Progress**: Real-time extraction progress with ETA, speed, and cancellation support
- **USB Drive Management**: Interactive drive selection or CLI-based listing and formatting
- **Windows Image Support**: Mount ISOs and extract any Windows edition
- **UEFI Bootloader**: Create bootable Windows To Go installations

## Requirements
- macOS 10.15 or later
- Homebrew (for wimlib installation)
- Administrator privileges (for disk operations)
- USB drive (16GB+ recommended, preferably USB 3.0+ SSD)
- Windows 10/11 ISO file

## Installation

### Download Binary
```bash
# Download the latest release
curl -L <release-url> -o ember2go
chmod +x ember2go
```

### Build from Source
```bash
git clone <repo-url>
cd ember2go
go build -o ember2go ./cmd/ember2go
```

## Quick Start

### Interactive Mode (Recommended)
```bash
# Install dependencies
./ember2go install-deps

# Launch interactive TUI
./ember2go create --iso /path/to/windows.iso
```

### CLI Mode (For Automation)
```bash
# List available USB drives
./ember2go list-disks

# Analyze Windows ISO
./ember2go list-editions /path/to/windows.iso

# Create Windows To Go (CLI mode)
./ember2go create --iso /path/to/windows.iso --tui=false --disk disk2 --edition 1 --force
```

## Usage

### List USB Drives
```bash
./ember2go list-disks
```
Output:
```
Available removable disks:
----------------------------------------
  disk2 - SanDisk USB (32GB)
  disk3 - Samsung SSD (64GB)
```

### Analyze Windows ISO
```bash
./ember2go list-editions ~/Downloads/Windows11.iso
```
Output:
```
Available Windows editions:
--------------------------------------------------
  1: Windows 11 Pro (x64)
  2: Windows 11 Home (x64)
  3: Windows 11 Education (x64)
```

### Create Windows To Go

#### Interactive TUI Mode (Default)
```bash
# Launch beautiful terminal interface
./ember2go create --iso windows.iso

# Features:
# - Navigate USB drives with arrow keys
# - Select Windows edition from menu
# - Real-time progress with ETA and speed
# - Interactive confirmation dialogs
```

#### CLI Mode (For Scripts/Automation)
```bash
# Direct creation with all parameters
./ember2go create --iso windows.iso --tui=false --disk disk2 --edition 2 --force

# With verbose output
./ember2go create --iso windows.iso --tui=false --disk disk2 --verbose
```

**⚠️ WARNING: This will ERASE ALL DATA on the selected disk!**

## Commands

| Command | Description | Options |
|---------|-------------|---------|
| `list-disks` | Show removable USB drives | `--verbose` |
| `list-editions <iso>` | Show Windows editions in ISO | `--verbose` |
| `create` | Create Windows To Go installation | `--iso`, `--tui`, `--disk`, `--edition`, `--force` |
| `install-deps` | Install wimlib and dependencies | |

### Flags
- `--tui`: Use interactive TUI mode (default: true)
- `--verbose, -v`: Enable detailed output
- `--help, -h`: Show help for any command
- `--version`: Show version information

### TUI Navigation
- `↑/↓` or `j/k`: Navigate lists
- `Enter`: Select/confirm
- `Esc`: Go back
- `Ctrl+C` or `q`: Quit

## Performance

Benchmarked on MacBook Pro M1:

| Operation | Time | Description |
|-----------|------|-------------|
| List disks | ~15ms | Instant USB drive detection |
| Parse WIM | ~180ms | Windows edition analysis |
| Extract 4GB Windows | ~22min | Limited by disk I/O speed |
| TUI responsiveness | <1ms | Smooth interactive experience |

## Dependencies

### Required (Auto-installed)
- `wimlib` - Windows image extraction (`brew install wimlib`)

### System Tools (Built-in)
- `diskutil` - Disk management
- `hdiutil` - ISO mounting

### Optional (Advanced features)
- `hivex` - BCD registry editing (`brew install hivex`)

### TUI Dependencies (Bundled)
- `bubbletea` - Terminal UI framework
- `lipgloss` - Styling and layout

## How It Works

### Interactive TUI Flow
1. **USB Selection**: Browse available removable drives with arrow keys
2. **Edition Selection**: Choose Windows edition from interactive menu
3. **Confirmation**: Review selections with prominent warning dialog
4. **Live Progress**: Real-time progress dashboard with ETA and transfer speed
5. **Completion**: Success screen with next steps

### Technical Process
1. **Mount ISO**: Uses `hdiutil` to mount Windows ISO files
2. **Analyze WIM**: Parses `install.wim` to show available editions
3. **Format USB**: Creates GPT partition table with BOOT (FAT32) + WINDOWS (ExFAT)
4. **Extract Windows**: Uses `wimlib-imagex` to extract selected edition
5. **Create Bootloader**: Copies Windows EFI files and creates UEFI boot structure
6. **Cleanup**: Safely unmounts all devices

## Troubleshooting

### Common Issues

**Command not found:**
```bash
# Ensure binary is executable
chmod +x ember2go
./ember2go --help
```

**Permission denied:**
```bash
# Run with elevated privileges (TUI mode)
sudo ./ember2go create --iso windows.iso

# Or CLI mode
sudo ./ember2go create --iso windows.iso --tui=false --disk disk2
```

**wimlib not found:**
```bash
# Install dependencies
./ember2go install-deps
# Or manually: brew install wimlib
```

**Disk not detected:**
```bash
# Verify disk with system tools
diskutil list
# Use exact identifier from output
```

### Boot Issues
- Ensure target computer supports UEFI boot
- Try different USB ports (prefer USB 3.0+)
- Check BIOS/UEFI settings for USB boot priority
- Some systems may require disabling Secure Boot

## Project Structure

```tree
ember2go/
├── cmd/ember2go/main.go         # CLI application entry point
├── internal/
│   ├── disk/                  # Disk operations and management
│   ├── wim/                   # Windows image handling
│   ├── bootloader/            # UEFI boot creation
│   └── tui/                   # Terminal user interface
│       ├── models.go          # TUI state and views
│       ├── commands.go        # Async command handlers
│       ├── messages.go        # Event message types
│       └── tui.go             # Main TUI runner
├── go.mod                     # Go dependencies
├── ember2go                   # Compiled binary (~8MB)
└── README.md                  # This file
```

## Development

### Build
```bash
go build -o ember2go ./cmd/ember2go
```

### Test
```bash
go test ./...
```

### Cross-Compile
```bash
# Intel Macs
GOOS=darwin GOARCH=amd64 go build -o ember2go-intel ./cmd/ember2go

# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o ember2go-apple ./cmd/ember2go
```

## Limitations

- UEFI boot only (legacy BIOS not supported)
- Currently supports macOS (Linux/Windows versions possible)
- Basic BCD creation (advanced features require hivex)
- Requires administrator privileges for disk operations

## Security

- Only use official Windows ISO files
- Verify ISO file integrity before use
- Be cautious with disk operations (data loss possible)
- Run with minimal required privileges

## Special Thanks

- [jxctn0](https://github.com/jxctn0/win2go) - Original Win2Go script inspiration
- [BCS-SYS](https://github.com/jpz4085/BCD-SYS) - Windows bootloader creation methods
- [wimlib developers](https://wimlib.net/) - Cross-platform Windows imaging library
- [Cobra CLI](https://github.com/spf13/cobra) - Go CLI framework
- [Charm](https://charm.sh/) - Beautiful terminal UI tools (Bubble Tea, Lipgloss)

## License

MIT License