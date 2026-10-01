# Kite 应用中心设计

> 状态：设计中 · 最近更新：2026-10-01 · 跟踪：[kite-plus/kite#16](https://github.com/kite-plus/kite/issues/16)
> 上级文档：[architecture.md](architecture.md) · 相关：[theme-system.md](theme-system.md) §8.4、§11.3，[plugin-system.md](plugin-system.md) §10、§15、§17，[architecture.md](architecture.md) §24（`kite.lock`）、§32

---

## 0. 结论

- **索引是一份静态 JSON，不是一个服务。** 在一个新仓库 `kite-plus/apps` 里，每个主题和插件只写一份很短的 YAML（是什么、在哪个仓库）；其余的一切由 CI 从各仓库的 GitHub Release 和包里的 `theme.yaml` / `plugin.yaml` 生成：版本、需要的 Kite 版本、sha256、截图、说明、会从哪些网站加载东西。Kite 的后台和命令行只读这份 JSON（§3、§4.1）。
- **装好的主题和插件仍然在站点仓库里，`kite.lock` 只记它们从哪来。** 构建不联网，也不依赖应用中心是否在线；lock 里的来源和摘要用来提示更新、发现手改和防止包被换（§4.2）。
- **安装走现有的路径。** 下载 zip，核对 sha256，然后和上传 zip、`kite theme add` 一样：解包限制、按站点加载的标准检查、`PutTheme` / `PutPlugin` 的 ChangeSet，再由发布器提交（§2、§4.7）。
- **第一版就收第三方的包。** 作者向 `kite-plus/apps` 提一个 PR 上架，CI 检查后由维护者审核；之后的新版本 CI 自动收录，权限变大的才回到人工审核。界面标出「官方」和「社区」。索引用 minisign 签名，Kite 不用没有签名的索引（§4.3）。
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
- 用 Kite Plus 账号提交：后面的阶段（§7）。
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

第三方的条目同样只有这几行，`official` 写 `false`（只有 `kite-plus` 的包能写 `true`）。CI 每小时跑一次，条目有改动时立即跑，也可以在 Actions 里手动运行，把条目展开成 `index.json`。官方仓库发 release 时立即触发它要一个能跨仓库调用的令牌，第一版不做，新版本最多晚一小时收录。格式的正式说明在 [`kite-plus/apps` 的 docs/index-format.md](https://github.com/kite-plus/apps/blob/main/docs/index-format.md)：

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
      "license": "Apache-2.0",
      "homepage": "https://github.com/kite-plus/theme-vane",
      "tags": ["docs", "dark mode"],
      "screenshot": "https://cdn.jsdelivr.net/gh/kite-plus/apps@theme-vane-1.0.1/screenshot.webp",
      "versions": [
        {
          "version": "1.0.1",
          "published": "2026-10-01T07:09:06Z",
          "notes": "https://github.com/kite-plus/theme-vane/releases/tag/v1.0.1",
          "api": "kite/v1",
          "requires": ">=0.1.4 <2.0.0",
          "archive": {
            "urls": [
              "https://cdn.jsdelivr.net/gh/kite-plus/apps@theme-vane-1.0.1/vane-1.0.1.zip",
              "https://github.com/kite-plus/theme-vane/releases/download/v1.0.1/vane-1.0.1.zip"
            ],
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

插件版本的 `api` 是 `kite/plugin/v1`（模块的 ABI 由 `kite plugin verify` 检查），另有 `inject`（往页面注入几段代码）、`hooks`（`plugin.yaml` 声明的构建期钩子）。`loads` 是包会让读者的浏览器去加载的外部网站，由 `kite theme verify --json` / `kite plugin verify --json` 报告：主题用夹具站点构建一次，扫出页面和样式表让浏览器加载的外部域名（脚本、样式表和图标、图片、音视频、iframe、样式表里引用的地址），读者自己点的链接不算；插件取它注入的代码里的地址。

包的图标（2026-10-02，随 v0.1.8 发布）：条目可以多写一行 `icon: docs/icon.svg`，指向包的仓库里一个方形的 SVG、PNG、WebP 或 JPEG。生成器按最新收录版本的 tag 读取并核对它，索引的 `icon` 是 jsDelivr 上固定在那个 tag 的地址；图标最大 64 KB，SVG 里不能有脚本、事件、嵌入的内容或从别处加载的东西。Kite 的服务端像代取截图一样代取图标，并再核对一遍（仓库的 tag 在收录后还可能被移动），SVG 带沙箱的 CSP 返回，后台在插件卡片和详情里显示它。官方的四个插件用的是仓库里原有的 `docs/icon.svg`。

规则：

- `id` 就是装进站点后的目录名：主题的 `name`、插件的 `id`。主题和插件各是一个命名空间。
- 一个版本一经收录就不再变：同一个版本号的 zip 换了内容，CI 拒绝，而不是悄悄更新 sha256。CI 只看比最新已收录版本更新的 release，没过检查的旧版本不会每小时重试。
- `title`、`description` 的中文取自包自己的语言包（`theme.title`、`theme.description`，插件同理），没有就只有英文。
- `format` 是索引格式的大版本。Kite 遇到不认识的大版本时只浏览、不安装，并提示升级 Kite。

### 4.2 装进仓库，`kite.lock` 只记来源

`kite.lock` 原来的设计（[architecture.md §24](architecture.md#24-kitelockm6)）是 lock 里记版本，构建时按 lock 下载。插件第一版已经改成文件装进站点仓库、随仓库提交（[plugin-system.md §0.1](plugin-system.md#01-第一版实施方案2026-09-26-定) 第 3 条），官网也一直这样用。应用中心沿用后者：

- 构建不联网，CI 不多一步，应用中心下线也不影响任何站点。
- 站点仓库就是站点的全部，换一台机器 `git clone` 就能构建。
- 和上传 zip、`kite theme add` 是同一条写入路径，三种装法结果一样。

lock 只记来源，和主题、插件文件在同一个 ChangeSet 里写入、一起提交。下面是一个新站点 `kite theme add vane`、`kite plugin add search` 之后的 lock（A3，2026-10-01）：

```yaml
# Written by Kite: where each theme and plugin installed from an index
# came from. Their files are in themes/ and plugins/.
lockfileVersion: 1
themes:
  vane:
    version: 1.0.1
    source: https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json   # 从哪份索引装的
    resolved: https://cdn.jsdelivr.net/gh/kite-plus/apps@theme-vane-1.0.1/vane-1.0.1.zip
    checksum: sha256:da82c093d5e466e0de1bbd8bd075f628aa71a50fa5084bc76437ac4b8cec2947   # 下载的 zip
    tree: sha256:1b9db074f2b4f28a5a161ef7d8e3cfbb7426754529f7c0a3512c8284450efe04       # 装好后的文件
plugins:
  search:
    version: 0.1.0
    source: https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json
    resolved: https://cdn.jsdelivr.net/gh/kite-plus/apps@plugin-search-0.1.0/search-0.1.0.zip
    checksum: sha256:20dee262d440cc4ebe5053d27ab46a3c11f346ebc9c95771b54285ff5fdfc59a
    tree: sha256:ddfe916a361c81860623fe4cad436ac931329c631623294d630da4724e6a61b2
    granted:                                      # 安装时确认过的
      inject: 2
      loads: []
      hooks:
        - build_complete
```

`source` 是索引的名字，也就是第一个地址（`apps.index` 或 `KITE_APPS_URL` 换了索引时就是那一个），不管这次实际是从哪个镜像取到的；`resolved` 同样记索引里给这个压缩包的第一个地址，内容由 `checksum` 保证。lock 由存储层写：`PutTheme` / `PutPlugin` 带着来源（`Origin`）时记下，`tree` 由存储层按实际写下的文件算，不信调用方；从压缩包或目录装、删除，都会去掉这一条；记录清空时删掉 kite.lock。所以后台上传 zip 也自动去掉旧记录，不会留下和文件对不上的来源。lock 读不出来时，任何装包、删包的 ChangeSet 在写任何文件之前就失败。

用法：

- **提示更新**：lock 里有记录的，按 `source` 那份索引找新版本；lock 里没有的（上传 zip 或手放的），只有当包自己的 `homepage` 和索引里的 `repo` 对得上时才提示，免得把同名的别人的主题当成官方的。
- **发现手改**：`tree` 是目录里每个文件按 `sha256  路径`（和 sha256sum 的输出一样）一行、按路径排序后，再取一次 sha256。换行一律按 `\n` 计算，Windows 上 Git 把换行改成 `\r\n` 不算改动；`.DS_Store` 这类系统自己加的文件不算在内，和读压缩包时的规则一样。对不上说明有人改过，`kite apps update` 不更新它，除非加 `--force`；`kite doctor` 也报告这一项。
- **没有记录的包**：从压缩包装的、homepage 对得上的包，更新前先下载索引里同一个版本的压缩包比对文件，一样才当作没改过；更新后补上记录。`kite apps update vane` 对同一个版本也会这样补记录。
- **防扩权**：插件更新后的 `loads`、`hooks` 或 `inject` 比 `granted` 多时，要重新确认才装（终端里问一句，脚本里要加 `--yes`）。`granted` 取自包本身（`plugin.yaml` 和它注入的代码），不取索引里的数字。已开启的插件，更新前先像开启时那样检查一遍模块。
- `kite:` 版本锁和 `kitew` 在 M6 里实现（2026-10-01，见 [architecture.md §17](architecture.md#17-cicd)）；这里先做的是 lock 的 `themes`、`plugins` 两节。

### 4.3 上架、审核与信任

第一版就开放第三方提交（2026-10-01 定）。

**怎么上架**：作者向 `kite-plus/apps` 提一个 PR，加一份 §4.1 那样的 YAML。仓库要满足：

- 公开；发 release 时附上 `<id>-<版本>.zip`，tag 是 `v<版本>`。zip 里只有主题或插件本身，`kite theme pack` / `kite plugin pack` 打出来的就是这样（A2 加上这两个命令，打包规则和官方仓库的 `scripts/package.sh` 一致）。
- `id` 等于包里 `theme.yaml` 的 `name` 或 `plugin.yaml` 的 `id`，在索引里还没人用；`default` 和官方包的名字是保留的。
- 用允许再分发的开源许可（MIT、Apache-2.0、GPL 等），`theme.yaml` / `plugin.yaml` 用 SPDX 标识写明 `license`，清单旁有许可证全文（`LICENSE` 等）并打进 zip。索引仓库要存一份 zip 副本经 jsDelivr 分发（§4.4），没有这个许可就不能这样做，许可证全文也要随副本一起。

**CI 对每个版本都做的检查**：下载并核对 GitHub 记录的大小和 sha256；`kite theme verify` 或 `kite plugin verify`；`requires` 至少能被一个已发布的 Kite 满足，`api`、`abi` 是已知的版本；zip 里只有该有的文件；扫出 `loads`（主题用 `kite theme verify` 的夹具站点构建一次，看 `script`、`link`、`img`、`iframe` 指向哪些外部域名；插件取 `plugin.yaml` 的声明）。同一个版本号的 zip 换了内容，直接拒绝。

**人工审核什么时候需要**：

| 情况 | 怎么收 |
|---|---|
| 新条目 | CI 通过后，维护者审核：仓库和 PR 作者是同一个人或组织，名字不冒充别人，介绍和截图属实，`loads` 里的网站说得过去 |
| 已收录的包发了新版本 | CI 通过、而且 `loads`、`hooks`、`inject` 没有变多，就自动收录 |
| 新版本的权限变多了 | 生成一个待审 PR，把这个版本加进条目的 `approve`；维护者合并才收录，关闭则不再提起；同意前索引里的最新版本停在旧的。组织不允许工作流提 PR 时，改开一个 issue，附上同一处改动的链接 |
| 发现问题的版本 | 在条目里标 `yanked: [版本]`；已经装了它的站点在后台和 `kite apps outdated` 里看到提醒，不能再新装它 |
| 要下架的包 | 条目标 `delisted` 并写明原因；列表不再显示，已装的站点同样收到提醒 |

**界面上怎么区分**：官方包标「官方」，其余标「社区」。安装社区包前多一句提醒：由第三方作者维护，Kite 检查过它能正常加载、会从哪些网站加载东西，但不对内容负责。

**安装前展示**：主题和插件都写出会从哪些网站加载东西；插件另外写出注入几段代码、导出哪些钩子。插件本来就跑在没有文件、网络和环境变量的沙箱里（[plugin-system.md §10.3](plugin-system.md#103-默认不开放完整-wasi-已冻结)），读者浏览器里会加载什么，才是要让站长看清楚的部分。

**签名**（A5，2026-10-01 定用 minisign）：`kite-plus/apps` 的 Index 工作流用私钥（仓库 secret `MINISIGN_SECRET_KEY`）把 `index.json` 签成旁边的 `index.json.minisig`，公钥公布在 `minisign.pub`，也编进 Kite（`internal/apps/sign.go` 的 `Key`，KEYID `5E21CB2A5C5314BB`）。Kite 取索引时连签名一起取，签名取不到、对不上，或者索引的 `generated` 比已经用过的索引更早（有人重放旧副本，比如撤回某个版本之前的那份），都不用这份索引（地址答复了却拿不出签名，报的是签名问题；网络中断取不到，只报连不上）；缓存的副本每次用之前也重新核对签名，所以 0.1.5 留下的、没有签名的缓存会被重新获取。包不单独签名：每个压缩包在签过名的索引里都有 sha256，收录后所在的 tag 也不再变，签过名的索引已经足以证明每个包。自建索引用自己的钥匙签名（比如 `minisign -Sm index.json`，命令行默认的预哈希签名也认），`apps.key` 或 `KITE_APPS_KEY` 写它的公钥；Kite 自带索引的镜像不用写，用内置的公钥核对。工作流运行时先核对 secret 是不是 `minisign.pub` 那把钥匙，不是就让这次运行失败，免得签出一份所有 Kite 都拒绝的索引。换钥匙要先发一个带新公钥的 Kite 版本，再用新钥匙签名。

用 Kite Plus 账号（`id.kite.plus`）提交、不必会用 Git，等账号服务就绪后再做。

### 4.4 分发和镜像

包都不大：主题 100–200 KB，只注入代码的插件十几 KB，带 WASM 模块的插件（搜索、公式与图表）1 MB 上下。难点在境内访问：GitHub Release 的下载走 `objects.githubusercontent.com`，在境内慢且不稳定。

定下的做法（2026-10-01）：**索引仓库存一份 zip 副本，经 jsDelivr 分发，GitHub Release 作备用。**

- jsDelivr 的 GitHub 通道有两个上限：一个仓库在某个版本下的整体快照不能超过 50 MB，单个文件不能超过 20 MB `[EV]`。所以 zip 不能都堆在 main 分支上：带 WASM 的插件一个版本就约 1 MB，几十个版本之后整个仓库就超限了。
- 做法是**每个版本一个 tag**：CI 收录一个版本时，在 `kite-plus/apps` 里建一个只含这个 zip 的孤立提交（主题另带截图 `screenshot.<扩展名>`，索引里的截图地址也指向这里），打上 tag `<kind>-<id>-<版本>`（如 `theme-vane-1.0.1`）。提交的作者和日期是固定的（github-actions[bot]、release 的发布时间），同一个版本在哪里生成都是同一个提交。包的地址是 `https://cdn.jsdelivr.net/gh/kite-plus/apps@theme-vane-1.0.1/vane-1.0.1.zip`：每个快照只有一个文件，永远不会碰到上限；tag 不再移动，内容永远不变，可以长期缓存。main 分支只放条目的 YAML 和 `index.json`。
- 收录的 zip 不能超过 20 MB（jsDelivr 单个文件的上限）；现有的主题和插件都远在这之下。
- 索引的地址是 `https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json`，CI 更新后调 jsDelivr 的 purge 刷新这个分支地址的缓存，签名 `index.json.minisig` 一起刷新；备用是 `https://raw.githubusercontent.com/kite-plus/apps/main/index.json`。两个文件在 CDN 上各自缓存，刷新的那几秒里可能取到新索引配旧签名：这时 Kite 换下一个地址，或者用缓存的副本，不会用对不上签名的索引。以后要换成 `apps.kite.plus`，只需要在 Kite 内置的地址列表前面加一个。
- 境内访问先实测。jsDelivr 不够稳时，两种补法（§8）：在香港加一个镜像，和 Kite Plus 的服务放在一起；或者把每个版本另发成一个 npm 包，借 npmmirror 在境内的 CDN 分发。

客户端的做法和地址无关：按索引里列出的地址依次尝试，sha256 对上才用，索引本身也签了名，所以镜像不需要被信任。`KITE_APPS_URL` 环境变量（或 `kite.yaml` 的 `apps.index`）可以换成自建的索引，给内网和离线环境用。索引缓存在 `.kite/cache/apps/`，带 ETag，一小时内不重复取；断网时用缓存，并说明是多久以前的。

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
kite apps update [名字]          # 更新；插件扩权时要交互确认，或加 --yes；手改过的要 --force
```

`theme/名字`、`plugin/名字` 用来区分同名的主题和插件；`--refresh` 不等一小时、立即重取索引。

`kite theme add <zip|目录>` 和 `kite plugin add <zip|目录>` 照旧。一个参数既不是文件也不是目录时才按名字到索引里找。

### 4.7 API

```
GET  /api/v1/apps?kind=theme|plugin&q=…&refresh=true   列表，标出已装、可更新、撤回；附索引取到的时间和是否离线
GET  /api/v1/apps/{kind}/{id}                          详情，含各版本和每个版本合不合得来
GET  /api/v1/apps/{kind}/{id}/screenshot               主题截图，服务端去取并缓存，只回 PNG/JPEG/GIF/WebP/AVIF
POST /api/v1/apps/{kind}/{id}/install                  {"version": "1.0.1"}；同名已装时要 replace=true
GET  /api/v1/apps/{kind}/{id}/update?version=…         更新计划：从哪个版本到哪个、手改会不会被覆盖、插件多要了什么
POST /api/v1/apps/{kind}/{id}/update                   {"version": "…", "confirm": {"overwrite": true, "grant": true}}
```

（A4 实现，2026-10-01）截图和更新计划是实现时加的两个接口：截图由服务端代取，浏览器不碰索引里的地址；不是图片的内容（比如 HTML、SVG）一律拒绝，免得在后台的源下当成页面打开。更新计划用结构化的字段（`changed`、`more_loads`、`more_hooks`、`injected`、`grows`），后台按自己的语言说给人听；`POST …/update` 缺少需要的确认时回 409 `update_needs_confirmation`，并带上同一份计划。索引取不到时回 502 `index_unreachable`。

安装和更新最后都是一个 ChangeSet：`PutTheme` 或 `PutPlugin`，加上写 `kite.lock` 的一步，任何一步失败都不写。下载的 zip 先存在 `.kite/cache/apps/archives/`，按 sha256 命名，同一个包不重复下载。

## 5. 安全

| 威胁 | 缓解 |
|---|---|
| 下载的包被替换 | 索引里的 sha256，对不上不装；lock 记下装的是哪个 |
| 索引本身被篡改 | minisign 签名，公钥编进 Kite：签名取不到或对不上的索引不用 |
| 重放旧索引（比如撤回某个版本之前的那份） | 比已经用过的索引更早生成的不用 |
| 压缩包炸弹、路径穿越、链接 | 现有的 `archive.Unpack` 限制 |
| 插件更新后静默扩权 | `granted` 钉在 lock 里，更新后的 `loads`、`hooks`、`inject` 变多要重新确认（[plugin-system.md §10.2](plugin-system.md#102-强制点)） |
| 主题带第三方脚本 | 安装前展示 `loads`；CI 扫出来，不靠作者自己申报 |
| 同名冒充官方包 | `id` 先到先得，官方名字保留；新条目人工审核；更新只认 lock 里的来源，或 `homepage` 与 `repo` 对上的；界面标出「官方」「社区」 |
| 社区包的新版本夹带新的外部脚本 | `loads`、`hooks`、`inject` 变多的版本回到人工审核；已装的站点更新时还要再确认一次 |
| 有问题的版本已经被装了 | `yanked` 和 `delisted`，装了的站点收到提醒 |
| 手改的主题被更新覆盖 | `tree` 摘要对不上时先提醒 |

## 6. 和其他计划的关系

- **M6 `kite.lock`**：先做了 lock 的 `themes`、`plugins` 两节；`kite:` 版本锁、`kitew` 和 Cloudflare Pages 的文档随后在 M6 完成（2026-10-01）。
- **Kite Plus 账号**：第一版用 PR 上架；等 `id.kite.plus` 就绪后，作者也可以登录提交（[Explore 的 identity-and-comments.md](https://github.com/kite-plus/explore/blob/main/docs/design/identity-and-comments.md)）。
- **网页版目录**：索引是公开的，以后可以由它生成官网上的一组页面，或者放进 Explore；第一版不做。
- **原定 V4 的市场**（[architecture.md §32](architecture.md#32-现在不要设计的东西)）：应用中心是它的第一步，付费和评分仍然不做。

## 7. 分阶段

| 阶段 | 内容 | 验收 |
|---|---|---|
| **A1** 已完成 | `kite theme list/add/remove/use/new`（0.1.4），主题和插件共用一个读压缩包和目录的入口 | — |
| **A2** 已完成（2026-10-01） | [`kite-plus/apps`](https://github.com/kite-plus/apps) 仓库：条目格式、生成并检查 `index.json` 的 CI、每个版本一个 tag 的 zip 副本和 jsDelivr 刷新、提交说明和 PR 模板、审核清单；`kite theme pack` / `kite plugin pack`；收录 6 个官方包 | 6 个官方包的 11 个版本都通过检查并收录，jsDelivr 和 GitHub 两个地址下载的 sha256 都对得上；再跑一次没有改动；同一版本换内容被拒绝、权限变多的版本生成待审 PR，由端到端测试覆盖；从零上架一个测试主题（`kite theme new` → 加许可证 → `kite theme pack` → 收录）由 CI 里用真实 Kite 跑的集成测试覆盖 |
| **A3** 已完成（2026-10-01） | `internal/apps`（读索引、缓存、按 `requires` 过滤、下载、校验）、`internal/lock`、`kite.lock` 的 `themes` / `plugins`、按名字安装和更新、`kite apps search/outdated/update`、`kite doctor` 检查 lock、`apps.index` / `KITE_APPS_URL` | 对线上索引验证：新站点 `kite theme add vane`、`kite plugin add search` 装上、能构建、lock 有记录；手改过的主题 `kite apps update` 不动、`--force` 才更新，`kite doctor` 报告；断网时用缓存并说明多久以前。插件扩权要确认、从压缩包装的包按 homepage 认领并比对后更新，由端到端测试覆盖 |
| **A4** 已完成（2026-10-01） | §4.5 的页面和 §4.7 的 API，「官方」「社区」标记，撤回和下架的提醒；「设置 → 主题」和「系统 → 插件」的卡片标出「可更新」并链到应用中心；主题装好后的提示可以直接进整站预览 | 对线上索引在真实页面验证了浏览、搜索、详情、安装（插件安装前列出加载的网站、注入和钩子）、更新（手改过的要先勾选同意覆盖）、主题装好后进预览、手机宽度和深色模式；移除沿用原有的删除，lock 由存储层一起去掉。浏览器回归测试用本地假索引覆盖主题的安装和更新、插件扩权要勾选；其他浏览器测试的站点指向一个不存在的索引，不连外网 |
| **A5** 签名已发布（Kite v0.1.6，2026-10-01）；账号提交待定 | minisign 签名：apps 仓库签索引、公布公钥，Kite 内置公钥并拒绝没签名、签错和更旧的索引，`apps.key` / `KITE_APPS_KEY` 给自建索引；用 Kite Plus 账号提交等 `id.kite.plus` | 线上索引由 Index 工作流签名（`kite-plus/apps` 的 38505f6）：Kite 经 jsDelivr 和 raw.githubusercontent.com 两个地址都验过，0.1.5 照常可用，minisign 官方命令行也验过；改过一个字段的副本、没有签名的索引都被拒绝；minisign 命令行默认签出的预哈希签名可以用于自建索引；端到端测试用 Go 和 Node 两种实现签名，后者顺带核对了格式 |

## 8. 决定与待定事项

2026-10-01 定下：

1. **收录范围**：第一版就开放第三方提交（§4.3）。
2. **分发**：索引仓库存 zip 副本，经 jsDelivr 分发，GitHub Release 作备用（§4.4）。
3. **lock**：装进站点仓库，`kite.lock` 只记来源（§4.2）。
4. **后台入口**：「系统 → 应用中心」一个入口（§4.5）。

已经解决的：打包命令和 `verify` 报告的加载网站随 Kite v0.1.5 发布（2026-10-01），`kite-plus/apps` 改用 v0.1.5 检查每个版本，说明里不再让作者装 main 上的 Kite。签名方案定为 minisign（2026-10-01，用户定），不用 Sigstore：验证不用联网，内网和离线都能用。上线按先签索引、再发 Kite 的顺序：线上索引 2026-10-01 签好之后，验签的 Kite 才随 v0.1.6 发布。

还开着的 `[待定]`：

1. **境内访问**：A2 上线后实测 jsDelivr 在境内的成功率和速度，不够时加香港镜像，或另发 npm 包借 npmmirror 分发。
2. **审核的人和时限**：审核清单已写进 `kite-plus/apps` 的说明；谁审、多久内回复还没定。
3. **索引的正式域名**：先用 jsDelivr 和 raw.githubusercontent.com 的地址；要不要、什么时候换成 `apps.kite.plus`。
4. **待审版本用 PR 还是 issue**：`kite-plus` 组织不允许工作流提 PR，所以待审的版本现在会以 issue 出现（附改动链接，一键开 PR）。要改成直接提 PR，在组织设置的 Actions → General → Workflow permissions 里打开 “Allow GitHub Actions to create and approve pull requests”，再在 `kite-plus/apps` 里打开同一项。

## 证据来源

- jsDelivr GitHub 通道的上限：整体快照超过 50 MB 时拒绝服务（[jsdelivr/jsdelivr#18294](https://github.com/jsdelivr/jsdelivr/issues/18294)、[openlayers/openlayers.github.io#113](https://github.com/openlayers/openlayers.github.io/issues/113)），单个文件 20 MB（[jsdelivr/jsdelivr#18268](https://github.com/jsdelivr/jsdelivr/issues/18268)）。
- GitHub API 的匿名调用每小时 60 次，搜索每分钟 10 次：GitHub REST API 文档的 rate limits 一节。
