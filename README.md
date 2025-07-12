# 🔗 LinkPatrol

[![Go Version](https://img.shields.io/badge/Go-1.24.5+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sirprodigle/linkpatrol)](https://goreportcard.com/report/github.com/sirprodigle/linkpatrol)
[![GoDoc](https://godoc.org/github.com/sirprodigle/linkpatrol?status.svg)](https://godoc.org/github.com/sirprodigle/linkpatrol)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/sirprodigle/linkpatrol)

> **A lightning-fast, concurrent link checker for Markdown and HTML files** 🚀

LinkPatrol is a high-performance tool designed to validate links in Markdown and HTML files. It uses concurrent processing to check thousands of links efficiently, making it perfect for documentation projects, static sites, and any content that needs link validation.

## ✨ Features

- 🔍 **Multi-format Support**: Checks links in both Markdown and HTML files
- ⚡ **High Performance**: Concurrent processing with configurable worker pools
- 👀 **Watch Mode**: Real-time monitoring of file changes
- 🎯 **Smart Caching**: Avoids re-checking previously validated links
- 🛡️ **Rate Limiting**: Respectful to servers with configurable request rates
- 📊 **Detailed Reporting**: Clear status indicators and error messages
- 🔧 **Flexible Configuration**: Command-line flags and config file support
- 🎨 **Beautiful Output**: Color-coded results with progress indicators

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/sirprodigle/linkpatrol.git
cd linkpatrol

# Build the binary
go build -o linkpatrol

# Or install directly
go install github.com/sirprodigle/linkpatrol@latest
```

### Basic Usage

```bash
# Check links in current directory
./linkpatrol

# Check links in a specific directory
./linkpatrol -d /path/to/your/docs

# Enable verbose output
./linkpatrol -v

# Watch mode for real-time monitoring
./linkpatrol -w
```

## 📖 Usage Examples

### Simple Link Check
```bash
# Check all Markdown and HTML files in current directory
./linkpatrol
```

### Verbose Output
```bash
# Get detailed information about each link
./linkpatrol -v -d ./docs
```

### Watch Mode
```bash
# Monitor files for changes and re-check links automatically
./linkpatrol -w -d ./website
```

### Custom Configuration
```bash
# Use custom timeout and concurrency settings
./linkpatrol -t 10s -n 8 --tester-concurrency 50 -r 5
```

## ⚙️ Configuration

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-d, --dir` | Root directory to scan | `.` |
| `-w, --watch` | Enable live watch mode | `false` |
| `-v, --verbose` | Enable verbose logging | `false` |
| `-n, --concurrency` | Max concurrent file readers | `CPU cores × 2` |
| `--tester-concurrency` | Max concurrent link testers | `100` |
| `-t, --timeout` | Per-request timeout | `5s` |
| `-r, --rate` | Max requests per second per domain | `10` |
| `--width` | Terminal width override | `auto-detect` |
| `--no-truncate` | Don't truncate URLs or error messages | `false` |

### Environment Variables

All flags can be set via environment variables with the `LINKPATROL_` prefix:

```bash
export LINKPATROL_DIR="./docs"
export LINKPATROL_VERBOSE="true"
export LINKPATROL_TIMEOUT="10s"
```

### Configuration File

Create a `linkpatrol.yaml` file in your project root:

```yaml
dir: "./docs"
watch: false
verbose: true
concurrency: 8
tester-concurrency: 100
timeout: 5s
rate: 10
```

## 📊 Output Format

LinkPatrol provides clear, color-coded output:

```
🔗 LinkPatrol Starting
📁 Scanning Files
   Found 5 markdown files and 3 HTML files
🧪 Testing Links
   https://example.com                    LIVE     ✅      -
   https://broken-link.com               DEAD     ❌      404 Not Found
   https://slow-site.com                 TIMEOUT  ⏰      context deadline exceeded
📊 Results
   Total entries: 150
   Found 5 dead links and 2 timeout links
```

### Status Indicators

- ✅ **LIVE**: Link is accessible and working
- ❌ **DEAD**: Link is broken or inaccessible
- ⏰ **TIMEOUT**: Request timed out
- 🔄 **CACHED**: Link was previously checked (in watch mode)

## 🏗️ Architecture

LinkPatrol uses a multi-layered architecture for optimal performance:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   File Scanner  │────│  Worker Pool    │────│  Link Testers   │
│                 │    │                 │    │                 │
│ • Markdown      │    │ • File Readers  │    │ • HTTP Clients  │
│ • HTML          │    │ • Concurrency   │    │ • Rate Limiting │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │     Cache       │
                       │                 │
                       │ • Results Store │
                       │ • Deduplication │
                       └─────────────────┘
```

## 🔧 Development

### Prerequisites

- Go 1.24.5 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/sirprodigle/linkpatrol.git
cd linkpatrol
# Build the binary
go build -o linkpatrol
```

### Project Structure

```
linkpatrol/
├── internal/              # Internal packages
│   ├── app/              # Main application logic
│   ├── cache/            # Link result caching
│   ├── config/           # Configuration management
│   ├── logger/           # Logging utilities
│   ├── scanner/          # File scanning logic
│   ├── tester/           # Link testing logic
│   ├── walker/           # File parsing (Markdown/HTML)
│   ├── watcher/          # File system watching
│   └── workers/          # Worker pool management
└── main.go              # Application entry point
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run the test suite (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Configuration management with [Viper](https://github.com/spf13/viper)
- File watching with [fsnotify](https://github.com/fsnotify/fsnotify)

## 📈 Performance

LinkPatrol is designed for speed and efficiency:

- **Concurrent Processing**: Multiple workers scan files and test links simultaneously
- **Smart Caching**: Avoids re-checking previously validated links
- **Rate Limiting**: Respectful to servers while maintaining speed
- **Memory Efficient**: Streams file processing to minimize memory usage

### Benchmarks

On a typical documentation project with 1000+ links:
- **LinkPatrol**: ~30-60 seconds
- **Memory usage**: <10MB for most projects

## 🐛 Troubleshooting

### Common Issues

**Slow Performance**
- Increase concurrency: `-n 16 --tester-concurrency 200`
- Reduce rate limiting: `-r 20`
- Use caching (enabled by default)

**Timeout Errors**
- Increase timeout: `-t 10s`
- Check network connectivity
- Verify target servers are responsive

**Memory Issues**
- Reduce concurrency settings
- Process smaller directories
- Monitor with `--memprofile`

### Getting Help

- 📖 [Documentation](https://github.com/sirprodigle/linkpatrol/wiki)
- 🐛 [Issue Tracker](https://github.com/sirprodigle/linkpatrol/issues)
- 💬 [Discussions](https://github.com/sirprodigle/linkpatrol/discussions)

---

<div align="center">

**Made with ❤️ by the LinkPatrol team**

[![GitHub stars](https://img.shields.io/github/stars/sirprodigle/linkpatrol?style=social)](https://github.com/sirprodigle/linkpatrol/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/sirprodigle/linkpatrol?style=social)](https://github.com/sirprodigle/linkpatrol/network/members)
[![GitHub issues](https://img.shields.io/github/issues/sirprodigle/linkpatrol)](https://github.com/sirprodigle/linkpatrol/issues)

</div>
