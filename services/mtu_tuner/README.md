# mtu-tuner

`mtu-tuner` 是一个独立维护的网络 MTU 调优工具，目标是通过自动测试定位更合适的 MTU，缓解 MTU blackhole 问题。项目同时提供 CLI 和桌面 GUI，两条入口都位于 `services/mtu_tuner` 下。

GUI 使用 `Wails v3` 本地绑定，不启动独立 HTTP 服务。

## 目录说明

- `cmd/cli`：命令行入口
- `cmd/gui`：桌面 GUI 入口
- `cmd/gui/frontend`：GUI 前端，基于 React + TypeScript + Vite
- `internal`：本项目私有业务、领域逻辑和基础设施实现
- `internal/views`：由 `api-blueprint` 生成的接口、provider runtime 和 Wails transport
- `scripts/blueprint`：接口 blueprint 真源
- `release.toml`：当前工具的发版版本号、产物和目标平台配置

## 生成真源

接口和生成配置的真源在下面两个位置：

- `scripts/blueprint/**`
- `scripts/api-blueprint.toml`

不要手改以下生成产物：

- `internal/**/gen_*`
- `cmd/gui/frontend/src/lib/api/**/gen_*`

## 常用命令

接口校验与生成：

```bash
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-check
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-all
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-golang
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-typescript
API_BLUEPRINT_PROJECT=/path/to/api-blueprint make mtu-tuner-api-gen-wails
```

测试与构建：

```bash
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
```

运行入口：

```bash
make mtu-tuner-run CMD=cli
make mtu-tuner-gui-run
```

## 打包说明

Windows GUI 打包：

```bash
make mtu-tuner-gui-build-windows WINDOWS_GOARCH=amd64
make mtu-tuner-gui-package-windows WINDOWS_GOARCH=amd64
```

如果在 macOS 或 Linux 上交叉构建 Windows GUI，需要额外提供可用的 Windows C 工具链，因为 Wails GUI 依赖 CGO。例如使用 Zig：

```bash
make mtu-tuner-gui-package-windows \
  WINDOWS_GOARCH=amd64 \
  WINDOWS_CC='zig cc -target x86_64-windows-gnu' \
  WINDOWS_CXX='zig c++ -target x86_64-windows-gnu'
```

Windows GUI 打包输出目录：

`services/mtu_tuner/build/packages/gui/windows_<arch>/mtu-tuner/`

## 发布约定

- 正式 tag：`mtu-tuner/vX.Y.Z`
- RC tag：`mtu-tuner/vX.Y.Z-rc.N`
- 版本号、产物种类和平台矩阵统一以 `release.toml` 为准

仓库级发布流程见 [../../docs/release-process.md](../../docs/release-process.md)。
