# mtu_tuner AGENTS

本文件是 `services/mtu_tuner` 的子项目补充规范，进入本目录工作时，需同时遵守仓库根目录的 `AGENTS.MD`。

## 1. 项目定位

- `services/mtu_tuner` 按“未来可迁出”的独立桌面工具维护。
- 默认把本项目看作独立 Go/Wails 仓库，只允许保留必要的 monorepo 挂接，例如根 `makefile`、`go.work`、本地 `api-blueprint` 生成命令入口，以及明确批准的共享基础层。
- 不要为了省事直接依赖其他业务服务、其他项目私有前端、或与本工具业务强耦合的 adapter / 组件 / 脚本；只有纯复用价值明确、且已被批准进入 `libs/appkit` 或 `libs/utils` 的共享基础能力可以直接依赖。

## 2. 目录职责

- `cmd/cli`：命令行入口，只做参数解析、依赖装配和结果输出。
- `cmd/gui`：Wails v3 桌面入口，只做应用装配、service 注册和静态资源挂载。
- `cmd/gui/frontend`：项目私有前端壳与界面实现，不依赖其他业务项目的前端工作区。
- `internal/`：本项目私有领域、服务、基础设施和手写 transport 接线。
- `internal/views/`：`api-blueprint` 生成的路由 core、provider runtime 与 Wails transport。
- `scripts/blueprint`：Blueprint DSL 真源；`scripts/api-blueprint.toml` 为生成配置。

## 3. 生成代码边界

- 新增或修改接口时，先改 `scripts/blueprint/**`，再重生成。
- 禁止手改：
  - `internal/**/gen_*`
  - `cmd/gui/frontend/src/lib/api/**/gen_*`
- 允许手改并长期维护：
  - route 包下非 `gen_*` 的实现文件
  - Wails app shell
  - frontend feature 与手写 transport shim

## 4. 独立性要求

- 新增能力时，优先写成本项目私有实现，不以“仓库里已有类似工具”为由直接 import 进来。
- 配置、任务状态、平台命令执行、Clash/Mihomo 解析、浏览器探测等能力，都应在本项目内自洽落地。
- 只有纯 monorepo 级挂接可留在仓库公共层，例如 `make/mtu-tuner.mk`、`go.work`。

## 5. Go 与前端拆分

- Go 文件命名要直接表达入口或链路，例如 `detect_interface.go`、`apply_mtu.go`、`run_sweep.go`。
- 不使用 `helper`、`util`、`misc` 这类模糊文件名。
- React 页面按功能拆到 `src/features/{feature}`，主入口保持薄。
- 单页工具也不要把状态、事件订阅、布局、表单和日志逻辑长期堆在一个超大文件里。

## 6. 文档维护

- `services/mtu_tuner/README.md` 只作为子项目入口，可保留工具定位、适用场景、核心目录、常用命令和中英文详细文档链接，不承载详细开发流程、发布流程或一次性变更说明。
- mtu-tuner 的长期工具说明维护在仓库级 `docs/zh/tools/mtu-tuner.md` 与 `docs/en/tools/mtu-tuner.md`，两侧内容必须同步表达同一组事实。
- `api-blueprint` 版本、包来源覆盖、本地工具源码覆盖等生成工具链细节，应由 `make/mtu-tuner.mk`、生成配置或专门工具文档承载，不写入子项目 README。
- 打包、发布资产、平台矩阵和 tag 规则以 `release.toml` 与仓库发布文档为准；README 只链接这些真源，不重复展开细节。

## 7. 修改前自检

- 本次改动是否保持了项目可迁出的独立性，没有偷偷依赖 monorepo 业务代码。
- 是否通过 Blueprint 真源驱动生成，而不是手改生成产物。
- `cmd/cli`、`cmd/gui`、frontend 入口是否仍然保持薄。
- Go 文件名和前端 feature 是否能让 reviewer 快速定位入口与职责。
- 如果改动了 README 或长期文档，是否保持子项目 README 只表达稳定入口、核心命令和导航，并同步更新 `docs/zh/` 与 `docs/en/` 的镜像内容。
