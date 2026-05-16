# 发布流程

## 单一真源

每个可发布工具都以 `services/<tool>/release.toml` 作为发布真源。该文件负责定义：

- 工具对外 ID 与版本号
- 允许的 Git tag 前缀
- CI、正式发布、RC 发布分别使用哪些构建目标
- 每个目标的原生 runner、目标平台和发布资产格式
- 对应工具的 Go module 与 GUI 前端 lockfile 路径，供 workflow 动态装配依赖

## Tag 约定

- 正式版：`<tool>/vX.Y.Z`
- 候选版：`<tool>/vX.Y.Z-rc.N`

版本号只在对应工具的 `release.toml` 中维护。打 tag 前先更新该文件，再用仓库命令生成和校验 tag。GitHub Actions 会根据 tag 前缀反解目标工具，因此新增工具只需要接入自己的 `release.toml` 和构建目标。

## 本地命令

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

这些命令都委托给 [scripts/releasectl.py](../../scripts/releasectl.py)，避免在 workflow 或文档里重复维护发布规则。

## GitHub Actions

- `ci.yml`
  - 校验 `release.toml`
  - 运行 `releasectl` 单测
  - 运行工具的快速测试
  - 按 matrix 在原生 runner 上验证 GUI 构建
- `release.yml`
  - 监听 `*/v*` 正式 tag
  - 根据 tag 反解目标工具
  - 构建并整理 CLI / GUI 发布资产
  - 创建 GitHub Release 并上传资产
- `release-rc.yml`
  - 监听 `*/v*-rc.N` tag
  - 根据 tag 反解目标工具
  - 使用同一套构建矩阵，但发布为 prerelease

发布资产格式和平台矩阵以对应工具的 `release.toml` 为准。Linux / macOS 下载的二进制如果缺少执行权限，可先运行 `chmod +x <file>`。

## 推荐发布步骤

1. 更新 `services/<tool>/release.toml` 中的 `version`
2. 巡检根 README、相关子项目 README、`docs/zh/` 与 `docs/en/` 下的长期文档，确认入口简洁、镜像同步、链接有效
3. 运行 `make release-version-stable TOOL=<tool> BASE_VERSION=<X.Y.Z> CHECK=1` 或 `make release-version-rc TOOL=<tool> BASE_VERSION=<X.Y.Z> RC=<N> CHECK=1`
4. 运行 `make release-preflight TOOL=<tool> RELEASE_TAG=<tag>`
5. 推送生成出的 tag
6. 等待对应的 `release.yml` 或 `release-rc.yml` 完成
