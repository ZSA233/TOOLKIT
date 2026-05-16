# mtu-tuner

语言: 中文 | [English](../../docs/en/tools/mtu-tuner.md)

## 概述

`mtu-tuner` 是一个独立维护的网络 MTU 调优工具，目标是通过自动测试定位更合适的 MTU，缓解 MTU blackhole 问题。项目同时提供 CLI 和桌面 GUI，两条入口都位于 `services/mtu_tuner` 下。

GUI 使用 Wails v3 本地绑定，不启动独立 HTTP 服务。

## 适合什么场景

- 网络路径存在 MTU blackhole 迹象，需要快速定位可用 MTU。
- 希望用 CLI 做脚本化检测，或用桌面 GUI 做交互式调优。
- 需要在一个可独立发布的工具内维护网络探测、配置和桌面入口。

## 核心入口

| 入口 | 目录 | 用途 |
|:---|:---|:---|
| CLI | `cmd/cli` | 命令行运行和脚本化集成 |
| GUI | `cmd/gui` | Wails v3 桌面应用入口 |
| Frontend | `cmd/gui/frontend` | React + TypeScript + Vite 前端 |
| API blueprint | `scripts/blueprint` | 接口定义真源 |
| Release config | `release.toml` | 版本号、发布产物和平台矩阵真源 |

## 常用命令

接口校验与生成：

```sh
make mtu-tuner-api-check
make mtu-tuner-api-gen-all
```

测试、构建和运行：

```sh
make mtu-tuner-test GO_TEST_FLAGS=-short
make mtu-tuner-gui-frontend-build
make mtu-tuner-gui-build
make mtu-tuner-run CMD=cli
make mtu-tuner-gui-run
```

## 下一步

| 主题 | 文档 |
|:---|:---|
| 完整中文说明 | [docs/zh/tools/mtu-tuner.md](../../docs/zh/tools/mtu-tuner.md) |
| English guide | [docs/en/tools/mtu-tuner.md](../../docs/en/tools/mtu-tuner.md) |
| 发布流程 | [docs/zh/release-process.md](../../docs/zh/release-process.md) |
