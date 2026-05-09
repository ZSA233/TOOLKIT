# 工具箱仓库

这是一个面向可独立发布工具的 monorepo。当前公开交付重点是 `mtu-tuner`：一个用 Go + Wails 构建的 MTU 调优工具，保留 CLI 与桌面 GUI 两条入口。

English summary: this repository hosts standalone tools that can be released independently. The first public tool is `mtu-tuner`, with shared release automation at the repo root and tool-owned implementation under `services/`.

## 工具概览

- `services/mtu_tuner/`: `mtu-tuner` 源码、GUI 前端、发布元数据
- `scripts/releasectl.py`: 仓库内置发布控制脚本，负责解析 `release.toml`、校验 tag、生成 GitHub Actions matrix、打包发布产物
- `.github/workflows/`: CI、RC 发布、正式发布工作流
- `docs/release-process.md`: 仓库级发布流程
- `docs/tools/mtu-tuner.md`: `mtu-tuner` 对外说明与产物约定

## 常用命令

```bash
make release-validate TOOL=mtu-tuner WORKFLOW=ci
make release-version-show TOOL=mtu-tuner
make release-version-stable TOOL=mtu-tuner BASE_VERSION=0.0.1 CHECK=1
make release-version-rc TOOL=mtu-tuner BASE_VERSION=0.0.1 RC=1 CHECK=1
make release-tag-check TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.0.1
make release-preflight TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.0.1
make release-matrix TOOL=mtu-tuner WORKFLOW=release
make mtu-tuner-test GO_TEST_FLAGS=-short
```

更多发布细节见 [docs/release-process.md](docs/release-process.md)。
