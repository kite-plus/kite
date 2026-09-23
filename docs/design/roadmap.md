# Kite 路线图与实现现状

> 状态：持续更新 · 最近核对：2026-09-23
> 里程碑的原始定义见 [architecture.md §28 Roadmap](architecture.md#28-roadmap)，验收标准见 [§29](architecture.md#29-每阶段验收标准)。
> 本文记录的是**对照代码和测试逐项核实后**的进度，不是对计划的复述；有疑问的项都实际运行确认过。

---

## 1. 结论

- **v1.0（M0–M4）的主体已经完成**，而且最难的几项都有测试兜底：
  - 派生索引删掉重建后逐字节一致，racy 时间戳规则生效，重复 ID 直接报错；
  - front matter 保真：只改标题时，文件的 diff 只有标题这一行；
  - build 和 serve 的输出逐字节一致，预览和最终页面也逐字节一致；
  - 发布只提交本次涉及的文件，不动用户暂存区里的其他改动。
- **§4 的收尾项只剩第 11 项**：要在 GitHub 组织下建仓库，需要仓库主人自己动手。另外 M4 还差一次在真实仓库上的端到端验收（§3）。
- **M5–M8 都还没开始**，其中 M5 有一部分已经提前做了。后台按设计稿画出的「即将推出」占位功能，各自归到哪个阶段见 §6。

## 2. 各里程碑完成情况

| 里程碑 | 状态 | 已实现 |
|---|---|---|
| **M0** 静态构建 | 完成 | 内容模型：ULID 作为 ID，ChangeSet 是唯一的写入口；YAML 和 TOML front matter 保真编解码；文件原子写；SQLite 派生索引；唯一的 `ContentReader`，支持复合游标分页和按标题/摘要/正文搜索；goldmark 渲染和代码高亮；URLResolver；主题引擎（查找顺序、命名空间化的函数、`apiVersion` 校验）和默认主题；BuildContext（冻结时钟、按投影记录依赖、缓存键）；HookBus，sitemap、RSS、高亮都走钩子；`kite build --verify`；CI 检查 import 边界、交叉编译和可重现发布 |
| **M1** serve | 完成 | 按请求从文件渲染；fsnotify 文件监听；热重载 |
| **M2** 只读后台 | 完成 | REST API，并从代码生成 OpenAPI；前端请求一律用生成的客户端；React 后台嵌入二进制；列表的筛选、排序、游标分页和搜索；索引一致性的三层机制：文件监听、stat 全树扫描、Git HEAD 哨兵（切分支时只重新索引变化的路径） |
| **M3** 可写后台 | 完成 | `PUT` + `If-Match` 走 `Apply(ChangeSet)`；在旧版本上保存时返回 409，并给出三方对比；由 schema 驱动的表单，内容字段、主题设置、站点设置共用；可视化编辑器（Tiptap）加 Markdown 源码模式（CodeMirror）；服务端渲染的预览；拖图进 page bundle；站点设置和主题设置页（改 `kite.yaml` 时保留注释和顺序）；`kite doctor --fix-ids` |
| **M4** Git 发布 | 基本完成，只差 §3 的端到端验收 | 发布前检查：不是仓库、子模块、游离 HEAD、有进行中的 merge/rebase/cherry-pick、缺 git-lfs、文件超出托管平台限制；`git commit --only` 只提交指定路径；`GIT_TERMINAL_PROMPT=0` 加空的 `GIT_ASKPASS`，缺凭据时立刻报错；`.kite/publish.lock` 加 `index.lock` 退避重试；从不强推；DeliveryState 和发布面板；`kite publish` 命令行；`kite init` 生成 GitHub Pages 部署 workflow 和发布定时文章的 `scheduled.yml`；远端有新提交且没有改到同样的文件时一键接到后面推送；hook 拒绝后可跳过 hooks 发布；「已部署」对接 GitHub Pages 的部署状态 |
| **M5** 主题契约 | 部分提前完成 | `apiVersion` 硬校验和 `requires` 检查；命名空间化的函数；由 `theme.yaml` 生成的主题设置页；`kite theme verify`。其余见 §5 |
| **M6** | 未开始 | 构建时已经按 OutputTarget 记录依赖和缓存键，只是跳过判断还没启用 |
| **M7** | 未开始 | 读模型已按双 Store 设计；单账号认证和 Docker 已提前完成 |
| **M8** | 未开始 | HookBus 已被内置功能使用 |

**规划外已完成的：**
- 单账号登录，带防暴力尝试；
- 没有终端也能用的网页安装引导；
- Docker 镜像和 compose；
- 后台中英文界面；
- 命令面板（⌘K）；
- 分类法页面；
- 后台界面按设计稿逐像素重做。

## 3. 验收标准核对

对照 [architecture.md §29](architecture.md#29-每阶段验收标准)。「未验证」表示机制已经实现，但没有测试或测量证明它达标。时延一项用 `make perf` 在 Apple M1 Pro（8 核）上测得，站点是生成的 2000 篇文章，每篇约 800 字，带代码块和 3 个标签。

| 里程碑 | 标准 | 结果 | 依据 |
|---|---|---|---|
| M0 | 在真实的 Hugo 内容仓库上产出可用站点 | 部分 | YAML 和 TOML（`+++`）front matter 都能读写（`TestATOMLPostReadsLikeItsYAMLTwin`）。还差 Hugo 的 shortcode：`{{< >}}` 会原样输出，语法要等 [theme-system.md §14](theme-system.md) 的开放问题定下来 |
| M0 | 2000 篇全量构建小于 2 秒 | 通过 | `make perf`：没有任何缓存时约 1.5 秒（其中建索引约 0.7 秒），索引已在时约 0.9 秒 |
| M0 | `kite build --verify` 在 CI 通过 | 通过 | CI 的可重现性步骤；`TestBuildIsReproducible` |
| M0 | `internal/content` 的 import 边界检查 | 通过 | `scripts/check-imports.sh`，由 CI 执行 |
| M1 | 改文件后，浏览器 500ms 内看到变化 | 通过 | `make perf`：从写文件到重新加载的页面显示新内容，三次中最慢约 390ms |
| M1 | build 和 serve 的 HTML 逐字节一致 | 通过 | `TestServedFilesAreByteIdenticalToBuiltOnes`，比较的是 build 写出的每个文件，RSS、sitemap 和静态文件也在内 |
| M2 | 2000 篇的仓库，列表首屏小于 300ms | 通过 | `make perf`：首屏的 6 个请求同时发出，约 25ms |
| M2 | 删掉缓存重建索引，结果逐字节一致 | 通过 | `TestRebuildProducesIdenticalRows` |
| M2 | 切分支后 3 秒内列表更新，且只重新索引变化的路径 | 通过 | `make perf`：约 380ms；改了 11 个文件、删了 1 个的分支，重读 11 个、删除 1 个、其余 1990 个不动 |
| M3 | 预览和最终渲染逐字节一致 | 通过 | `TestPreviewOfSavedContentIsByteIdenticalToTheBuiltPage` |
| M3 | 用 VS Code 改文件后，后台 3 秒内更新 | 通过 | `make perf`：约 200ms |
| M3 | 在旧版本上保存得到 409 和三方对比 | 通过 | `TestSavingAgainstAReplacedVersionIsRefusedWithWhatIsStored`；后台的冲突对话框 |
| M3 | 只改标题时，`git diff` 只有标题一行 | 通过 | `TestChangingTitleTouchesOnlyTitleLine`、`TestEditingTheTitleRewritesOnlyTheTitleLine` |
| M4 | 点「发布」后 5 分钟内在 GitHub Pages 上可见 | 未验证 | 没做过端到端验收 |
| M4 | commit 只包含这篇文章的文件，用户的其他改动原封不动 | 通过 | `TestPublishCommitsOnlyWhatItWasAskedTo`、`TestARefusedPlanLeavesTheRepositoryExactlyAsItWas` |
| M4 | 远端有新提交时被拦下，并给出选项 | 通过 | 能拦下并报告（`TestARemoteWithNewCommitsIsReportedBeforePublishing`）；没有重叠时可以一键接到远端之后推送（`TestARemoteThatMovedOnElsewhereIsOfferedARebase`），有重叠时给出远端的 diff（`TestAnOverlapIsShownAndNothingIsReplayed`） |
| M4 | 没有凭据时立刻报错，不会卡住 | 通过 | `TestAnUnreachableRemoteFailsQuicklyRatherThanHanging` |

## 4. v1.0 收尾清单

| # | 问题 | 现状与位置 | 优先级 |
|---|---|---|---|
| 1 | **定时文章会提前上线（bug）** | **已修复。** 构建只收录构建时刻已经公开的内容：`published_at` 不晚于构建时刻的 `published` 和 `scheduled`，以及没有日期的 `published`（`published` 也看日期是第 12 项定的）。规则和 `Content.IsPublic` 相同，过滤在 SQL 里完成（`content.Query.PublicAt`），首页、列表、分类、RSS、sitemap 和文章页一起生效。serve 记下下一篇定时文章的时间，到点后的第一个请求先重新规划页面再应答。`kite build --verify` 的两次构建共用同一时刻。测试：`TestPublicAtAgreesWithIsPublic`、`TestScheduledContentWaitsForItsTime`、`TestAScheduledPostIsServedOnceItsTimeComes` | P0 |
| 2 | **静态站点的定时发布没有东西去触发** | **已解决。** `kite init` 另外生成一个 `scheduled.yml`，每小时运行一次（`17 * * * *`，避开整点的排队高峰）。每次构建在报告里给出下一篇定时文章的时间（`kite build --json` 的 `next_due`），`deploy.yml` 把它存进 Actions 缓存；`scheduled.yml` 只在这个时间已过、或者找不到记录时才调用 `deploy.yml` 构建和部署，其余情况一个很短的检查 job 就结束。定时触发单独放一个文件，是因为公开仓库 60 天没有提交时，GitHub 会把带 `schedule` 的 workflow 整个关掉，push 触发也一起失效。测试：`TestBuildReportsWhenTheNextScheduledPostIsDue`、`TestTheScheduleNeverSitsInTheDeployWorkflow`、`TestTheScheduledWorkflowReadsTheDueTimeTheBuildReports`，生成的两个 workflow 都通过 actionlint。`kite init` 不会改写已有的 workflow，之前建的站点需要从新生成的项目里复制这两个文件 | P0 |
| 3 | 远端有新提交时只会拒绝 | **已解决。** push 因非快进被拒后先 fetch，再判断三件事：远端的新提交有没有改到这次发布的文件，这次发布是不是唯一没推送的提交，远端改过的文件在本地有没有未提交的改动。都没有问题时，后台显示「接在后面推送」（命令行是 `kite publish --push --rebase`），把提交接到远端之后再推送；做法见 [architecture.md §16.6](architecture.md#16-git-workflow最高危模块)，工作区里的其他改动不受影响。有重叠时展示远端那一侧的 diff，交还作者处理。另外新增 `POST /publish/push` 和 `kite publish --push`，推送失败后可以单独重试。测试：`internal/publish/git/remote_test.go` 的 6 个用例和 `TestAPushRefusedByAMovedRemoteCanBeReplayedOnIt` | P1 |
| 4 | 「已部署」这一步永远不会完成 | **已解决。** 远端在 github.com、并且部署到 `github-pages` 环境的仓库，「已部署」按推送的那个提交的部署状态显示：成功、失败或进行中，成功后给出站点链接；被后来的提交取代的部署也算上线。其他托管平台、私有仓库、从不部署到 Pages 的仓库显示「托管平台不回报」，不再一直等待。查询在后台进行，不阻塞面板；匿名调用每小时只有 60 次，所以进行中的部署会逐步拉长查询间隔，上线后不再查询，剩余次数不到 10 次时暂停。测试：`internal/publish/git/deploy_internal_test.go`、`TestAPushToGitHubPagesIsReportedDeployedWhenItIs`、`TestAnyOtherHostLeavesDeploymentNotApplicable`，并对真实的 GitHub API 核对过一次。[architecture.md §17](architecture.md#17-cicd) 原先写的是 V1「不做任何 API 集成」，已改为「不做需要凭据的 API 集成」 | P1 |
| 5 | 性能没有验收 | **已解决。** `make perf`（`internal/perf`）生成 2000 篇的站点，用真实的服务器、文件监听和热重载通道测量 §3 的五项时延，达不到目标就失败，结果见 §3。第一次测量时构建 3.7 秒、改文件到页面 564ms，都没有达标，为此做了四处优化：全量构建按目标并行渲染（输出与顺序无关，observer 仍按计划顺序收到页面）；规划时一次批量取出所有条目（`Reader.GetMany`），不再每篇查两次；分类和标签的列表直接从已经取出的条目分组，不再每个标签查一次；建索引时每个文件的语句只准备一次。没有放进 CI：GitHub 的机器比作者的电脑慢，速度也不稳定，发布前在本机跑 | P1 |
| 6 | `kite theme verify` 命令没有实现 | **已解决。** `kite theme verify [dir]` 用内置的 fixture 站点（`internal/themecheck`：七篇文章分三页、一个页面、带图片的 page bundle、标签和分类及其 term 页、404，还有不该出现的草稿和定时文章）以 build 和 serve 各渲染一遍，逐字节比较 build 写出的全部 26 个文件（23 个页面，加上 RSS、sitemap 和 page bundle 里的图片），指出每个文件第一处不同的行。不给目录时检查当前项目的主题，项目之外检查内置主题。还没做的：fixture 没有覆盖多语言（i18n 在 M5 才定）；`--strict` 要等引擎有了废弃告警再加。测试：`internal/themecheck/themecheck_test.go` | P2 |
| 7 | 只支持 YAML front matter | **已解决。** `internal/frontmatter` 支持 Hugo 写在 `+++` 之间的 TOML，保真和 YAML 一样：没改动的文件逐字节不变；只改标题时只动标题那一行，值后面的注释保留；新 key 写在第一个表之前，否则会落进那个表里；多行字符串和多行数组整体替换；日期保持原来的写法（带不带时区）；改动过的表移到末尾重写，其余的表原样不动。值由 `github.com/pelletier/go-toml/v2` 解析，这是新增的依赖；哪一段属于哪个 key 由这里自己判断，遇到判断不了的写法时仍能读取，只是拒绝改写。测试：`internal/frontmatter/toml_test.go`、`TestATOMLPostReadsLikeItsYAMLTwin`、`TestEditingATOMLTitleRewritesOnlyTheTitleLine` | P2 |
| 8 | 主题的 `requires` 只读取、不检查 | **已解决。** 每次加载主题（打开站点、构建、serve）都检查 `requires`：支持 `>=`、`>`、`<=`、`<`、`=`，空格隔开表示同时满足，`||` 表示任一满足，按 semver 比较，预发布版本排在正式版之前。不满足就拒绝加载，并说明主题要求的范围和正在运行的 Kite 版本。从源码构建的版本（`dev` 或提交哈希）不参与比较；`git describe` 生成的「tag 之后又有提交」按那个 tag 比较。测试：`internal/render/theme/requires_test.go` | P2 |
| 9 | 文章列表不显示「置顶」 | **已解决。** 列表摘要带上 `pinned`，在 SQL 里从存储的 meta 取出（`json_type(meta_json, '$.pinned') = 'true'`），列表仍然不需要逐行解析 meta；只有真正的 `true` 才算置顶，和条目本身的判断一致。文章列表在标题后按设计稿画出琥珀色的「置顶」标记，深色模式有对应的配色。测试：`TestSummariesSayWhichItemsArePinned` | P2 |
| 10 | 发布的备用路径没有实现 | **已解决。** 按 [architecture.md §16.3](architecture.md#16-git-workflow最高危模块) 的逃生舱实现：在临时 index 里从 HEAD 出发 `git add` 这次发布的路径（clean filter 和 LFS 照常生效），`commit-tree` 生成提交，`update-ref` 以旧值做 CAS 移动分支，最后只更新真实 index 里这几个路径，失败时把分支移回原处。用在 hook 拒绝发布之后：拒绝会报成 `hook_refused` 并带上 hook 的输出，后台提供「跳过 hooks 发布」，命令行是 `kite publish --no-verify`。测试：`internal/publish/git/escape_test.go`、`TestAPublishAHookRefusedCanGoAheadWithoutTheHooks` | P2 |
| 11 | 周边仓库和文档站没有建 | 组织下现在有 `kite`、`.github`，以及另一个产品 Explore（内容发现平台）用的 `explore`。官网文档站 `website`（用 Kite 自己搭）、`starters`、`setup-kite`，以及存放预研和设计存档的 `lab` 都还没有 | P2 |
| 12 | Hugo 里日期在未来的文章会立即公开 | **已解决。** 日期在未来的 `published` 和 `scheduled` 一样，等到发布时间才公开，和 Hugo、Jekyll 的默认做法一致；没有日期的 `published` 仍然立即公开。Kite 把没有 `status` 的文件读成 `published`、把 Hugo 的 `date` 读成发布时间，所以从 Hugo 迁过来的站点，排在未来的文章会按时上线：serve 到点后的第一个请求重新规划页面，静态站点由构建报告的 `next_due` 和 `scheduled.yml` 在到点后一小时内补上。规则在 `Content.IsPublic` 和 SQL 过滤里各有一份，`TestPublicAtAgreesWithIsPublic` 保证两者一致。测试：`TestAPublishedPostDatedLaterWaitsForItsDate`、`TestAScheduledPostIsServedOnceItsTimeComes/published` | P2 |
| 13 | serve 没有 RSS 和 sitemap | **已解决。** 核对第 12 项时发现：`kite serve` 对 `/rss.xml` 和 `/sitemap.xml` 都返回 404，而每个页面都链接着 RSS。两者由 build 结束时的 completion hook 生成，serve 从来不调用这些 hook；build 和 serve 的比对只看 `.html`，所以一直没被发现。现在 serve 在第一次有请求用到时，用和 build 相同的代码算出每页交给 hook 的信息（不画模板，2000 篇的站点约 120ms，完整构建约 1 秒），跑同一组 hook，结果留在内存里，直到下一次重新规划：文件改动和定时文章到点都会让它重算。同名时 hook 的产物优先于静态文件，和 build 后写覆盖先写一致。测试：`TestServedFilesAreByteIdenticalToBuiltOnes`（改为比较 build 写出的每个文件）、`TestTheFeedFollowsTheSite`、`TestExtrasAreWhatABuildWrites`；`kite theme verify` 也改为比较全部文件 | P2 |

## 5. 分阶段计划

里程碑编号沿用 [architecture.md §28](architecture.md#28-roadmap)。其中「v1.5 媒体库」是本文**新增**的阶段，原规划只写了完整媒体库要等 V1.5 或 V3（architecture.md §20.2）。

| 阶段 | 版本 | 范围 | 已有基础 |
|---|---|---|---|
| **1. v1.0 收尾** | v1.0 | §4 的 P0 和 P1 项；另外可以顺手做两个低成本占位：「存储空间」（统计内容目录大小）、版本号旁的「最新」（查询 GitHub Releases）；最后打 `v1.0.0` 标签 | — |
| **2. M5 主题契约** | v1.1 | 写第二套风格完全不同的主题，并用 `kite theme verify`（已有）检查它；`kite theme list/add/new`；检查 `requires`；菜单（`.Site.Menus`）写入契约；冻结之前要把 [theme-system.md](theme-system.md) §12 i18n 的三件事（尤其是多语言 URL 策略）和 §14 的开放问题定下来；发布 `kite/v1`，建 themes 仓库和主题开发文档 | 查找顺序、带方法的 RenderContext、命名空间函数、`apiVersion` 校验、由 settings schema 生成的配置页都已经有了 |
| **3. M6 可重现构建** | v1.2 | `kite.lock`、`kitew`、Cloudflare Pages 部署模板；启用增量构建里的跳过判断 | 依赖记录和缓存键已经有了 |
| **4. 媒体库**（新增） | v1.5 | 在现有索引上汇总所有 page bundle 里的文件：媒体列表、跨文章复用、找出没人引用的文件、上传入口 | 单篇的附件上传和删除 API 已经有了 |
| **5. M7 动态模式** | v2.0 | SQLite 作为真相源，写入同一套读模型；`kite migrate` 在文件和数据库之间互转；多用户和角色；私密文章；数据库备份和 `kite export`；Kite 自己存储的评论 | 读模型、单账号认证、Docker 都已经有了 |
| **6. M8 插件** | v3.0 | 基于 wazero 的 WebAssembly 插件、Host ABI、能力和权限声明、`kite plugin`、插件 SDK | HookBus 已经被内置功能使用 |

## 6. 后台占位功能的归属

后台按设计稿画出了 Kite 暂时还没有的功能，这些地方显示为灰色，悬停时提示「即将推出」，不显示编造的数字（`web/src/components/Soon.tsx`）。它们的归属如下：

| 占位功能 | 做法 | 阶段 |
|---|---|---|
| 菜单 | 在 `kite.yaml` 里定义菜单，并写进主题契约 | M5 |
| 附件、上传附件 | 全站媒体库 | v1.5 |
| 评论、待审核评论、最近评论 | 静态模式可以先接 Giscus 或 Waline（前端评论组件，对应 architecture.md §13 的 client 能力）；由 Kite 自己存储评论要等动态模式或插件 | 先接组件；自存放到 M7 或 M8 |
| 用户、可见性、备份、上次备份 | 多用户、私密文章、数据库备份都依赖动态模式 | M7 |
| 插件、插件可更新 | WebAssembly 插件 | M8 |
| 存储空间 | 统计内容目录和上传文件的大小 | 阶段 1 可做 |
| 版本号旁的「最新」 | 查询 GitHub Releases | 阶段 1 可做 |
| 总访问量、访问量列、RSS 订阅数 | 纯静态站点自己无法统计。要么接第三方统计服务（如 Umami、Plausible），要么等动态模式按请求统计；RSS 订阅数基本拿不到 | 待定 |
| 文章列表里的「置顶」 | 列表接口返回 `pinned` | 已完成（§4 第 9 项） |

## 7. 待定事项 `[待定]`

1. **静态站点的定时发布怎么触发**：已定。每小时检查一次，下一篇的时间由构建算出，只在有文章到点时才构建和部署（§4 第 2 项）。私有仓库每小时的检查仍按 1 分钟计费，大约每月 720 分钟。
2. **评论**：先接 Giscus 或 Waline 这类现成组件，还是等 Kite 自己实现？
3. **访问统计**：接哪一家第三方服务，还是先不做？
4. **v1.0 标签的时机**：§4 的 P0 和 P1 已经修完。打标签前建议再做一次 §3 里 M4 的端到端验收：在真实仓库里发布一篇，看 5 分钟内是否出现在 GitHub Pages 上。

## 8. 维护本文

- 每完成一项，就更新 §2–§4 的状态和开头的核对日期；新增或推迟的事项要写明原因。
- 判断一项是否完成，以代码和测试为准，不以计划为准。
- 如果和 architecture.md 的里程碑定义冲突，先改 architecture.md，再改本文。
