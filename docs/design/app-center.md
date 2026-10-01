# Kite 应用中心设计

> 状态：设计中 · 最近更新：2026-10-01 · 跟踪：[kite-plus/kite#16](https://github.com/kite-plus/kite/issues/16)
> 上级文档：[architecture.md](architecture.md) · 相关：[theme-system.md](theme-system.md) §8.4、§11.3，[plugin-system.md](plugin-system.md) §10、§15、§17，[architecture.md](architecture.md) §24（`kite.lock`）、§32

---

## 0. 结论

- **索引是一份静态 JSON，不是一个服务。** 在一个新仓库 `kite-plus/apps` 里，每个主题和插件只写一份很短的 YAML（是什么、在哪个仓库）；其余的一切由 CI 从各仓库的 GitHub Release 和包里的 `theme.yaml` / `plugin.yaml` 生成：版本、需要的 Kite 版本、sha256、截图、说明、会从哪些网站加载东西。Kite 的后台和命令行只读这份 JSON（§3、§4.1）。
- **装好的主题和插件仍然在站点仓库里，`kite.lock` 只记它们从哪来。** 构建不联网，也不依赖应用中心是否在线；lock 里的来源和摘要用来提示更新、发现手改和防止包被换（§4.2）。
- **安装走现有的路径。** 下载 zip，核对 sha256，然后和上传 zip、`kite theme add` 一样：解包限制、按站点加载的标准检查、`PutTheme` / `PutPlugin` 的 ChangeSet，再由发布器提交（§2、§4.7）。
- **第一版就收第三方的包。** 作者向 `kite-plus/apps` 提一个 PR 上架，CI 检查后由维护者审核；之后的新版本 CI 自动收录，权限变大的才回到人工审核。界面标出「官方」和「社区」。签名放在后面的阶段（§4.3、§7）。
- **包经 jsDelivr 分发，GitHub Release 作备用。** 索引仓库里存一份每个版本的 zip，因此收录的包必须用允许再分发的开源许可（§4.3、§4.4）。
- 2026-10-01 定下的四件事和还开着的问题见 §8。

## 1. 目标与非目标

### 目标

1. 在后台浏览、安装、更新、移除主题和插件，不用先去 GitHub 下载 zip 再上传。
2. 命令行同样能按名字装：`kite theme add vane`、`kite plugin add search`。
3. 告诉站点哪些装了的东西有新版本，哪些版本和正在跑的 Kite 合得来。
4. 装进来的东西可以验证：从哪来、有没有被换过、有没有被手改过、插件要往页面上加什么、会从哪些网站加载。
5. 构建不依赖应用中心：主题和插件随站点仓库提交，断网、应用中心下线都照常构建。

### 非目标（第一版）

- 付费、评分、评论、下载量统计。
- 在线编辑主题（[theme-system.md §11.3](theme-system.md#113-明确不做的东西)）。
- 构建时按 lock 下载（§4.2 选的是装进仓库）。
- 用 Kite Plus 账号提交、签名：后面的阶段（§7）。
- 网页版的应用目录：后面再定放在官网还是 Explore（§6）。

## 2. 现状

| 能力 | 现在怎么做 |
|---|---|
| 安装主题 | 后台「设置 → 主题」上传 zip（`POST /api/v1/themes`）；`kite theme add <zip\|目录>`（0.1.4） |
| 安装插件 | 后台「系统 → 插件」上传 zip（`POST /api/v1/plugins`）；`kite plugin add <zip\|目录>` |
| 检查 | `archive.Unpack` 限制压缩包大小、解开后的大小和文件数，拒绝链接和跳出目录的路径；`theme.Load` / `plugin.Load` 按站点加载的标准检查 `apiVersion`、`requires`、布局、语言包、插件模块 |
| 写入 | ChangeSet 里的 `PutTheme` / `PutPlugin`，同名的要确认后才替换；发布器像对待文章一样暂存和提交 |
| 插件安装前的说明 | 注入几段代码、从哪些网站加载（`loaded.Hosts()`）、导出哪些钩子 |
| 官方的包 | 主题：风标（`kite-plus/theme-vane`）、年鉴（`kite-plus/theme-almanac`）；插件：统计、评论、公式与图表、搜索（`kite-plus/plugin-*`）。release 的附件统一是 `<名字>-<版本>.zip`，tag 是 `v<版本>` |

设计里原有的约定：市场和注册表在 V4，v1 用 Git 地址加 checksum 安装（[theme-system.md §11.3](theme-system.md#113-明确不做的东西)、[plugin-system.md §17](plugin-system.md#17-明确不做的东西)）；`kite.lock` 记版本、来源和 checksum（[architecture.md §24](architecture.md#24-kitelockm6)，M6）；插件权限在安装时展示、钉进 lock，更新后扩权要重新确认（[plugin-system.md §10.2](plugin-system.md#102-强制点)）。应用中心把这些提前，并且在第一点上改了做法：不是 Git 地址，是一份索引。

## 3. 总体结构

```
主题、插件各自的仓库 ──release（<名字>-<版本>.zip）──┐
                                                    ▼
kite-plus/apps：每个包一份 YAML ──CI──▶ 下载每个版本、核对 sha256、verify、
                                       找出外部网站 ──▶ index.json
                                                         │ HTTPS / 镜像
                                                         ▼
                                Kite 服务端（后台的 API）和命令行
                                  │ 读索引（缓存）、按 requires 过滤
                                  │ 下载 zip、核对 sha256
                                  ▼
                 现有的安装路径：Unpack → Load → PutTheme / PutPlugin + kite.lock
                                  ▼
                       站点仓库（随发布提交，构建不联网）
```

浏览器不直接访问索引和下载地址，一律由 Kite 服务端去取：缓存和校验只有一处，后台的 CSP 不用放宽，内网和离线时也只需要服务端能联网。

## 4. 方案

### 4.1 索引：静态 JSON

三种做法：

| 做法 | 问题 |
|---|---|
| 运行时搜 GitHub（按 topic） | 匿名调用每小时 60 次、搜索每分钟 10 次；拿不到校验过的 sha256 和兼容范围；GitHub 搜索的结果不归我们管 |
| 一个服务（像 Explore 那样有后端、账号和审核后台） | 上架和审核用 GitHub 的 PR 就够了；多一个要运维的服务和一套账号 |
| **静态索引仓库 + CI 生成 JSON** | 只多一个仓库和一个 workflow；索引可以放在任何 CDN 上；上架和审核走 PR，以后改用账号提交，输出仍然是同一份 JSON |

选第三种。`kite-plus/apps` 里每个包一份 YAML，人只写不能推导出来的东西：

```yaml
# themes/vane.yaml
kind: theme
id: vane                         # 主题的 name；插件的 id
repo: kite-plus/theme-vane
official: true
```

第三方的条目同样只有这几行，`official` 写 `false`（只有维护者能写 `true`）。CI 每小时跑一次，官方仓库在发 release 时也会立即触发它，把条目展开成 `index.json`：

```json
{
  "format": 1,
  "generated": "2026-10-01T08:00:00Z",
  "apps": [
    {
      "kind": "theme",
      "id": "vane",
      "official": true,
      "repo": "kite-plus/theme-vane",
      "title": {"en": "Vane", "zh-CN": "风标"},
      "description": {"en": "…", "zh-CN": "…"},
      "author": {"name": "Kite", "url": "https://github.com/kite-plus"},
      "license": "MIT",
      "homepage": "https://github.com/kite-plus/theme-vane",
      "tags": ["docs", "dark mode"],
      "screenshot": "https://…/vane/1.0.1/screenshot.webp",
      "versions": [
        {
          "version": "1.0.1",
          "published": "2026-10-01T07:05:00Z",
          "notes": "https://github.com/kite-plus/theme-vane/releases/tag/v1.0.1",
          "api": "kite/v1",
          "requires": ">=0.1.4 <2.0.0",
          "archive": {
            "urls": ["https://…/vane-1.0.1.zip", "https://github.com/kite-plus/theme-vane/releases/download/v1.0.1/vane-1.0.1.zip"],
            "sha256": "da82c093…",
            "size": 108040
          },
          "loads": []
        }
      ]
    }
  ]
}
```

插件的版本另有 `abi`（插件契约）、`inject`（往页面注入几段代码）、`hooks`（导出的构建期钩子）；`loads` 是包会让读者的浏览器去访问的外部网站，插件取自 `plugin.yaml` 的声明，主题由 CI 用 `kite theme verify` 的夹具站点构建一次，扫出 `script`、`link`、`img`、`iframe` 指向的外部域名。

规则：

- `id` 就是装进站点后的目录名：主题的 `name`、插件的 `id`。主题和插件各是一个命名空间。
- 一个版本一经收录就不再变：同一个版本号的 zip 换了内容，CI 拒绝，而不是悄悄更新 sha256。
- `title`、`description` 的中文取自包自己的语言包（`theme.title`、`theme.description`，插件同理），没有就只有英文。
- `format` 是索引格式的大版本。Kite 遇到不认识的大版本时只浏览、不安装，并提示升级 Kite。

### 4.2 装进仓库，`kite.lock` 只记来源

`kite.lock` 原来的设计（[architecture.md §24](architecture.md#24-kitelockm6)）是 lock 里记版本，构建时按 lock 下载。插件第一版已经改成文件装进站点仓库、随仓库提交（[plugin-system.md §0.1](plugin-system.md#01-第一版实施方案2026-09-26-定) 第 3 条），官网也一直这样用。应用中心沿用后者：

- 构建不联网，CI 不多一步，应用中心下线也不影响任何站点。
- 站点仓库就是站点的全部，换一台机器 `git clone` 就能构建。
- 和上传 zip、`kite theme add` 是同一条写入路径，三种装法结果一样。

lock 只记来源，和主题、插件文件在同一个 ChangeSet 里写入、一起提交：

```yaml
# kite.lock
lockfileVersion: 1
themes:
  vane:
    version: 1.0.1
    source: https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json   # 从哪份索引装的
    resolved: https://…/vane-1.0.1.zip
    checksum: sha256-da82c093…                    # 下载的 zip
    tree: sha256-…                                # 装好后目录里每个文件的摘要
plugins:
  search:
    version: 0.1.0
    source: https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json
    resolved: https://…/search-0.1.0.zip
    checksum: sha256-…
    tree: sha256-…
    granted:                                      # 安装时确认过的
      inject: 2
      loads: []
      hooks: [build_complete]
```

用法：

- **提示更新**：lock 里有记录的，按 `source` 那份索引找新版本；lock 里没有的（上传 zip 或手放的），只有当包自己的 `homepage` 和索引里的 `repo` 对得上时才提示，免得把同名的别人的主题当成官方的。
- **发现手改**：`tree` 是目录里每个文件（路径加 sha256）排序后的摘要。对不上说明有人改过，更新前提醒“会覆盖你的修改”。`kite doctor` 也报告这一项。
- **防扩权**：插件更新后的 `loads`、`hooks` 或 `inject` 比 `granted` 多时，要重新确认才装。
- `kite:` 版本锁和 `kitew` 仍在 M6；这里只先做 lock 的 `themes`、`plugins` 两节，格式按 M6 的写法留好位置。

### 4.3 上架、审核与信任

第一版就开放第三方提交（2026-10-01 定）。

**怎么上架**：作者向 `kite-plus/apps` 提一个 PR，加一份 §4.1 那样的 YAML。仓库要满足：

- 公开；发 release 时附上 `<id>-<版本>.zip`，tag 是 `v<版本>`。zip 里只有主题或插件本身，`kite theme pack` / `kite plugin pack` 打出来的就是这样（A2 加上这两个命令，打包规则和官方仓库的 `scripts/package.sh` 一致）。
- `id` 等于包里 `theme.yaml` 的 `name` 或 `plugin.yaml` 的 `id`，在索引里还没人用；`default` 和官方包的名字是保留的。
- 用允许再分发的开源许可（MIT、Apache-2.0、GPL 等），仓库里有 LICENSE，`theme.yaml` / `plugin.yaml` 写明 `license`。索引仓库要存一份 zip 副本经 jsDelivr 分发（§4.4），没有这个许可就不能这样做。

**CI 对每个版本都做的检查**：下载并核对大小和 sha256；`kite theme verify` 或 `kite plugin verify`；`requires` 至少能被一个已发布的 Kite 满足，`api`、`abi` 是已知的版本；zip 里只有该有的文件；扫出 `loads`（主题用 `kite theme verify` 的夹具站点构建一次，看 `script`、`link`、`img`、`iframe` 指向哪些外部域名；插件取 `plugin.yaml` 的声明）。同一个版本号的 zip 换了内容，直接拒绝。

**人工审核什么时候需要**：

| 情况 | 怎么收 |
|---|---|
| 新条目 | CI 通过后，维护者审核：仓库和 PR 作者是同一个人或组织，名字不冒充别人，介绍和截图属实，`loads` 里的网站说得过去 |
| 已收录的包发了新版本 | CI 通过、而且 `loads`、`hooks`、`inject` 没有变多，就自动收录 |
| 新版本的权限变多了 | 生成一个待审 PR，维护者同意后才收录；同意前索引里的最新版本停在旧的 |
| 发现问题的版本 | 在条目里标 `yanked: [版本]`；已经装了它的站点在后台和 `kite apps outdated` 里看到提醒，不能再新装它 |
| 要下架的包 | 条目标 `delisted` 并写明原因；列表不再显示，已装的站点同样收到提醒 |

**界面上怎么区分**：官方包标「官方」，其余标「社区」。安装社区包前多一句提醒：由第三方作者维护，Kite 检查过它能正常加载、会从哪些网站加载东西，但不对内容负责。

**安装前展示**：主题和插件都写出会从哪些网站加载东西；插件另外写出注入几段代码、导出哪些钩子。插件本来就跑在没有文件、网络和环境变量的沙箱里（[plugin-system.md §10.3](plugin-system.md#103-默认不开放完整-wasi-已冻结)），读者浏览器里会加载什么，才是要让站长看清楚的部分。

**签名**放在 A5：索引和包用 minisign 或 Sigstore 签名，公钥内置在 Kite，对不上不安装；同一阶段，作者可以用 Kite Plus 账号（`id.kite.plus`）提交，不必会用 Git。

### 4.4 分发和镜像

包都不大：主题 100–200 KB，只注入代码的插件十几 KB，带 WASM 模块的插件（搜索、公式与图表）1 MB 上下。难点在境内访问：GitHub Release 的下载走 `objects.githubusercontent.com`，在境内慢且不稳定。

定下的做法（2026-10-01）：**索引仓库存一份 zip 副本，经 jsDelivr 分发，GitHub Release 作备用。**

- jsDelivr 的 GitHub 通道有两个上限：一个仓库在某个版本下的整体快照不能超过 50 MB，单个文件不能超过 20 MB `[EV]`。所以 zip 不能都堆在 main 分支上：带 WASM 的插件一个版本就约 1 MB，几十个版本之后整个仓库就超限了。
- 做法是**每个版本一个 tag**：CI 收录一个版本时，在 `kite-plus/apps` 里建一个只含这个 zip 的孤立提交，打上 tag `<kind>-<id>-<版本>`（如 `theme-vane-1.0.1`）。包的地址是 `https://cdn.jsdelivr.net/gh/kite-plus/apps@theme-vane-1.0.1/vane-1.0.1.zip`：每个快照只有一个文件，永远不会碰到上限；tag 不再移动，内容永远不变，可以长期缓存。main 分支只放条目的 YAML 和 `index.json`。
- 收录的 zip 不能超过 20 MB（jsDelivr 单个文件的上限）；现有的主题和插件都远在这之下。
- 索引的地址是 `https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json`，CI 更新后调 jsDelivr 的 purge 刷新这个分支地址的缓存；备用是 `https://raw.githubusercontent.com/kite-plus/apps/main/index.json`。以后要换成 `apps.kite.plus`，只需要在 Kite 内置的地址列表前面加一个。
- 境内访问先实测。jsDelivr 不够稳时，两种补法（§8）：在香港加一个镜像，和 Kite Plus 的服务放在一起；或者把每个版本另发成一个 npm 包，借 npmmirror 在境内的 CDN 分发。

客户端的做法和地址无关：按索引里列出的地址依次尝试，sha256 对上才用，所以镜像不需要被信任。`KITE_APPS_URL` 环境变量（或 `kite.yaml` 的 `apps.index`）可以换成自建的索引，给内网和离线环境用。索引缓存在 `.kite/cache/apps/`，带 ETag，一小时内不重复取；断网时用缓存，并说明是多久以前的。

### 4.5 后台

一个入口（2026-10-01 定）：「系统 → 应用中心」，主题和插件两个标签。已安装的「设置 → 主题」和「系统 → 插件」显示「可更新」并链接过来。发现新东西的地方只有一个，主题和插件的检查、安装、更新提示也是一套界面。

页面：

- **列表**：搜索；按类型、标签、「官方」「社区」筛选；默认只显示和正在跑的 Kite 合得来的版本；已装的标出版本，有新版的标「可更新」，装了被撤回版本的标出来。
- **详情**：截图、介绍、作者、许可、各版本的更新说明、需要的 Kite 版本；会从哪些网站加载；插件注入几段代码、导出哪些钩子。
- **安装**：服务端下载、校验、写入，和上传 zip 共用确认和错误提示；主题装好后可以直接进整站预览，不用先切换。
- **更新**：先说明会改什么（版本、更新说明、外链和钩子的变化、有没有手改会被覆盖），确认后替换，设置保留。
- **移除**：现有的删除，同时去掉 lock 里那一项。

只读部署（没带 `--write` 的 `kite serve`）只能浏览，安装和更新的入口不出现，和其他写操作一致。

### 4.6 命令行

```bash
kite theme add vane              # 按名字装最新的、合得来的版本
kite theme add vane@1.0.1        # 指定版本
kite plugin add search
kite apps search 文档            # 主题和插件一起搜
kite apps outdated               # 列出有新版本的
kite apps update [名字]          # 更新；插件扩权时要交互确认，或加 --yes
```

`kite theme add <zip|目录>` 和 `kite plugin add <zip|目录>` 照旧。一个参数既不是文件也不是目录时才按名字到索引里找。

### 4.7 API

```
GET  /api/v1/apps?kind=theme|plugin&q=…        列表，标出已装、可更新、合不来
GET  /api/v1/apps/{kind}/{id}                  详情，含各版本
POST /api/v1/apps/{kind}/{id}/install          {"version": "1.0.1"}；同名已装时要 replace=true
POST /api/v1/apps/{kind}/{id}/update           {"version": "…", "confirm": {...}}
```

安装和更新最后都是一个 ChangeSet：`PutTheme` 或 `PutPlugin`，加上写 `kite.lock` 的一步，任何一步失败都不写。下载的 zip 先存在 `.kite/cache/apps/archives/`，按 sha256 命名，同一个包不重复下载。

## 5. 安全

| 威胁 | 缓解 |
|---|---|
| 下载的包被替换 | 索引里的 sha256，对不上不装；lock 记下装的是哪个 |
| 索引本身被篡改 | HTTPS；签名阶段起验证索引的签名 |
| 压缩包炸弹、路径穿越、链接 | 现有的 `archive.Unpack` 限制 |
| 插件更新后静默扩权 | `granted` 钉在 lock 里，更新后的 `loads`、`hooks`、`inject` 变多要重新确认（[plugin-system.md §10.2](plugin-system.md#102-强制点)） |
| 主题带第三方脚本 | 安装前展示 `loads`；CI 扫出来，不靠作者自己申报 |
| 同名冒充官方包 | `id` 先到先得，官方名字保留；新条目人工审核；更新只认 lock 里的来源，或 `homepage` 与 `repo` 对上的；界面标出「官方」「社区」 |
| 社区包的新版本夹带新的外部脚本 | `loads`、`hooks`、`inject` 变多的版本回到人工审核；已装的站点更新时还要再确认一次 |
| 有问题的版本已经被装了 | `yanked` 和 `delisted`，装了的站点收到提醒 |
| 手改的主题被更新覆盖 | `tree` 摘要对不上时先提醒 |

## 6. 和其他计划的关系

- **M6 `kite.lock`**：先做 lock 的 `themes`、`plugins` 两节；`kite:` 版本锁、`kitew` 和 Cloudflare Pages 仍在 M6。
- **Kite Plus 账号**：第一版用 PR 上架；A5 起作者也可以用 `id.kite.plus` 登录提交（[Explore 的 identity-and-comments.md](https://github.com/kite-plus/explore/blob/main/docs/design/identity-and-comments.md)）。
- **网页版目录**：索引是公开的，以后可以由它生成官网上的一组页面，或者放进 Explore；第一版不做。
- **原定 V4 的市场**（[architecture.md §32](architecture.md#32-现在不要设计的东西)）：应用中心是它的第一步，付费和评分仍然不做。

## 7. 分阶段

| 阶段 | 内容 | 验收 |
|---|---|---|
| **A1** 已完成 | `kite theme list/add/remove/use/new`（0.1.4），主题和插件共用一个读压缩包和目录的入口 | — |
| **A2** 索引和上架 | `kite-plus/apps` 仓库：条目格式、生成并检查 `index.json` 的 CI、每个版本一个 tag 的 zip 副本和 jsDelivr 刷新、提交说明和 PR 模板、审核清单；`kite theme pack` / `kite plugin pack`；收录 6 个官方包 | 新 release 后 CI 自动更新索引；每个版本都通过 §4.3 的检查；同一版本换内容被拒绝；权限变多的版本生成待审 PR；按说明能从零上架一个测试主题 |
| **A3** Kite 客户端和命令行 | `internal/apps`（读索引、缓存、按 `requires` 过滤、下载、校验）、`kite.lock` 的 `themes` / `plugins`、按名字安装和更新、`kite apps`、`kite doctor` 检查 lock | 新站点 `kite theme add vane` 装上、能构建、lock 有记录；手改过的主题更新前提醒；断网时用缓存 |
| **A4** 后台应用中心 | §4.5 的页面和 §4.7 的 API，「官方」「社区」标记，撤回和下架的提醒 | 浏览、详情、安装、更新、移除在真实页面可用；浏览器回归测试覆盖安装和更新 |
| **A5** 签名和账号提交 | 索引和包的签名；用 Kite Plus 账号提交 | — |

## 8. 决定与待定事项

2026-10-01 定下：

1. **收录范围**：第一版就开放第三方提交（§4.3）。
2. **分发**：索引仓库存 zip 副本，经 jsDelivr 分发，GitHub Release 作备用（§4.4）。
3. **lock**：装进站点仓库，`kite.lock` 只记来源（§4.2）。
4. **后台入口**：「系统 → 应用中心」一个入口（§4.5）。

还开着的 `[待定]`：

1. **境内访问**：A2 上线后实测 jsDelivr 在境内的成功率和速度，不够时加香港镜像，或另发 npm 包借 npmmirror 分发。
2. **审核的人和时限**：谁审、多久内回复；审核清单随 A2 写进 `kite-plus/apps` 的说明。
3. **索引的正式域名**：先用 jsDelivr 和 raw.githubusercontent.com 的地址；要不要、什么时候换成 `apps.kite.plus`。

## 证据来源

- jsDelivr GitHub 通道的上限：整体快照超过 50 MB 时拒绝服务（[jsdelivr/jsdelivr#18294](https://github.com/jsdelivr/jsdelivr/issues/18294)、[openlayers/openlayers.github.io#113](https://github.com/openlayers/openlayers.github.io/issues/113)），单个文件 20 MB（[jsdelivr/jsdelivr#18268](https://github.com/jsdelivr/jsdelivr/issues/18268)）。
- GitHub API 的匿名调用每小时 60 次，搜索每分钟 10 次：GitHub REST API 文档的 rate limits 一节。
