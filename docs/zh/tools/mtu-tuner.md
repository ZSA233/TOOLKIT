# mtu-tuner

`mtu-tuner` 是一个独立维护、可从 monorepo 迁出的网络 MTU 调优工具。源码位于 [services/mtu_tuner](../../../services/mtu_tuner)。

## 工具定位

`mtu-tuner` 通过自动测试定位更合适的 MTU，用于缓解 MTU blackhole 问题。工具提供 CLI 和桌面 GUI 两种入口；GUI 使用 Wails v3 本地绑定，不启动独立 HTTP 服务。

## 源码结构

- `cmd/cli`：命令行入口
- `cmd/gui`：桌面 GUI 入口
- `cmd/gui/frontend`：React + TypeScript + Vite 前端
- `internal`：项目私有业务、领域逻辑和基础设施实现
- `internal/views`：生成的接口、provider runtime 和 Wails transport
- `scripts/blueprint`：接口 blueprint 真源
- `release.toml`：版本号、发布产物和目标平台矩阵真源

## 接口生成

接口和生成配置的真源在下面两个位置：

- `scripts/blueprint/**`
- `scripts/api-blueprint.toml`

不要手改以下生成产物：

- `internal/**/gen_*`
- `cmd/gui/frontend/src/lib/api/**/gen_*`

常用生成命令：

```bash
make mtu-tuner-api-check
make mtu-tuner-api-gen-all
make mtu-tuner-api-gen-golang
make mtu-tuner-api-gen-typescript
make mtu-tuner-api-gen-wails
```

## 开发命令

```bash
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
make mtu-tuner-run CMD=cli
make mtu-tuner-gui-run
```

## 发布相关

- 正式 tag：`mtu-tuner/vX.Y.Z`
- RC tag：`mtu-tuner/vX.Y.Z-rc.N`
- 版本号、发布产物和目标平台矩阵以 [release.toml](../../../services/mtu_tuner/release.toml) 为准

仓库级发布流程见 [发布流程](../release-process.md)。
