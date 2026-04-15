# Building TitleFlow

TitleFlow is written in Go and builds to a single stateless binary. No external dependencies are required at runtime.

## Prerequisites

- **Go 1.26+** (download from [golang.org](https://golang.org))
- No other system dependencies required

## Building

### Current Platform
```bash
go build -o titleflow
```

On Windows, this creates `titleflow.exe`. On Linux/macOS, it creates `titleflow`.

### Cross-Platform Builds

TitleFlow is built for three platforms:

```bash
# Linux x86-64
GOOS=linux GOARCH=amd64 go build -o titleflow-linux-amd64

# macOS x86-64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o titleflow-macos-amd64

# macOS ARM64 (Apple Silicon / M1+)
GOOS=darwin GOARCH=arm64 go build -o titleflow-macos-arm64

# Windows x86-64
GOOS=windows GOARCH=amd64 go build -o titleflow.exe
```

All binaries are available in `build/`:
- `titleflow-linux-amd64` — 12 MB
- `titleflow-macos-amd64` — 13 MB
- `titleflow-macos-arm64` — 12 MB

## Running

On any platform, run the binary directly:

```bash
./titleflow           # Linux/macOS
titleflow.exe         # Windows
```

The application is self-contained. It will:
1. Scan the `packs/` directory (must exist in the working directory or current directory when invoked)
2. Start three HTTP servers (ports 8080 runtime, 8081 editor, 8082 dock)
3. Output a startup message with connection URLs

## Distribution

Pack everything needed to distribute TitleFlow:

```
titleflow/
├── titleflow           # (or titleflow.exe on Windows)
├── packs/
│   ├── news-pack/
│   ├── sports-pack/
│   └── ... other packs ...
├── README.md
└── PACK_SPEC.md        # Guide for pack authors
```

The single binary plus the `packs/` folder is all that's needed.

## Development

### Live Editing

During development, you can edit element files directly and reload:

1. Start TitleFlow: `go run main.go`
2. Open the editor at `http://localhost:8081`
3. Edit element HTML/CSS/JS in the three-panel editor
4. Click **Save** to write files and update runtime

Changes are reflected immediately in the dock and runtime pages.

### Building for Distribution

```bash
# Test your current platform first
go build -o titleflow-test

# Run tests
go test ./...

# Build all platforms
./build-all.sh    # see Build.sh below
```

### Build Script (Optional)

Create `build-all.sh`:

```bash
#!/bin/bash
mkdir -p build
GOOS=linux GOARCH=amd64 go build -o build/titleflow-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o build/titleflow-macos-amd64
GOOS=darwin GOARCH=arm64 go build -o build/titleflow-macos-arm64
GOOS=windows GOARCH=amd64 go build -o build/titleflow.exe
echo "✓ Built all platforms in build/"
ls -lh build/
```

Then: `chmod +x build-all.sh && ./build-all.sh`
