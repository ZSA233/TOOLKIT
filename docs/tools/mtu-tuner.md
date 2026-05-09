# mtu-tuner

`mtu-tuner` 是一个独立维护、可从 monorepo 迁出的网络 MTU 调优工具。源码位于 [services/mtu_tuner](../../services/mtu_tuner)。

## 组成

- CLI：`services/mtu_tuner/cmd/cli`
- GUI：`services/mtu_tuner/cmd/gui`
- GUI 前端：`services/mtu_tuner/cmd/gui/frontend`
- 发布真源：`services/mtu_tuner/release.toml`

## 当前公开发布目标

- CLI
  - Linux `amd64`
  - macOS `arm64`
  - Windows `amd64`
- GUI
  - Linux `amd64`
  - macOS `arm64`
  - Windows `amd64`

GUI 产物统一打成包含以下内容的归档：

- `mtu-tuner` 或 `mtu-tuner.exe`
- `dist/` 静态资源目录

CLI 产物为单二进制归档。

## 开发与校验

```bash
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
make release-validate TOOL=mtu-tuner WORKFLOW=ci
make release-preflight TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.0.1
```

## 发布相关

- 正式 tag：`mtu-tuner/vX.Y.Z`
- RC tag：`mtu-tuner/vX.Y.Z-rc.N`
- 版本和目标矩阵以 [release.toml](../../services/mtu_tuner/release.toml) 为准

更完整的流程见 [docs/release-process.md](../release-process.md)。
