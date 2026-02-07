# ONNX Embedder Sidecar

Advanced documentation for the ONNX embedding sidecar architecture.

## Overview

Mehrhof's main binary (`mehr`) is statically compiled (CGO_ENABLED=0) for maximum portability. ONNX Runtime requires CGO, so semantic embeddings use a **sidecar pattern**: a separate `mehr-embedder` binary handles ONNX operations.

## Architecture

```
┌─────────────────┐     JSON-RPC/stdio       ┌──────────────────┐
│      mehr       │ ◄──────────────────────► │  mehr-embedder   │
│  (CGO_ENABLED=0)│                          │  (CGO_ENABLED=1) │
│  Static binary  │                          │  ONNX Runtime    │
└─────────────────┘                          └──────────────────┘
        │                                            ▲
        │  Downloads on first ONNX use               │
        └────────────────────────────────────────────┘
```

**Why this design?**
- Main binary stays portable (no runtime dependencies)
- ONNX functionality is opt-in (only downloaded when configured)
- Single embedder process shared across memory and library systems

## Auto-Download Behavior

When `embedding_model: onnx` is configured:

1. **First use**: `mehr` downloads `mehr-embedder` from GitHub releases
2. **Checksum verification**: `.sha256` file validates download integrity
3. **Model download**: Embedder downloads ONNX model (~22MB for MiniLM)
4. **Caching**: Both binaries and models cached in `~/.valksor/mehrhof/`

**Download locations:**
```
~/.valksor/mehrhof/
  ├── bin/
  │   └── mehr-embedder              # Sidecar binary
  └── models/
      └── all-MiniLM-L6-v2/          # ONNX model files
          ├── model.onnx
          └── tokenizer.json
```

## Configuration

Enable ONNX embeddings in `.mehrhof/config.yaml`:

```yaml
memory:
  enabled: true
  vector_db:
    embedding_model: onnx           # Triggers sidecar download
    onnx:
      model: all-MiniLM-L6-v2       # Model name
      # cache_path: ~/.valksor/mehrhof/models/  # Custom location
```

### Available Models

| Model               | Size | Quality | Speed  | Use Case              |
|---------------------|------|---------|--------|-----------------------|
| `all-MiniLM-L6-v2`  | 22MB | Good    | Fast   | General use (default) |
| `all-MiniLM-L12-v2` | 33MB | Better  | Medium | Higher accuracy needs |

## Platform Support

| Platform     | Embedder Binary | Notes                                          |
|--------------|-----------------|------------------------------------------------|
| linux-amd64  | Available       | Native ONNX support                            |
| linux-arm64  | Available       | Native ONNX support                            |
| darwin-arm64 | Available       | Apple Silicon                                  |
| darwin-amd64 | Available       | Intel Macs                                     |
| windows      | WSL only        | Use Linux binary (`amd64`/`arm64`) inside WSL2 |

On unsupported native platforms or architectures, Mehrhof falls back to hash-based embeddings automatically. No error is raised; semantic search uses keyword matching in that mode.

## Integration with Library

When both memory and library are enabled, library uses memory's embedding model for semantic document scoring:

1. Memory system initializes with ONNX embedder
2. Conductor shares embedding model with library
3. Library uses same embedder for relevance scoring
4. Single sidecar process serves both systems

This avoids spawning multiple sidecar processes and ensures consistent scoring behavior.

## Building from Source

Use this when you need a custom local build:

```bash
# Requires ONNX Runtime installed locally
make build-embedder

# Or build directly
CGO_ENABLED=1 go build -o mehr-embedder ./cmd/mehr-embedder
```

**Dependencies:**
- Go 1.25+
- CGO-capable toolchain (gcc/clang)
- ONNX Runtime library (platform-specific)

### Installing ONNX Runtime

**macOS (Homebrew):**
```bash
brew install onnxruntime
```

**Ubuntu/Debian:**
```bash
# Download from ONNX Runtime releases
wget https://github.com/microsoft/onnxruntime/releases/download/v1.17.0/onnxruntime-linux-x64-1.17.0.tgz
tar xzf onnxruntime-linux-x64-1.17.0.tgz
sudo cp -r onnxruntime-linux-x64-1.17.0/lib/* /usr/local/lib/
sudo cp -r onnxruntime-linux-x64-1.17.0/include/* /usr/local/include/
sudo ldconfig
```

## Troubleshooting

### Embedder Download Fails

Check network connectivity to GitHub releases. Manually download:

```bash
# Ensure mehr is installed with the platform-aware installer first
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash

# Get the release tag matching your mehr version
VERSION=$(mehr version --short)
TAG=${VERSION#v}
if [ "$TAG" = "nightly" ]; then TAG=nightly; else TAG="v${TAG}"; fi
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

curl -L -o ~/.valksor/mehrhof/bin/mehr-embedder \
  "https://github.com/valksor/go-mehrhof/releases/download/${TAG}/mehr-embedder-${OS}-${ARCH}"

chmod +x ~/.valksor/mehrhof/bin/mehr-embedder
```

### Model Download Fails

Models download from Hugging Face. If blocked by firewall, download manually:

```bash
mkdir -p ~/.valksor/mehrhof/models/all-MiniLM-L6-v2

# Download model files
curl -L -o ~/.valksor/mehrhof/models/all-MiniLM-L6-v2/model.onnx \
  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx"

curl -L -o ~/.valksor/mehrhof/models/all-MiniLM-L6-v2/tokenizer.json \
  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json"
```

### Embedder Crashes on Startup

Usually caused by missing ONNX Runtime library. Verify installation:

```bash
# macOS
brew list onnxruntime

# Linux - check library path
ldconfig -p | grep onnxruntime
```

If missing, install ONNX Runtime (see "Installing ONNX Runtime" above).

### Switching Embedding Models

When switching between `default` (hash) and `onnx` embeddings, existing vectors become incompatible. Clear memory after changing models:

```bash
mehr memory clear
```

This removes all stored embeddings. They will be regenerated on next task completion.

## Performance Considerations

| Operation          | Hash Embedding   | ONNX Embedding      |
|--------------------|------------------|---------------------|
| First use          | Instant          | ~30s (download)     |
| Per-text embedding | <1ms             | ~10ms               |
| Memory usage       | Minimal          | ~200MB              |
| Search quality     | Exact match only | Semantic similarity |

ONNX embeddings provide better search quality at the cost of initial download time and memory usage. For most projects, the default hash-based embedding is sufficient.

## See Also

- [CLI: memory](/cli/memory.md) - Memory commands and embedding model configuration
- [Advanced: Semantic Memory](/advanced/semantic-memory.md) - Memory architecture details
- [CLI: library](/cli/library.md) - Library auto-include mechanism
- [Configuration](/configuration/index.md) - Full config reference
