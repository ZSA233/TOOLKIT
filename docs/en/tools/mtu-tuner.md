# mtu-tuner

`mtu-tuner` is an independently maintained network MTU tuning tool that can be moved out of the monorepo. Its source lives in [services/mtu_tuner](../../../services/mtu_tuner).

## Purpose

`mtu-tuner` helps mitigate MTU blackhole issues by automatically testing for a more suitable MTU. It provides both CLI and desktop GUI entry points. The GUI uses Wails v3 native bindings and does not start a standalone HTTP service.

## Source Layout

- `cmd/cli`: command-line entry point
- `cmd/gui`: desktop GUI entry point
- `cmd/gui/frontend`: React + TypeScript + Vite frontend
- `internal`: project-private business logic, domain code, and infrastructure
- `internal/views`: generated API routes, provider runtime, and Wails transport
- `scripts/blueprint`: source of truth for API blueprint definitions
- `release.toml`: source of truth for version, release assets, and target platform matrix

## API Generation

The API definitions and generation configuration are maintained in:

- `scripts/blueprint/**`
- `scripts/api-blueprint.toml`

Do not edit these generated outputs by hand:

- `internal/**/gen_*`
- `cmd/gui/frontend/src/lib/api/**/gen_*`

Common generation commands:

```bash
make mtu-tuner-api-check
make mtu-tuner-api-gen-all
make mtu-tuner-api-gen-golang
make mtu-tuner-api-gen-typescript
make mtu-tuner-api-gen-wails
```

## Development Commands

```bash
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
make mtu-tuner-run CMD=cli
make mtu-tuner-gui-run
```

## Release Notes

- Stable tag: `mtu-tuner/vX.Y.Z`
- RC tag: `mtu-tuner/vX.Y.Z-rc.N`
- Version, release assets, and target platform matrix are defined by [release.toml](../../../services/mtu_tuner/release.toml)

See the repository-level [release process](../release-process.md).
