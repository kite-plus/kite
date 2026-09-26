# Kite 插件系统设计

> 状态：第一版已实现，范围和落地记录见 [§0.1](#01-第一版实施方案2026-09-26-定) · 最近更新：2026-09-26
> 上级文档：[architecture.md](architecture.md) · 姊妹文档：[theme-system.md](theme-system.md)
> `[EV]` 标记的结论有既有项目的实证支撑，来源见 [证据来源](#证据来源)。

---

## 0. 读这篇之前先看这一节

本文档描述的**绝大部分内容在 M8 才实现**。

但 [§12 V1 必须预留的东西](#12-v1-必须预留的东西) 里的三件事**必须在 M0 就做**，总成本约 200 行代码。不做它们，M8 就是一次 Core 重写。

如果你只有五分钟，读 §12。

### 0.1 第一版实施方案（2026-09-26 定）

插件从 M8 提前到第一版正式发布之前。第一版只做两种能力，其余按本文后面的设计留到以后：

| 能力 | 第一版 | 说明 |
|---|---|---|
| `client`：往页面注入代码和资源 | ✅ | 在 `plugin.yaml` 里声明，不写 WASM 也能做：统计、评论组件、公式渲染这一类插件大多只需要这个 |
| `build`：构建期处理内容 | ✅ | WASM，挂在已有的 HookBus 上：`transform_markdown`、`transform_html`、`build_complete` |
| `runtime`：服务端接口和存储 | ❌ | 依赖服务端部署，之后再做；因此第一版没有 `network`、`storage` 权限 |
| `admin`：后台扩展 | ❌ | 插件的设置页不算，它由 `settings` 自动生成 |

**和前文设计不同的决定：**

1. **直接采用 Extism 的 ABI 和它的 Go 宿主 SDK**（`github.com/extism/go-sdk`，底层是 wazero），不再自定义 `kite_alloc` / `kite_free`。§5.1 本来就要求照抄 Extism；直接用它，Rust、JS、Go、AssemblyScript 等语言现成的 PDK 就都能写 Kite 插件，Kite 不必先自己做 SDK。§5.2 的导出名作废，`PluginABIVersion` 改为标识 Kite 的钩子约定（函数名和 JSON 形状）。
2. **官方插件用标准 Go 写**（Go 1.24 起支持 `//go:wasmexport`，`GOOS=wasip1 -buildmode=c-shared`），不需要 TinyGo。2026-09-26 实测：2.3 MB 的模块编译约 0.5 秒（有编译缓存后只发生一次），实例化约 1 ms，调用约 16 µs。
3. **第一版不做 `kite.lock` 的插件表**。插件和主题一样放在站点仓库的 `plugins/<id>/` 里、随仓库提交，来源和版本由 git 记录；构建期插件没有任何权限，被替换也拿不走东西。等有了 `runtime` 能力和网络权限，再按 §10 把授予的权限钉进 lock。
4. **构建期插件拿不到真实时间和随机数**：wazero 默认给的是假时钟和固定种子，正好满足 §6.3 对纯函数的要求。

**插件包**：`plugins/<id>/` 下的 `plugin.yaml`（身份、`inject` 注入规则、`hooks`、`settings`）、可选的 `plugin.wasm`、可选的 `assets/`（发布到站点的 `plugins/<id>/` 下）。`settings` 的字段类型和后台表单与主题完全共用。

**站点配置**：`kite.yaml` 的 `plugins.enabled` 是启用的插件列表，顺序即执行顺序；`plugins.settings.<id>` 是各插件的设置。停用插件不会丢掉它的设置。

**后台与命令行**：「系统 → 插件」列出、上传 zip 安装、启用和停用、设置、删除；安装时说明插件会往页面里加什么、从哪些网站加载内容。命令行 `kite plugin list / add / remove / verify / new`。

**官方插件**（各自独立仓库 `kite-plus/plugin-<name>`，和主题的规则一致）：评论（Giscus / Waline / Twikoo）、统计（百度统计 / Umami / Google Analytics）、站内搜索（构建期生成索引 + 页面搜索框，纯静态可用）、公式与图表（KaTeX / Mermaid）。它们同时是检验插件接口够不够用的测试品。

**分三步**：① 插件包、配置、注入与资源、设置、后台和命令行；② WASM 运行时和构建期钩子；③ 官方插件。

**落地记录（2026-09-26）**：三步都已完成，用法写在 [reference.md 的 Plugins 一节](../reference.md#plugins)。和上面的方案相比，定下来的细节有：

1. **实例怎么复用**：transform 钩子每页调用一次，页面在所有核上并发渲染，所以共用一个实例池（每个核最多留一个空闲实例），实测一次调用约 0.2 ms；插件因此不能在两次调用之间保存状态。`build_complete` 每次构建用全新实例（约 2.6 ms、分配 8.7 MB），这样即使插件读了假时钟或随机数，同一个站点两次构建的输出也完全一样。出错或超时的实例直接丢弃。
2. **限制**：内存 64 MiB；每页 10 秒，`build_complete` 2 分钟。出错、超时、试图联网都会让构建失败，错误里带插件名。`build_complete` 写出的文件只能落在输出目录的 `plugins/<id>/` 下。
3. **编译**：进程内按模块的 sha256 缓存，最多留 16 个（预览时作者反复重编插件，旧模块不能一直占着内存）；磁盘缓存在 `.kite/cache/wasm`。插件被开启时（后台开关、改设置、`kite plugin enable`）先编译并检查声明的钩子都已导出，列表页不编译。
4. **钩子的 JSON**：输入都带 `settings`、`site`，transform 钩子还带 `page`（地址、种类、标题，单页上还有条目的 id、类型、params、分类、发布时间），字段与注入模板里的 `.Page` 同源；返回 JSON，或什么都不返回表示不改。`build_complete` 拿到除 404 外的全部页面，每页带纯文本 `text`（与字数统计同一套规则，同时修正了纯文本里残留 Markdown 转义的问题）。
5. **清单新增两项**：`hosts` 声明插件自带脚本会访问的网站，后台和注入代码里检测到的一起列出；注入规则的 `skip` 按 front matter 排除页面，如 `comments: false`，`no`、`off` 也算关闭。
6. **代价**：Extism SDK 连带引入 wazero 和它的追踪依赖，二进制增加约 5 MB（24.9 MB → 30.0 MB）。
7. **官方插件**：统计多支持了 Plausible；公式插件的 WASM 在渲染前保护公式（`$$` 块转成 `math` 代码块，行内公式转义标点后放进 `\(` `\)`，价格和代码不动），搜索插件的 WASM 在构建完成时写出索引。

---

## 1. 目标与非目标

### 目标

1. **安全** —— 第三方插件不能读用户的文件系统、不能起进程、不能任意联网。这是相对于所有"加载原生代码"方案的根本优势。
2. **跨平台、零依赖** —— 用户不需要安装 Wasmtime / Node / PHP / Python。插件运行时 embed 在 Kite 二进制里。
3. **跨语言** —— Go / Rust / Zig / AssemblyScript 都是一等公民。
4. **同时服务 Static 与 Dynamic** —— 一个插件可以在构建期跑、在请求期跑、在浏览器里跑、在 Admin 里跑，由 Capability 声明。
5. **不破坏增量构建** —— 构建期插件必须是纯函数，且其输出参与缓存键。

### 非目标

- 不追求原生级性能。需要极致性能的能力应该进 Core，不是做成插件。
- 不支持插件替换 Core 的核心行为（Store、读模型、Publisher 的选择权属于用户配置，不属于插件）。
- v1 不做插件市场、不做签名审核。

---

## 2. 为什么是 WebAssembly

### 2.1 被排除的方案

| 方案 | 排除理由 |
|---|---|
| **Go `plugin.so`** | 必须与主程序用**完全相同**的 Go 版本、相同的依赖版本、相同的编译参数构建，否则加载失败。不支持 Windows。无法卸载。**没有任何沙箱——插件与主程序同一地址空间，一个 panic 带走整个进程，一个恶意插件拥有全部权限。** 对一个要做插件生态的产品，这是不可接受的。 |
| **子进程 + RPC**（如 HashiCorp go-plugin） | 沙箱靠操作系统，跨平台一致性差；每个插件一个进程，内存与启动成本高；分发的是平台相关二进制，插件作者要为每个 OS×ARCH 出包。 |
| **嵌入脚本语言**（Lua / JS / Starlark） | 单一语言绑定；JS 引擎体积大；Lua 生态与 Web 开发者错位；沙箱要自己从头做。 |
| **全部编译进 Core** | 不是扩展系统。Core 会无限膨胀，且每加一个功能都要发版。 |

### 2.2 选择 wazero

**wazero**：纯 Go 实现的 WebAssembly 运行时，**零 CGO**，可直接 embed。

- 用户侧零安装
- 交叉编译不破（与 [architecture.md §18](architecture.md#18-deployment-model) 的 Single Binary 目标一致）
- 沙箱是 WASM 规范自带的，不是自己发明的
- 插件是平台无关的 `.wasm` 文件，一次构建到处运行

---

## 3. wazero 的真实约束

**选型前必须知道这些，它们直接决定运行模型的设计。** 全部有官方文档或生产案例佐证 `[EV]`。

### 3.1 编译器不是全平台可用 —— 而且会 panic

`NewRuntimeConfigCompiler` 只支持 **amd64 / arm64**（Linux / Windows / macOS / 部分 BSD），在不支持的平台上**直接 panic**。
解释器全平台可用（含 riscv64），但官方描述是"通常慢一个数量级（10×）或更多"。

→ **设计约束**：启动时检测能力，不支持则降级到解释器并打印**一次**告警，**绝不让它 panic**。

### 3.2 module instance 不是并发安全的

`Runtime` 和 `CompiledModule` 可以跨 goroutine 共享；**`api.Module` 实例不行**——每个并发调用需要自己的实例。
同一个 `Runtime` 里不允许重名模块，多实例要用 `WithName("")` 匿名实例化。

→ **设计约束**：编译一次 → 共享 `CompiledModule` → 每次调用从 `sync.Pool` 取实例。

### 3.3 线性内存只增不减

每次 grow 都会重新分配（除非 `WithMemoryCapacityFromMax(true)`）。长生命周期的池化实例会**永久保留内存高水位**。

→ **设计约束**：实例按「调用次数」或「内存高水位」回收重建，不能无限复用。

### 3.4 编译耗时是秒级，不是毫秒级

生产案例（Arcjet）：在服务器启动时一次性编译全部模块，"耗时数秒"；他们用 Wizer 预初始化来规避运行时编译，性能预算是 p50 10ms / p99 30ms。他们还指出：**无法从 Go 侧 profile 进 WASM 内部**，只能测顶层调用耗时。

→ **设计约束**：
- `NewCompilationCacheWithDir(".kite/cache/wasm")` 是**必须的**，尤其对 `kite build` 这种短命进程
- 缓存键 = `sha256(wasm) ‖ wazero_version`
- ⚠️ 已知并发 bug：从文件缓存加载的模块可能在入口预编译完成前被发布，并发实例化时 panic。**缓解办法就是确保只产生一个 `CompiledModule` 并共享它** —— 这与 §3.2 的设计正好一致

### 3.5 TinyGo 作为 guest 的坑很深

- `reflect` 不完整 → **`json.Marshal(struct)` 直接 panic**（`unimplemented: (reflect.Type).Name()`）
- goroutine 通过 Binaryen Asyncify 实现，有**20~100% 的运行时与代码体积开销**
- 从 `main` 以外的导出函数创建 goroutine 会 panic
- 行为等同 `GOMAXPROCS=1`
- 好处：二进制比 big-Go 小 10~20×

`GOOS=wasip1`（官方 Go）给你完整 stdlib，但模块是数 MB、每个实例带一整套 runtime + GC——适合一个长驻插件，不适合池化多个。

→ **设计约束（重要）**：**不要规定插件必须用 Go 写，要规定 ABI。**
这样 Rust / Zig / AssemblyScript 是一等公民，TinyGo 的 `reflect` 缺陷就不再是 Kite 的问题。官方 SDK 优先做 **Rust**，Go SDK 明确标注 TinyGo 的限制。

---

## 4. 运行模型

```
                    kite 启动 / kite build 开始
                              │
                   ┌──────────▼───────────┐
                   │ 读 kite.lock 的插件表 │
                   └──────────┬───────────┘
                              │ 校验 sha256
                   ┌──────────▼───────────┐
                   │  编译（串行，一次性） │◀── CompilationCache(.kite/cache/wasm)
                   └──────────┬───────────┘
                              │ 产出共享的 CompiledModule
                   ┌──────────▼───────────┐
                   │  校验 kite_abi_version │◀── 不匹配 → 拒绝加载，不降级
                   └──────────┬───────────┘
                              │
                   ┌──────────▼───────────┐
                   │  注册声明的 Hook      │──▶ HookBus
                   └──────────┬───────────┘
                              │
         ┌────────────────────▼────────────────────┐
         │        调用时：sync.Pool 取实例          │
         │  ctx timeout · 内存页上限 · 权限门控      │
         └────────────────────┬────────────────────┘
                              │
                 调用 N 次 或 内存高水位超阈值
                              │
                   ┌──────────▼───────────┐
                   │      回收并重建       │
                   └──────────────────────┘
```

**每次调用强制施加：**

| 约束 | 默认值 | 可配置 |
|---|---|---|
| `context.WithTimeout`（wazero 尊重取消） | build hook 5s / request hook 200ms | ✅ |
| 线性内存上限 | 64 MiB | ✅ |
| 实例复用次数上限 | 1000 次 | ✅ |
| 实例内存高水位阈值 | 32 MiB | ✅ |

超时 / 超限的处理：**构建期 → 构建失败并指名插件；请求期 → 跳过该 Hook，记录并在 Admin 告警**。绝不静默吞掉。

---

## 5. Host ABI

### 5.1 原则：照抄 Extism，不要自己发明

内存与句柄协议是**已解决的问题**。Extism kernel 的设计（bump allocator + 块头以支持复用、offset 0 表示 NULL/失败、`input_*` / `output_*` / `error_*` 三组通道）已经被多语言 PDK 验证过 `[EV]`。

自己发明一套只会在"字符串怎么跨边界""错误怎么传回来""内存谁释放"这些问题上重新踩一遍坑。

### 5.2 Guest 必须导出 `[M0 冻结常量]`

```
kite_abi_version() -> u32           // 最先调用；不匹配直接拒绝加载
kite_alloc(size u32) -> u32         // 返回线性内存偏移，0 = 失败
kite_free(ptr u32, size u32)
kite_hook_<name>(ptr u32, len u32) -> i64   // 返回 (ptr<<32)|len 打包
```

### 5.3 Host 提供的导入（module 名 `"kite"`）

| 函数 | 作用 | 权限键 |
|---|---|---|
| `log(level u32, ptr, len)` | 结构化日志 | 默认允许 |
| `content_query(ptr, len) -> i64` | 走 `ContentReader`（[architecture.md §9.1](architecture.md#91-读侧一个读模型一个实现)） | `content: [read]` |
| `content_apply(ptr, len) -> i64` | 走 `ContentWriter.Apply(ChangeSet)` | `content: [write]` |
| `config_get(ptr, len) -> i64` | 读插件自身配置与站点公开配置 | 默认允许 |
| `kv_get(ptr, len) -> i64` | 插件私有 KV | `storage: [plugin]` |
| `kv_set(ptr, len) -> i64` | 插件私有 KV | `storage: [plugin]` |
| `http_request(ptr, len) -> i64` | 出站 HTTP（域名白名单） | `network: [<domain>…]` |
| `media_get(ptr, len) -> i64` | 读媒体元数据 | `media: [read]` |
| `route_register(ptr, len) -> i64` | 注册 HTTP 路由 | `route: [register]`，仅 `runtime` capability |

**每一个 host 函数入口处都做权限校验，不信任插件自述。** 权限来自 `kite.lock` 里解析并钉死的清单，不是运行时从 `plugin.yaml` 现读的。

### 5.4 数据编码

| 场景 | 编码 | 理由 |
|---|---|---|
| 生命周期 / 配置 / 低频 Hook | **JSON** | 简单、跨语言、可读、易调试 |
| 热路径 Hook（如 `TransformHTML`） | 固定二进制布局或 MessagePack | 避免大文档的 JSON 编解码开销 |

**起步全部用 JSON。** WIT / Component Model / Protobuf 的优化**明确推迟到有真实性能数据之后**——现在选它们只会增加插件作者的门槛，收益未经验证。

### 5.5 错误传递

- Guest 返回的 `i64` 高 32 位为 0 且低 32 位为 0 → 表示无输出
- 错误走独立的 `error_set` / `error_get` 通道，不混在输出里
- Host 侧把错误包装为 `PluginError{PluginID, Hook, Message, Stack?}`，**永远带上是哪个插件**

---

## 6. HookBus —— V1 就要落地的部分

### 6.1 这是全篇最重要的一节

**HookBus 在 M0 就要实现，而且 V1 的内置功能必须走它。**

具体地说，下面这些在 M0 就要注册成 Hook，而不是写成 Core 内部的直接调用：

- sitemap 生成
- RSS 生成
- 代码语法高亮
- 图片处理
- SEO meta 标签注入

### 6.2 为什么这件事的优先级这么高

> 如果 V1 把这些写成 Core 内部的直接调用，到 M8 你会发现：**钩子点位不够、时机不对、拿不到需要的数据**。届时补钩子等于重构整个渲染与构建流程。

**让内置功能成为"第一批插件"，是验证扩展点够不够用的唯一可靠办法。** 你不可能靠想象设计出够用的 Hook 集合——只有真的用它实现了五个功能，才知道缺什么。

成本：约 200 行。收益：避免一次 Core 重写。这是整份设计里投入产出比最高的一条。

### 6.3 Hook 的三个元数据 `[M0 冻结]`

```go
type Hook interface {
    Name() string
    Phase() Phase              // PhaseBuild | PhaseRequest
    CacheKey() []byte          // 参与构建缓存键；无状态 Hook 返回版本号即可
}

type Phase int
const (
    PhaseBuild   Phase = iota  // 必须是纯函数
    PhaseRequest               // 可以有副作用
)
```

- **`Phase`**：`PhaseBuild` 的 Hook **必须是纯函数**——同样的输入必须产出同样的输出，不读时间、不读环境、不联网。否则直接摧毁 [architecture.md §14](architecture.md#14-build-engine) 的增量构建。
- **`CacheKey()`**：让插件的输出参与构建缓存键。**没有这个，换一个插件版本不会触发重建，用户会看到过期页面**——这是 R6 类风险。
- **形状必须是批量的**：见 [§8](#8-为什么-hook-必须是批量形状)。

---

## 7. Hook 目录

**命名一旦发布不可更改。插件 API 的稳定性优先级高于 Core 内部实现的优雅度。**

| Hook | Phase | 形状 | 用途 |
|---|---|---|---|
| `AppStart` / `AppStop` | Request | 单次 | 初始化 / 清理 |
| `ContentBeforeCreate` | Request | 单条 | 校验、补默认值 |
| `ContentAfterCreate` | Request | 单条 | 通知、索引 |
| `ContentBeforeUpdate` | Request | 单条 | 校验 |
| `ContentAfterUpdate` | Request | 单条 | 通知 |
| `ContentAfterDelete` | Request | 单条 | 清理关联数据 |
| `MarkdownBeforeRender` | Build | **整篇文档** | 预处理源文本 |
| `MarkdownNodeTransform` | Build | **批量节点** | 见 §8：声明式匹配 + 批量传递 |
| `MarkdownAfterRender` | Build | **整篇文档** | 后处理 HTML |
| `RenderBefore` / `RenderAfter` | Build | 整页 | 模板渲染前后 |
| `BuildStart` | Build | 单次 | 准备工作 |
| `BuildPageRendered` | Build | 单页 | 收集信息（如搜索索引） |
| `BuildComplete` | Build | 单次 | 产出附加文件（sitemap / RSS / search-index.json） |
| `PublishBefore` / `PublishAfter` | Request | ChangeSet | 发布前校验 / 发布后通知 |
| `RequestBefore` / `RequestAfter` | Request | 单请求 | 仅 Dynamic |
| `AdminMenuRegister` | Request | 单次 | 注册 Admin 菜单项 |

---

## 8. 为什么 Hook 必须是批量形状

### 8.1 按 AST 节点的钩子是死路

一篇 3000 字的文章有**上万个 inline 节点**（文本片段、强调、链接、代码片段……）。

如果每个节点都跨一次 host/guest 边界，每次的成本包括：

- guest 侧 `kite_alloc`
- host → guest 的一次 memcpy
- 一次 WASM 函数调用
- guest → host 的一次 memcpy
- JSON 编解码（如果用 JSON）
- 如果 guest 用了 goroutine，还要叠加 Asyncify 的 20~100% 开销 `[EV]`

**数量级估算**（待 M8 实测校正）：即使单次跨边界只要几微秒，两万个节点也是**几十毫秒一篇**；2000 篇文章的站点就是**分钟级**的构建时间——而 Kite 全量构建的目标是 2 秒以内（[architecture.md §29](architecture.md#29-每阶段验收标准)）。

结论：**节点级 Hook 不可用，不是"慢一点"，是根本不能做。**

### 8.2 正确的形状

**所有 Hook 一律以文档为单位，绝不以节点为单位。**

```
TransformMarkdown(doc) -> doc      # 整篇源文本
TransformHTML(doc) -> doc          # 整篇渲染后 HTML
```

**对于确实需要处理特定节点的插件，提供声明式节点匹配器：**

插件在 `plugin.yaml` 里声明"我处理什么"：

```yaml
nodeMatchers:
  - hook: MarkdownNodeTransform
    match:
      type: fenced_code
      attr:
        language: mermaid
```

宿主遍历 AST，**把所有命中的节点收集起来，一次性批量传给插件**，拿回批量结果再写回 AST。

这样一篇文章里有 5 个 mermaid 代码块 → **1 次跨边界调用**，而不是 5 次；更不是两万次。

### 8.3 其他性能红线

| 场景 | 判断 |
|---|---|
| Dynamic 下的 per-request Hook + **池化实例** | ✅ 微秒级，可接受 |
| Dynamic 下的 per-request Hook + **每请求实例化** | ❌ 毫秒级，致命 |
| 构建期 Hook 有副作用（读时间、联网） | ❌ 摧毁增量构建 |
| 构建期 Hook 未实现 `CacheKey()` | ❌ 插件升级后不重建，用户看到过期内容 |

---

## 9. Capability

因为 Kite 同时支持 Static 与 Dynamic，插件不能简单二分为"能跑 / 不能跑"。

| Capability | 执行时机 | Static | Dynamic |
|---|---|---|---|
| `build` | `kite build` 期间 | ✅ | ✅（构建静态导出时） |
| `runtime` | HTTP 请求期间 | ❌ | ✅ |
| `client` | 浏览器端（注入 JS / 组件） | ✅ | ✅ |
| `admin` | Admin 界面扩展 | ✅ | ✅ |

一个插件可以同时具备多个 Capability。

### **Admin 必须在安装时告诉用户"这个插件在你当前模式下哪些功能可用"**

这是 Capability 模型的用户价值所在，不是内部概念。例如在 Static 站点安装一个 `runtime + client` 的评论插件时，Admin 应当明确提示：

> 当前为静态模式。该插件的服务端评论功能不可用，将使用 Giscus 适配器（需配置 GitHub 仓库）。

---

## 10. Permission

### 10.1 清单

```yaml
permissions:
  content: [read]                  # read | write
  media:   [read]
  storage: [plugin]                # 只能是 plugin（私有 KV）
  network: [api.indexnow.org]      # 域名白名单，不允许通配符
  route:   [register]              # 仅 runtime capability 有意义
```

### 10.2 强制点

- **安装时**：Admin 展示完整权限清单并要求用户确认（类似手机 App 授权）
- **解析时**：授予的权限写进 `kite.lock`，运行时从 lock 读取，**不从 `plugin.yaml` 现读**——防止插件更新后静默扩权
- **调用时**：每个 host 函数入口校验，越权 → 拒绝 + 记录审计 + Admin 告警

### 10.3 默认不开放完整 WASI `[已冻结]`

禁止：filesystem、process、env、任意网络、时钟（构建期）。

> 这是相对 Go `plugin.so` 方案最重要的安全收益。**不能因为"开发方便"而放弃**——一旦开了口子，整个插件生态的安全模型就退化成"信任插件作者"。

`network` 必须是**具体域名**白名单：

```yaml
network: [api.indexnow.org]        # ✅
network: ["*.example.com"]         # ❌ 不支持
network: ["*"]                     # ❌ 不支持
```

---

## 11. `plugin.yaml`

```yaml
# ── 身份 ──
id: seo                            # 全局唯一，kebab-case
name: SEO
version: 1.0.0
apiVersion: kite/plugin/v1         # 插件契约版本；未知即拒绝
requires: ">=1.0.0 <2.0.0"         # 对 Kite 主程序的要求

description: 生成结构化数据、OG 标签与 IndexNow 推送
author: { name: Someone, url: https://example.com }
license: MIT
homepage: https://github.com/kite-plus/plugins/tree/main/seo

# ── 能力与权限 ──
capabilities: [build]
permissions:
  content: [read]
  storage: [plugin]
  network: [api.indexnow.org]

# ── 注册的 Hook ──
hooks:
  - name: RenderAfter
    phase: build
  - name: BuildComplete
    phase: build

# ── 声明式节点匹配（可选）──
nodeMatchers:
  - hook: MarkdownNodeTransform
    match: { type: fenced_code, attr: { language: mermaid } }

# ── 运行时资源限制（可被站点配置收紧，不可放宽）──
limits:
  memoryMiB: 64
  timeoutMs: 5000

# ── 设置 schema → Admin 自动生成配置页 ──
settings:
  - key: enable_indexnow
    type: boolean
    label: 启用 IndexNow 推送
    default: false
  - key: indexnow_key
    type: string
    label: IndexNow Key
    showIf: { enable_indexnow: true }

# ── client capability 的前端产物（可选）──
client:
  entry: dist/seo.js
  mount: head                      # head | body | admin
```

**settings schema 的字段类型与 Admin 渲染器与主题完全共用**——见 [theme-system.md §8.2](theme-system.md#82-settings-字段类型v1-基线-已冻结)。

---

## 12. V1 必须预留的东西

**只做这五件事，不引入 wazero 依赖，不写一行 WASM 代码。**

| # | 做什么 | 不做会怎样 |
|---|---|---|
| **1** | **HookBus 落地**，且 V1 的内置功能（sitemap / RSS / 高亮 / 图片）全部注册为 Hook | M8 发现钩子点位不够、时机不对、数据拿不到 → 重构渲染与构建流程 |
| **2** | Hook 的 **`Phase`** 分类（Build 必须纯 / Request） | 构建期混入副作用，增量构建永久不可用 |
| **3** | Hook 的 **`CacheKey()`** 方法 | 插件输出不参与缓存键，升级插件不重建 → 用户看到过期页面 |
| **4** | **`kite.lock` 预留插件字段**：`plugins: [{id, version, sha256, capabilities}]` | lock 文件格式变更很痛，且涉及所有已有站点 |
| **5** | **ABI 版本常量与导出名**（`kite_abi_version` / `kite_alloc` / `kite_free`） | 命名一旦发布就冻结，现在定零成本 |

**Hook 的形状（批量 vs 节点级）也必须现在定**——它决定了 Core 里 Markdown pipeline 的组织方式，是结构性的，不是接口签名问题。

---

## 13. 两个完整案例

Capability 模型不是抽象概念。下面两个例子展示同一个插件在两种模式下走完全不同的实现路径。

### 13.1 评论插件（`runtime + client`）

```
Dynamic 模式：
  浏览器 → POST /api/plugins/comment/submit
         → route_register 注册的路由
         → 插件的 RequestBefore Hook
         → kv_set / content_apply
         → 存入 Kite 的数据库
         → 渲染时通过 client capability 注入的 JS 拉取并展示

Static 模式（没有 runtime）：
  插件的 client 产物检测到 runtime 不可用
         → 走 Adapter 模式，对接 Giscus / Waline / Disqus
         → 配置项（GitHub 仓库 / Waline 服务地址）在 Admin 的插件设置页填
         → 构建期通过 RenderAfter Hook 注入对应的嵌入代码
```

**同一个插件包，同一份配置界面，两条实现路径。** 用户不需要为了换部署方式而换插件。

### 13.2 搜索插件（`build + runtime + client`）

```
Static 模式：
  BuildPageRendered Hook  → 收集每页的标题/摘要/正文片段
  BuildComplete Hook      → 产出 public/search-index.json
  client 产物             → 浏览器端加载索引，本地检索（MiniSearch / Fuse.js）

Dynamic 模式：
  route_register          → GET /api/search
  RequestBefore Hook      → content_query 走 ContentReader 的 TextQuery
  client 产物             → 请求服务端接口
```

注意 Dynamic 模式下搜索走的是 **`ContentReader.Query` 的 `TextQuery` 字段**，也就是[唯一读模型](architecture.md#02-只有一个读模型read-model)——插件不需要自己碰数据库。

---

## 14. SDK 与开发流程

### 14.1 插件作者不应该理解 WASM ABI

```rust
// Rust SDK（官方优先支持）
use kite_plugin::*;

#[kite::hook(RenderAfter, phase = "build")]
fn inject_og_tags(ctx: RenderContext) -> Result<Html> {
    let og = format!(r#"<meta property="og:title" content="{}">"#, ctx.page.title);
    Ok(ctx.html.insert_into_head(&og))
}

kite::register!();
```

```go
// Go SDK（标注 TinyGo 限制）
package main

import kite "github.com/kite-plus/plugin-sdk/go"

func main() {
    kite.OnRenderAfter(kite.PhaseBuild, func(ctx *kite.RenderContext) (string, error) {
        return ctx.HTML + "<!-- seo -->", nil
    })
    kite.Register()
}
```

### 14.2 为什么 Rust 优先

[§3.5](#35-tinygo-作为-guest-的坑很深) 的 TinyGo 限制是实打实的：`json.Marshal(struct)` 会 panic、goroutine 有 Asyncify 税、从非 main 导出函数创建 goroutine 会 panic。

Go SDK 当然要做（Kite 本身是 Go 项目，社区重叠度高），但文档必须**明确写出这些限制**，并提供不依赖 `reflect` 的序列化路径。

### 14.3 开发流程

```bash
kite plugin new my-plugin --lang rust    # 脚手架
cd my-plugin
kite plugin build                        # → my-plugin.wasm
kite plugin test                          # 在 fixture 站点上跑
kite plugin install ./my-plugin.wasm      # 本地安装到当前站点
```

---

## 15. 安全模型

### 15.1 威胁面

| 威胁 | 缓解 |
|---|---|
| 插件读取用户的私钥 / SSH config | 不开放 filesystem |
| 插件把内容外传 | `network` 域名白名单 + 安装时展示 |
| 插件起进程 / 执行命令 | 不开放 process |
| 插件占满内存 / 死循环 | 内存页上限 + `context.WithTimeout` |
| 插件读环境变量拿 token | 不开放 env；`config_get` 只返回插件自身配置与站点**公开**配置 |
| 插件更新后静默扩权 | 权限钉死在 `kite.lock`，更新要重新确认 |
| 供应链：插件被替换 | `kite.lock` 里的 `sha256`，不匹配直接失败 |
| 插件 panic 拖垮主进程 | WASM trap 被 wazero 捕获，不影响宿主 |

### 15.2 不在 v1 威胁模型内

- 侧信道 / 计时攻击
- 插件之间的互相干扰（v1 不提供插件间通信）
- 恶意插件作者的身份认证（没有签名机制，靠 checksum + 人工信任）

---

## 16. 性能预算

M8 实现时要**实测**并写进 CI 基准，不能靠估算：

| 指标 | 目标 |
|---|---|
| 冷启动（含 5 个插件，有编译缓存） | < 200ms |
| 冷启动（无编译缓存，首次） | < 5s，且只发生一次 |
| 单次 build Hook 调用（中等文档） | < 5ms |
| 单次 request Hook 调用（池化实例） | < 500µs |
| 插件对 2000 篇全量构建的总开销 | < 20%（相对无插件基线） |
| 二进制体积增量（wazero embed） | < 8MB |

---

## 17. 明确不做的东西

| 不做 | 何时再看 |
|---|---|
| WASI filesystem / network preopen | 不做（与安全模型冲突） |
| 插件间通信 / 依赖 | V4 |
| 插件提供自定义内容类型 | V3，且需要 ContentType Registry 先开放 |
| 插件提供完整 Admin 页面（不止菜单项） | V4 |
| 热重载 | M8 之后 |
| 插件市场 / 注册表 / 签名审核 | V4；v1 用 Git URL + checksum |
| Component Model / WIT | 有真实性能数据后再评估 |
| 插件替换 Store / Publisher / 读模型 | 不做——这些是用户的配置权，不是插件的 |

---

## 18. 开放问题 `[待定]`

1. **Hook 的执行顺序** —— 多个插件注册同一个 Hook 时如何排序？倾向于：`plugin.yaml` 里声明 `priority`，同优先级按 lock 文件顺序，**且顺序必须是确定的**（否则破坏构建可重现性）。
2. **Hook 能否中止链条** —— `ContentBeforeCreate` 返回错误时是中止整个操作还是跳过该插件？倾向于：`Before*` 可中止，`After*` 不可。
3. **`client` capability 的产物如何参与 asset pipeline** —— 插件的 JS 要不要 fingerprint、要不要 bundle 进主题的 bundle？
4. **构建期插件的并行度** —— 实例池大小与 build worker pool 的关系。
5. **插件配置的 schema 迁移** —— 插件升级后 settings schema 变了，旧配置怎么办？

---

## 证据来源

**wazero**
- [wazero godoc](https://pkg.go.dev/github.com/tetratelabs/wazero) —— `NewRuntimeConfigCompiler` 的平台限制与 panic 行为、解释器 10× 慢、module instance 非并发安全、内存只增不减、`WithMemoryCapacityFromMax`
- [wazero 文档站](https://wazero.io/docs/)
- [wazero 编译缓存 cache.go](https://github.com/tetratelabs/wazero/blob/main/cache.go) —— `NewCompilationCacheWithDir`
- [wazero 文件缓存并发 bug #2547](https://github.com/wazero/wazero/issues/2547) —— 共享单个 `CompiledModule` 作为缓解
- [Arcjet：在生产环境用 Go + wazero 的经验](https://blog.arcjet.com/lessons-from-running-webassembly-in-production-with-go-wazero/) —— 启动时编译"耗时数秒"、Wizer 预初始化、无法从 Go 侧 profile WASM 内部、p50/p99 预算

**TinyGo guest**
- [TinyGo #2519：reflect 不完整导致 json.Marshal panic](https://github.com/tinygo-org/tinygo/issues/2519)
- [TinyGo 兼容性文档](https://tinygo.org/docs/guides/compatibility/)
- [TinyGo #3095：非 main 导出函数创建 goroutine 会 panic](https://github.com/tinygo-org/tinygo/issues/3095)
- [TinyGo #1101：Asyncify](https://github.com/tinygo-org/tinygo/issues/1101)
- [Asyncify 的 20~100% 开销](https://kripken.github.io/blog/wasm/2019/07/16/asyncify.html)
- [wazero 的 TinyGo 说明](https://wazero.io/languages/tinygo/)

**ABI 设计参考**
- [Extism kernel 实现](https://github.com/extism/extism/blob/main/kernel/src/lib.rs) —— alloc/free/length/load/store/input/output/error 的完整协议
- [Extism PDK 概念文档](https://extism.org/docs/concepts/pdk)
