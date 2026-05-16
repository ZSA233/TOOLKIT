# Toolkit

[![GitHub Stars](https://img.shields.io/github/stars/ZSA233/TOOLKIT)](https://github.com/ZSA233/TOOLKIT/stargazers)
[![License](https://img.shields.io/github/license/ZSA233/TOOLKIT)](LICENSE)

语言: 中文 | [English](docs/en/README.md)

## 概述

`Toolkit` 是一个面向可独立发布小工具的仓库。每个工具都在 `services/` 下独立维护，并通过仓库统一的构建、校验和发布流程发版。

核心组织方式是：

```text
services/<tool> -> release.toml -> releasectl / GitHub Actions -> release assets
```

## 适合什么场景

- 需要把小型工具按独立版本、独立产物发布。
- 希望工具源码在 monorepo 中统一维护，但仍保持可迁出边界。
- 需要共享一套 release 配置、预检命令和 GitHub Actions 发布流程。

## 工具列表

| 工具 | 作用 | 跳转 |
|:---|:---|:---|
| `mtu-tuner` | 缓解 MTU blackhole 问题，通过自动测试找出更合适的 MTU，提供 CLI 和桌面 GUI 两种入口。 | [使用说明](docs/zh/tools/mtu-tuner.md) · [English](docs/en/tools/mtu-tuner.md) · [源码目录](services/mtu_tuner/) |

## 常用命令

```sh
make release-validate TOOL=mtu-tuner WORKFLOW=ci
make release-preflight TOOL=mtu-tuner RELEASE_TAG=<tag>
```

工具自己的测试、构建和运行命令见对应工具文档。

## 下一步

| 主题 | 文档 |
|:---|:---|
| 中文文档入口 | [docs/zh/README.md](docs/zh/README.md) |
| English documentation | [docs/en/README.md](docs/en/README.md) |
| mtu-tuner | [docs/zh/tools/mtu-tuner.md](docs/zh/tools/mtu-tuner.md) |
| 发布流程 | [docs/zh/release-process.md](docs/zh/release-process.md) |
