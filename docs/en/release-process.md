# Release Process

## Single Source Of Truth

Each releasable tool uses `services/<tool>/release.toml` as its release source of truth. This file defines:

- Public tool ID and version
- Allowed Git tag prefix
- Build targets for CI, stable releases, and RC releases
- Native runner, target platform, and release asset format for each target
- The tool's Go module and GUI frontend lockfile paths, used by workflows to assemble dependencies dynamically

## Tag Convention

- Stable release: `<tool>/vX.Y.Z`
- Release candidate: `<tool>/vX.Y.Z-rc.N`

The version is maintained only in the tool's `release.toml`. Update that file before creating a tag, then use the repository commands to generate and validate the tag. GitHub Actions resolves the target tool from the tag prefix, so adding a new tool only requires its own `release.toml` and build targets.

## Local Commands

```bash
make release-validate TOOL=mtu-tuner WORKFLOW=ci
make release-version-show TOOL=mtu-tuner
make release-version-stable TOOL=mtu-tuner BASE_VERSION=<X.Y.Z> CHECK=1
make release-version-rc TOOL=mtu-tuner BASE_VERSION=<X.Y.Z> RC=<N> CHECK=1
make release-tag-check TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v<X.Y.Z>
make release-preflight TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v<X.Y.Z>
make release-local TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v<X.Y.Z>
make release-matrix TOOL=mtu-tuner WORKFLOW=release
make release-metadata TOOL=mtu-tuner TAG=<tag>
```

These commands delegate to [scripts/releasectl.py](../../scripts/releasectl.py), so release rules are not duplicated in workflows or documentation.

## GitHub Actions

- `ci.yml`
  - Validates `release.toml`
  - Runs `releasectl` unit tests
  - Runs the tool's fast test suite
  - Verifies GUI builds on native runners according to the matrix
- `release.yml`
  - Listens for `*/v*` stable tags
  - Resolves the target tool from the tag
  - Builds and stages CLI / GUI release assets
  - Creates the GitHub Release and uploads assets
- `release-rc.yml`
  - Listens for `*/v*-rc.N` tags
  - Resolves the target tool from the tag
  - Uses the same build matrix and publishes the result as a prerelease

Release asset formats and platform matrices are defined by each tool's `release.toml`. If a downloaded Linux / macOS binary lacks execute permission, run `chmod +x <file>` before starting it.

## Recommended Release Steps

1. Update `version` in `services/<tool>/release.toml`
2. Review the root README, relevant service README, and long-lived docs under `docs/zh/` and `docs/en/` to confirm concise entry points, mirrored content, and valid links
3. Run `make release-version-stable TOOL=<tool> BASE_VERSION=<X.Y.Z> CHECK=1` or `make release-version-rc TOOL=<tool> BASE_VERSION=<X.Y.Z> RC=<N> CHECK=1`
4. Run `make release-preflight TOOL=<tool> RELEASE_TAG=<tag>`
5. Push the generated tag
6. Wait for the matching `release.yml` or `release-rc.yml` run to finish
