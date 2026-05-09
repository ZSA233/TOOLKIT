# mtu-tuner

`services/mtu_tuner` is the primary implementation of `mtu-tuner`.
It keeps MTU-specific logic local while reusing a small shared desktop foundation from `libs/appkit` and `libs/utils`:

- backend: pure project-owned Go code under `internal/`
- cli: `cmd/cli`
- gui: `cmd/gui`
- frontend: tool-local React + TypeScript + Vite app under `cmd/gui/frontend`
- contracts: `scripts/blueprint/**` + `scripts/api-blueprint.toml`

The GUI uses `Wails v3` local bindings only. It does not start an HTTP server.

## Source Of Truth

- edit API contracts in `scripts/blueprint/**`
- edit generator config in `scripts/api-blueprint.toml`
- do not hand-edit:
  - `internal/**/gen_*`
  - `cmd/gui/frontend/src/lib/api/**/gen_*`

## Common Commands

Generate API artifacts:

```bash
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-check
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-all
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-golang
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-typescript
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-wails
```

The generator config uses the `api-blueprint v1.x` target-based layout:

- `[[go.server]]` writes the shared Go contract/runtime into `internal/views`
- `[[typescript.client]]` writes the shared TypeScript client core into `cmd/gui/frontend/src/lib/api`
- `[[transport.wails]]` writes the Wails v3 overlay; the GUI consumes only this generated facade and does not host a standalone HTTP server

Run tests and builds:

```bash
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
```

Build and package the Windows GUI:

```bash
make mtu-tuner-gui-build-windows WINDOWS_GOARCH=amd64
make mtu-tuner-gui-package-windows WINDOWS_GOARCH=amd64
```

When cross-building from macOS or Linux, pass a Windows-capable C toolchain because the Wails GUI uses CGO. Example with Zig:

```bash
make mtu-tuner-gui-package-windows \
  WINDOWS_GOARCH=amd64 \
  WINDOWS_CC='zig cc -target x86_64-windows-gnu' \
  WINDOWS_CXX='zig c++ -target x86_64-windows-gnu'
```

The packaged output is written to `services/mtu_tuner/build/packages/gui/windows_<arch>/mtu-tuner/`.
The Windows GUI build automatically uses the `windowsgui` subsystem, so the app does not open an extra console window when launched.

Release metadata and public target matrices live in `release.toml`.
Repo-level release flow is documented in `../../docs/release-process.md`.

Run the CLI:

```bash
make mtu-tuner-run CMD=cli
```

Run the GUI:

```bash
make mtu-tuner-gui-run
```
