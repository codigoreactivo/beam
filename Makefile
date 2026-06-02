.PHONY: build gui gui-dev gui-app install clean

BIN := beam

# ── CLI/TUI/MCP binary (no GUI) ───────────────────────────────────────────────
build:
	go build -o $(BIN) .

# ── GUI: build frontend then compile with wails tag ──────────────────────────
gui:
	@echo "→ building frontend…"
	cd gui/frontend && npm install --silent && npm run build
	@echo "→ compiling beam with GUI…"
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags "wails production" -o $(BIN) .
	@echo "✓ done — run: ./$(BIN) gui"

# ── GUI: native .app bundle (requires Wails CLI) ─────────────────────────────
gui-app:
	@command -v wails >/dev/null 2>&1 || { echo "install Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1; }
	@[ -e frontend ] || ln -s gui/frontend frontend
	wails build -tags wails
	@echo "✓ app → build/bin/Beam.app"

# ── GUI live dev mode (requires Wails CLI) ───────────────────────────────────
gui-dev:
	@command -v wails >/dev/null 2>&1 || { echo "install Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1; }
	@[ -e frontend ] || ln -s gui/frontend frontend
	wails dev -tags wails

install: build
	go install .

clean:
	rm -f $(BIN)
	rm -rf gui/frontend/dist build/
