# Kite 设计文档

Kite 的产品与技术总体设计。这些文档是**开发期的约束来源**，不是事后补写的说明书——任何与文档冲突的实现都应该先改文档（走 PR + 讨论），再改代码。

## 文档索引

| 文档 | 内容 | 读者 |
|---|---|---|
| [architecture.md](architecture.md) | 总体架构：产品定位、Content Model、Storage Model、Build Engine、Publisher、Git Workflow、API、Roadmap、架构决策 | 所有贡献者，**先读这篇** |
| [theme-system.md](theme-system.md) | 主题系统：引擎选型、模板查找顺序、RenderContext 数据契约、函数命名空间、`theme.yaml`、Asset Pipeline、契约冻结流程 | 主题开发者、渲染层贡献者 |
| [plugin-system.md](plugin-system.md) | 插件系统：WASM 运行时、Host ABI、Hook 目录、Capability、Permission、`plugin.yaml`、SDK | 插件开发者、扩展层贡献者 |
| [roadmap.md](roadmap.md) | 路线图与实现现状：逐项核实的完成情况、验收标准核对、v1.0 收尾清单、分阶段计划、后台占位功能的归属 | 所有贡献者，排期前读 |
| [admin-backlog.md](admin-backlog.md) | 后台页面盘点、主流程风险、功能缺口、优先级与验收条件 | 后台开发与排期 |

## 阅读顺序

1. `architecture.md` 的 **§0（两个核心判断）** —— 全部设计的地基，不读这节后面都看不懂为什么。
2. `architecture.md` 的 **§8 Content Model** 与 **§9 Storage Model** —— 领域模型。
3. 按你要参与的方向，读 `theme-system.md` 或 `plugin-system.md`。
4. 动手前读 `architecture.md` 的 **§31 必须现在确定的架构决策** 与 **§32 现在不要设计的东西**。

## 文档约定

- **语言**：正文中文，专有名词（Content Model / Publisher / Capability / ChangeSet 等）保留英文。代码、注释、commit message 一律纯英文。
- **`[EV]` 标记**：该结论有既有项目的实证支撑（Hugo / Halo / TinaCMS / Decap / Kirby / go-git / wazero 等），来源见各文档末尾的「证据来源」。区分"我们的判断"与"别人踩过的坑"。
- **「现在留接口，不现在实现」**：文中大量出现。判断标准统一为——*将来做这件事时，如果只需新增代码、不需修改已有接口，那就现在不要做；反之必须现在做。*
- **状态标注**：`[已冻结]` 表示改动需要破坏性版本；`[设计中]` 表示实现前还可调整；`[待定]` 表示明确推迟决策。

## 当前状态

| 文档 | 状态 | 最近更新 |
|---|---|---|
| architecture.md | 设计中（M0 开工前的基线） | 2026-09-21 |
| theme-system.md | 设计中，契约计划于 **M5** 冻结 | 2026-09-21 |
| plugin-system.md | 设计中，实现计划于 **M8** | 2026-09-21 |
| roadmap.md | 持续更新 | 2026-09-23 |
| admin-backlog.md | 待办盘点 | 2026-09-24 |

里程碑定义见 [architecture.md §28 Roadmap](architecture.md#28-roadmap)，实际进度见 [roadmap.md](roadmap.md)。
