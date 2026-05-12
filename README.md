# CS Foundations Pyramid — Quiz Game

A terminal-based flashcard quiz that drills you on core computer science concepts across 18 categories — from CPU internals and OS primitives to networking, system design, and LLMs.

---

## Running from the command line

### Option 1 — run the pre-built binary (Windows)

```bat
basics.exe
```

### Option 2 — build & run from source (requires [Go 1.21+](https://go.dev/dl/))

```bash
# clone / enter the repo, then:
go run .
```

### Option 3 — build a binary for your platform

```bash
# current platform
go build -o basics .

# Linux (from any OS)
GOOS=linux GOARCH=amd64 go build -o basics-linux .

# macOS (from any OS)
GOOS=darwin GOARCH=arm64 go build -o basics-mac .

# Windows (from any OS)
GOOS=windows GOARCH=amd64 go build -o basics.exe .
```

Then run the produced binary:

```bash
# Unix / macOS
./basics          # or ./basics-linux / ./basics-mac

# Windows (PowerShell / cmd)
.\basics.exe
```

---

## Requirements

| Tool | Version |
|------|---------|
| Go   | 1.21 +  |

No external runtime is needed once the binary is built.
