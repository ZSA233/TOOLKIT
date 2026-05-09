# 发布流程

## 单一真源

每个可发布工具都以 `services/<tool>/release.toml` 作为发布真源。当前已接入的公开工具是 `mtu-tuner`，其真源文件位于 [services/mtu_tuner/release.toml](../services/mtu_tuner/release.toml)。

`release.toml` 负责定义：

- 工具对外 ID 与版本号
- 允许的 Git tag 前缀
- CI、正式发布、RC 发布分别使用哪些构建目标
- 每个目标的原生 runner、目标平台和归档格式
- 对应工具的 Go module 与 GUI 前端 lockfile 路径，供 workflow 动态装配依赖

## Tag 约定

- 正式版：`mtu-tuner/vX.Y.Z`
- 候选版：`mtu-tuner/vX.Y.Z-rc.N`

版本号只在 `release.toml` 中维护。打 tag 前先更新该文件，再用仓库命令生成和校验 tag。GitHub Actions 会根据 tag 前缀自动反解出对应工具，因此未来新增工具只需要接入自己的 `release.toml` 和构建目标，不需要再复制新的 release workflow。

## 本地命令

```bash
make release-validate TOOL=mtu-tuner WORKFLOW=ci
make release-version-show TOOL=mtu-tuner
make release-version-stable TOOL=mtu-tuner BASE_VERSION=0.1.0 CHECK=1
make release-version-rc TOOL=mtu-tuner BASE_VERSION=0.1.0 RC=1 CHECK=1
make release-tag-check TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.1.0
make release-preflight TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.1.0
make release-local TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.1.0
make release-matrix TOOL=mtu-tuner WORKFLOW=release
make release-metadata TOOL=mtu-tuner TAG=mtu-tuner/v0.1.0
```

这些命令都委托给 [scripts/releasectl.py](../scripts/releasectl.py)，不再在 workflow 或文档里手写重复规则。

## GitHub Actions

- `ci.yml`
  - 校验 `release.toml`
  - 运行 `releasectl` 单测
  - 运行 `mtu-tuner` 的短测试
  - 按 matrix 在原生 runner 上验证 GUI 构建
- `release.yml`
  - 监听 `*/v*` 正式 tag
  - 根据 tag 反解目标工具
  - 构建并归档 CLI / GUI 产物
  - 创建 GitHub Release 并上传资产
- `release-rc.yml`
  - 监听 `*/v*-rc.N` tag
  - 根据 tag 反解目标工具
  - 使用同一套构建矩阵，但发布为 prerelease

Linux GUI runner 会在工作流里安装 Wails 所需系统依赖。GUI 发布包统一包含可执行文件和 `dist/` 静态资源目录，避免运行时缺少前端资源。

## 推荐发布步骤

1. 更新 [services/mtu_tuner/release.toml](../services/mtu_tuner/release.toml) 中的 `version`
2. 运行 `make release-version-stable TOOL=mtu-tuner BASE_VERSION=<X.Y.Z> CHECK=1` 或 `make release-version-rc TOOL=mtu-tuner BASE_VERSION=<X.Y.Z> RC=<N> CHECK=1`
3. 运行 `make release-preflight TOOL=mtu-tuner RELEASE_TAG=<生成出的 tag>`
4. 推送生成出的 tag
5. 等待对应的 `release.yml` 或 `release-rc.yml` 完成
