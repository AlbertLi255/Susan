# Susan MVP 改造计划

> 目标：把 Ollama 全套改造为独立品牌 Susan，形成可发布的产品。
> 原则：Susan 是**全新独立软件**，不承接 Ollama 老用户数据，数据目录直接改名，老用户全新开始。

## 已定关键决策

| 项 | 值 |
|---|---|
| GitHub 仓库 | https://github.com/AlbertLi255/Susan （当前 public，稍后改 private） |
| Go module 路径 | **保持** `github.com/ollama/ollama`（A1 不做，为了 `git merge upstream/main`）。仓库 URL 与 module 路径不必相同。 |
| 运行时环境变量 | `OLLAMA_HOST=http://localhost:11434` → `SUSAN_HOST=http://localhost:14343`（其余 `OLLAMA_*` 同样改为 `SUSAN_*`） |
| 数据目录 | `%LOCALAPPDATA%\Susan`（直接改名，**不做迁移**） |
| 域名 | 自行购买 `susan.com`（官网 www / 文档 docs / cloud api / registry） |
| 自建启动时机 | **域名申请下来并完成 DNS（www/docs/api/registry）后，立即开始自建**（官网、Model Hub、`registry.susan.com`、对象存储等）；不等「Hub 完全稳定」再动手 |
| pull 过渡 | 自建 Registry **写入/分发未就绪前**，pull 仍走 `registry.ollama.ai`；就绪后切 `registry.susan.com`。全程与 `api.susan.com` 硬拆分（见对比表 8.4 坑 1） |
| Cloud vs Registry | **硬拆分**：鉴权/推理 → `api.susan.com`；blob/pull → Registry 域。禁止全局改端点把下载指到 API |
| Namespace 防抢注 | 精确黑名单 **+** 前缀正则 `^susan` / `^official`（服务端强制，见对比表 8.7.3 坑 2） |
| API Key scopes | `api_keys.scopes` 必填；默认 `inference:run`；网关校验真伪 + scope；禁止无 scope 超管 Key |
| Web 仓库 | 独立新仓库 `susan_web`（private） |
| 官网定位 | 对标 ollama.com，UI/功能一致，仅品牌替换为 Susan；**自行开发、自托管**，不 fork 也不依赖第三方平台 |
| 服务器 | **自行购买实体服务器**，不使用云服务，全部自托管 |
| 对象存储 | **开源 MinIO** 自托管 + 免费 CDN（如 Cloudflare 免费版） |

---

## 阶段 1：仓库内 brand 改造（无新基础设施依赖，可立即开始）

按依赖顺序：

### 1. A1 模块路径改名（**不做**）
- 原计划：`go.mod` 第 1 行 `module github.com/ollama/ollama` → `module github.com/AlbertLi255/Susan`，并全仓替换 import。
- **决定：不做。** 为了保持和上游 `git merge upstream/main` 可合入：Go module 路径与所有 `.go` import 继续使用 `github.com/ollama/ollama`。GitHub 仓库仍是 `AlbertLi255/Susan`，仓库 URL 与 module 路径不必相同。
- 构建 ldflags 继续 `-X=github.com/ollama/ollama/version.Version=...`，不随品牌改名。

### 2. A2 数据目录直接改名（不迁移）
- 将代码中所有 `"Ollama"` 数据目录引用改为 `"Susan"`：
  - [app/server/server_windows.go:18-19](file:///d:/projects/susanAssist/susanPlatform/Susan/app/server/server_windows.go#L18-L19)：`ollama.pid`、`server.log` 路径
  - [app/wintray/menus.go:92-96](file:///d:/projects/susanAssist/susanPlatform/Susan/app/wintray/menus.go#L92-L96)：View logs 打开的目录
  - [app/updater/updater_windows.go:63-64, 105](file:///d:/projects/susanAssist/susanPlatform/Susan/app/updater/updater_windows.go#L63-L64)：updates 目录
  - [app/store/store.go:198, 210](file:///d:/projects/susanAssist/susanPlatform/Susan/app/store/store.go#L198)：`db.sqlite`、`config.json`
  - [cmd/start_windows.go:23-44](file:///d:/projects/susanAssist/susanPlatform/Susan/cmd/start_windows.go#L23-L44)：CLI 启动查找路径
- **不写迁移函数**。Susan 与 Ollama 是两个独立软件，数据互不干扰。
- 模型目录已改为 `~/.susan/models/`（[envconfig/config.go:123](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go#L123)），由环境变量控制（见下方 I 项）。

### 3. A3 CLI 二进制名
- `ollama` / `ollama.exe` → `susan` / `susan.exe`
- 影响：cmd/ 入口、PATH 调用、API 文档示例、安装脚本

### 3.5. 端口改造（只改端口号 11434 → 14343，其余沿用 Ollama 设计）
- **产品决策**：只把默认端口从 `11434` 改成 `14343`，避免和本机 Ollama 抢端口。
- **不改默认监听地址**：继续与 Ollama 一致，未设置 `SUSAN_HOST` 时默认为 **`127.0.0.1:14343`**（仅本机 loopback）。不要把默认 host 改成 `0.0.0.0`。
- **Ollama 原设计（保持，仅变量名按 I 改为 `SUSAN_HOST`）**：
  - `Host()` 由环境变量 `SUSAN_HOST` 同时驱动服务端 `Listen` 和客户端连接 URL。
  - 默认 `127.0.0.1`，外网/局域网进不来。
  - 用户要对外提供 API 时，自行设置 `SUSAN_HOST=0.0.0.0:14343`（bind 语义：所有网卡）。
  - 客户端不要直连 `0.0.0.0`：继续用已有的 `ConnectableHost()`，把未指定地址换成 `127.0.0.1` / `::1`（Windows 上直连 `0.0.0.0` 会失败）。
- 代码：
  - [envconfig/config.go](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go)：`defaultPort := "14343"`；默认 host 保持 `"127.0.0.1"`。
  - 注释/环境变量说明表同步为 `127.0.0.1:14343`。
- 测试与文档：把示例里的 **`11434` 换成 `14343`**；FAQ 里「如何对外暴露」写 `SUSAN_HOST=0.0.0.0:14343`，那是用户显式配置，不是默认值。
- **已验证**：14343 端口在本机空闲可用。
- 误改记录：批次 2 曾把默认 host 改成 `0.0.0.0`（文档写错）。已改回 `127.0.0.1`，与 Ollama 一致。

### 4. B CLI 文案 + User-Agent
- [api/client.go:116-140, 181-205](file:///d:/projects/susanAssist/susanPlatform/Susan/api/client.go#L116-L140)：`User-Agent: ollama/<version>` → `susan/<version>`
- [cmd/cmd.go:652-664](file:///d:/projects/susanAssist/susanPlatform/Susan/cmd/cmd.go#L652-L664) 等 cmd/ 下所有 "Ollama" 用户可见文案 → "Susan"
- 服务端响应里的 "Ollama" 文案 → "Susan"

### 5. C 前端文案
- [ChatForm.tsx:709-712](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/ChatForm.tsx#L709-L712)："Ollama account" → "Susan account"
- [Chat.tsx:267-270](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/Chat.tsx#L267-L270)：ollama.com/settings/billing、ollama.com/upgrade → susan.com 对应路径
- [ClaudeDesktopModelsSettings.tsx](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/ClaudeDesktopModelsSettings.tsx) / [CodexDesktopModelsSettings.tsx](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/CodexDesktopModelsSettings.tsx)：UI 文案 "Ollama model" → "Susan model"
- 重命名 [lib/ollama-client.ts](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/lib/ollama-client.ts) → `susan-client.ts`，改 [api.ts](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/api.ts) 引用
- localStorage / cookie key 中 `ollama_*` 前缀 → `susan_*`

### 6. F 文档站（Mintlify，docs/）
- [docs/docs.json](file:///d:/projects/susanAssist/susanPlatform/Susan/docs/docs.json)：`name`、logo href、navbar 链接全部 ollama.com → susan.com
- 替换 logo 资源：`docs/ollama.png`、`docs/ollama-logo.svg`、`docs/logo.svg`、`docs/favicon.svg`、`docs/images/{logo,logo-dark,favicon}.png` → Susan 资源
- ~40 个 .mdx 全文 "Ollama" → "Susan"、"ollama.com" → "susan.com"（可脚本化批量替换）

### 7. G 构建脚本联动
- [scripts/build_windows.ps1:904-994](file:///d:/projects/susanAssist/susanPlatform/Susan/scripts/build_windows.ps1#L904-L994)：产物名 `windows-ollama-app-${arch}.exe` → `windows-susan-app-${arch}.exe`。**ldflags 的 module 路径不改**（A1 不做，保持 `github.com/ollama/ollama`）。
- [app/ollama.iss](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ollama.iss)：`MyAppURL`、打包文件名联动。文件名暂保持 `ollama.iss`（内容已是 Susan）。
- [scripts/install.ps1](file:///d:/projects/susanAssist/susanPlatform/Susan/scripts/install.ps1)：下载域名 `ollama.com/download` → `susan.com/download`；签名组织 `O=Ollama Inc.` → 新证书主体
- [CMakeLists.txt:3-55](file:///d:/projects/susanAssist/susanPlatform/Susan/CMakeLists.txt#L3-L55)：project 名**不改**（可选，且 CMake 内部 `OLLAMA_*` 缓存变量保持原名，便于合上游）
- Dockerfile / .github/workflows/：镜像名 `ollama/ollama` → `susan/susan`

### 8. I 环境变量前缀改造（OLLAMA_ → SUSAN_）（**已做**）
- `OLLAMA_HOST=http://localhost:11434` → `SUSAN_HOST=http://localhost:14343`
- 其余运行时变量同样改前缀：`SUSAN_MODELS`、`SUSAN_ORIGINS`、`SUSAN_KEEP_ALIVE`、`SUSAN_DEBUG` 等
- 改动文件：[envconfig/config.go](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go)、测试 `t.Setenv`、文档
- CMake / CI 构建变量（如 `-DOLLAMA_LLAMA_BACKENDS`）不改
- 合上游时，环境变量相关文件会有冲突

---

## 阶段 2：Cloud 后端（MVP 必做）

### 8. D1-D3 云代理改造
- [server/cloud_proxy.go:28-33](file:///d:/projects/susanAssist/susanPlatform/Susan/server/cloud_proxy.go#L28-L33)：`https://ollama.com:443` → `https://api.susan.com:443`（占位）
- 签名 host `ollama.com` → `susan.com`
- `X-Ollama-Client-Version` header → `X-Susan-Client-Version`
- [cloud_proxy.go:362-369](file:///d:/projects/susanAssist/susanPlatform/Susan/server/cloud_proxy.go#L362-L369)：`signin_url` → `https://susan.com/signin`

### 9. H4 自建云推理代理（新服务）
- 实现 OpenAI/Anthropic 兼容路由 + 鉴权
- 提供 cloud model 列表
- MVP 可先做 1-2 个模型的路由 + token 校验
- 部署到 `api.susan.com`

---

## 阶段 3：官网 + Model Hub（新仓库 susan_web，private）

### 10. H1 官网
- **定位**：对标 `ollama.com` 官网，UI/功能一致，仅品牌替换为 Susan
- **自研而非 fork**：在 `susan_web` 仓库自行开发，不直接 fork ollama.com 源码（ollama.com 源码未开源，且需独立可控）
- **自托管**：部署在自己的服务器上，不依赖第三方平台托管
- 页面：首页 / Download / Sign in / Settings / Upgrade / Model Hub
- 技术栈建议：Next.js + Tailwind（与桌面 app 前端同栈，便于复用组件）
- 部署到 `www.susan.com`
- 数据源：Model Hub 页面模型数据见 H2

### 11. H2 Model Hub（即 ollama 的 Library，Susan 改名为 Model Hub）
- **与 H1 的关系**：ollama 官网上的 "Library" 页面，在 Susan 官网叫 "Model Hub"，是同一个功能模块
- 功能：模型列表页、模型详情页、pull 命令展示
- **MVP 阶段**：数据源 mirror ollama library（先做只读展示，后端接口暂未自建）
- **后续阶段**：完全建立自有 registry 后，数据源切换为 Susan 自己的 mirror（susan library）
- 用户上传/发布功能留待后续

### 12. H3 Registry 后端（域名到手即开工，非无限延后）
- 触发：`susan.com` 已购 + `registry` 子域 DNS/证书就绪 → **立即开工**自建 Registry（`/v2/...` manifest/blobs + MinIO）
- 实现 `/v2/...` manifest/blobs 接口，部署到 `registry.susan.com`
- **切换完成前**：客户端 pull 仍指向 `registry.ollama.ai`（不改 E 项默认 Host；与 `api.susan.com` 硬拆分，见对比表 8.4 坑 1）
- **自建分发就绪后**：默认 Registry Host 切到 `registry.susan.com`；再开放用户 Push（对齐对比表 B7）

### 13. H5 文档站部署
- Mintlify CLI 部署 docs/ 到 `docs.susan.com`

### 14. H6 下载 CDN
- install.ps1 和官网 Download 按钮指向的发布物托管

---

## 阶段 4：外部资源（非代码）

### 15. 域名
- 自行购买 `susan.com`
- DNS 配置子域名：www / docs / api / registry
- HTTPS 证书：可用 Let's Encrypt 免费证书（自托管服务器）
- **域名 + DNS 就绪 = 自建开工信号**：官网、`api`、`registry`、对象存储并行推进，不再以「先镜像展示、Registry 无限延后」为默认节奏

### 16. 实体服务器（自托管，不使用云）
- 至少 1 台实体服务器（跑官网 + Model Hub + 文档站 + 对象存储）
- 配置建议：CPU 4 核以上、内存 8GB+、SSD 500GB+、固定公网 IP 或动态域名
- 系统环境：Linux（Ubuntu/Debian）+ Docker + Nginx
- GPU 服务器（Cloud 功能需要本地跑模型时再采购）：A100 / 4090 等
- 网络：公网带宽建议 100Mbps+，模型下载和安装包分发需要带宽

### 17. 对象存储 + CDN（自托管 + 免费 CDN）
- **MinIO**（开源）自托管，存储模型文件、安装包、静态资源
- 免费 CDN：Cloudflare 免费版（静态资源加速 + DDoS 防护）
- MinIO 暴露 S3 兼容接口，供官网和 registry 后端调用

### 18. 代码签名证书
- Windows EV 代码签名证书（给 susan.exe / 安装包签名，避免 SmartScreen 警告）
- Apple Developer ID（$99/年，macOS 应用签名 + 公证）
- MVP 内测阶段可暂缓，正式发布前必须

### 19. GitHub 仓库
- `AlbertLi255/Susan`：当前 public，稍后改 private
- 新建 `susan_web` 仓库（private）：官网 + Model Hub
- 配置 GitHub Actions CI/CD + GitHub Secrets（签名证书、服务器 SSH 密钥等）

### 20. 第三方服务（按需）
- 邮件服务：用户注册/找回密码（SendGrid 免费额度 / 自建 SMTP）
- 支付服务：如果有付费功能（Stripe）
- 日志监控：Sentry 免费版 / 自建 Prometheus + Grafana
- 第三方大模型 API：Cloud 后端转发（OpenAI / Anthropic / DeepSeek 等，如不自研模型）

### 21. 法律合规（发布前）
- 隐私政策 + 服务条款页面
- 开源协议合规：ollama 为 MIT 协议，fork 后保留原 LICENSE
- "Susan" 商标注册（品牌保护，可选）

---

## 执行批次

### 批次 1：纯文案替换（最低风险，无功能改动）
- B：CLI 文案 + UA
- C：前端文案 + 文件重命名
- F：文档站 docs.json + logo + mdx 全局替换（含 11434 → 14343）

### 批次 2：数据目录改名 + 二进制名 + 端口
- A2：所有 `"Ollama"` 数据目录引用 → `"Susan"`（不写迁移）
- A3：`ollama` CLI → `susan` CLI
- 端口改造：**只改端口号** 11434 → 14343；默认仍监听 `127.0.0.1`（与 Ollama 一致，含测试文件联动）

### 批次 3：dev 环境启动链路 pre-existing bug 根治 + Susan 客户端对接 Susan 平台

- 3.1、3.2：启动链 pre-existing bug。**两项已落地。**
- 3.3：Susan 客户端对接已就绪的 susan-platform（B0～B2 Device Flow / 账号设备）。
- **责任划分（定稿）**：
  - **susan-platform**：账号、Device Flow（`/auth/device`、`/auth/token`）、Web `/device`、`/account/devices` **已完成**（有自动化测试）。本批次**不改平台仓业务**（仅联调时本地跑起来）。
  - **Susan 本仓必须完成三块**（缺一不可，禁止再用「最小实现」砍掉下列任一项）：
    1. **Plan**：与平台对比表第七章对齐并保持同步（可微调）
    2. **联调前置 3.3.0**：本机把平台 API + Web 跑通
    3. **实现 3.3.1～3.3.7**：CloudHost、保留字、密钥单测、完整本地 Device Flow API、Whoami/Signout、CLI、Desktop（详见任务总览）

#### 3.1 resolvePath Windows .exe 扩展名缺失
- **背景**：dev 环境下 `dist/susan-app.exe` 无法自动 spawn `susan.exe serve` 子进程，导致 UI 起来后端口 14343 不绑定、API 不可用
- **根因**：[app/server/server.go:54-84](file:///d:/projects/susanAssist/susanPlatform/Susan/app/server/server.go#L54-L84) 的 `resolvePath(name)` 在 Windows 上用 `os.Stat(filepath.Join(dir, name))` 检查（`name="susan"`，无 `.exe` 后缀）；Windows 不像 POSIX 自动补 `.exe`，三步查找全部 miss，最后返回字面量 `"susan"`，`exec.Command("susan", "serve")` 失败
- **确认方式**：原 ollama 代码逻辑结构一致，批次 2 只替换字符串 `"ollama"` → `"susan"`，未动 `resolvePath` 实现；正式安装版靠 Inno Setup 把 `susan.exe` 装入 `C:\Program Files\Susan\` 并加 PATH，走 `exec.LookPath` 兜底才能跑
- **修复内容**：采用"双候选查找"写法，Windows 上同时尝试 `name` 和 `name+".exe"`（仅当 `filepath.Ext(name) == ""` 才追加，避免 `susan.exe` → `susan.exe.exe`）。具体实现：

  ```go
  func resolvePath(name string) string {
      candidates := []string{name}
      if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
          candidates = append(candidates, name+".exe")
      }
      tryCandidates := func(dir string) string {
          for _, c := range candidates {
              if _, err := os.Stat(filepath.Join(dir, c)); err == nil {
                  return filepath.Join(dir, c)
              }
          }
          return ""
      }

      if exe, _ := os.Executable(); exe != "" {
          var dir string
          if runtime.GOOS == "windows" {
              dir = filepath.Dir(exe)
          } else {
              dir = filepath.Join(filepath.Dir(exe), "..", "Resources")
          }
          if p := tryCandidates(dir); p != "" {
              return p
          }
      }

      for _, dir := range []string{
          filepath.Join("dist", runtime.GOOS),
          filepath.Join("dist", runtime.GOOS+"-"+runtime.GOARCH),
      } {
          if p := tryCandidates(dir); p != "" {
              return p
          }
      }

      if p, _ := exec.LookPath(name); p != "" {
          return p
      }

      return name
  }
  ```

- **设计要点**（避开 3 个边角问题）：
  - 用 `filepath.Ext(name) == ""` 门控，**不无条件追加** `.exe`，避免未来调用方传入 `susan.exe` 变成 `susan.exe.exe`
  - Step 3 `exec.LookPath` 保持不变：Windows 上 Go 的 LookPath 自动按 PATHEXT 尝试 `.exe`/`.com`/`.bat`，无需改
  - 优先匹配无扩展名的 `name`（保留原行为），找不到才试 `name+".exe"`，向后兼容
  - macOS/Linux 不进 Windows 分支，零影响
- **联动检查**：`ollamaServeArgs`（[server.go:64-69](file:///d:/projects/susanAssist/susanPlatform/Susan/app/server/server.go#L64-L69)）已 `switch "susan", "susan.exe"` 双形式接受；`server_windows.go` 中 `wmic ... name='susan.exe'` 是查询字符串与 resolvePath 无关，无需改
- **验证标准**：dev 环境下 `./dist/susan-app.exe` 启动后，端口 14343 自动绑定，`http://127.0.0.1:14343/` 返回 `Susan is running`，无需手动 `susan.exe serve`
- **关联代码**：[app/server/server.go](file:///d:/projects/susanAssist/susanPlatform/Susan/app/server/server.go)、[app/server/server_windows.go](file:///d:/projects/susanAssist/susanPlatform/Susan/app/server/server_windows.go)

#### 3.2 useHealth 失败后不轮询导致永久 Loading
- **背景**：批次 2 验证期间遇到一次蓝屏重启，Windows 自启的 susan-app 实例因用户级 `OLLAMA_HOST=http://localhost:11434` 遗留连不上 server，UI 健康检查失败后**永久卡在 "Loading..."**，即使 server 后来在 14343 端口恢复也不会自愈，必须手动重启窗口
- **根因**：[app/ui/app/src/hooks/useHealth.ts:8-12](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/hooks/useHealth.ts#L8-L12) 的轮询条件有逻辑 bug：
  ```ts
  refetchInterval: (query) => {
    return query.state.data === false ? 10 : false;
  },
  retry: false,
  ```
  请求**失败**时 react-query 的 `query.state.data` 是 `undefined`（不是 `false`），所以 `data === false` 永远不成立 → `refetchInterval` 返回 `false` → 失败后停止轮询。结果：启动那一刻没连上 server，就永久卡死，即使 server 后来恢复也不重试
- **确认方式**：原 ollama 代码即如此，与批次 2 改造无关；批次 2 验证期间通过截图确认修复后窗口恢复正常，反向证明根因正确
- **修复内容**：把轮询条件改为"非 true 就重试"，例如：
  ```ts
  refetchInterval: (query) => {
    return query.state.data !== true ? 1000 : false;
  },
  retry: true,
  ```
  让 `undefined`（失败）和 `false`（明确不健康）两种状态都能持续重试，server 恢复后自动恢复
- **验证标准**：
  - 启动 susan-app 时 server 未就绪，UI 显示 "Loading..."，启动 server 后**无需重启窗口**，1 秒内自动恢复为 "Select a model"
  - 模拟 server 中途 kill，UI 应每秒重试，server 重启后自动恢复
- **关联代码**：[app/ui/app/src/hooks/useHealth.ts](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/hooks/useHealth.ts)、[app/ui/app/src/components/ModelPicker.tsx](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/ModelPicker.tsx#L167-L169)（消费 `isDisabled` 显示 Loading 文案）

---

#### 3.3 Susan 客户端对接 Susan 平台（对比表 B0～B2 中 Susan 仓的任务）

> **这一节给谁看**：负责改 Susan（本仓库，Go + Desktop 前端）的人。读完本节 + 下文各 3.3.x 细则就能动手。
>
> **Susan 平台是什么**：账号、设备授权、Model Hub 所在的云端后端和官网。仓库 `susan-platform`（对比表亦称 `susan_web`）：`backend/`（FastAPI）+ `web/`（Next.js）。
>
> **现状**：平台侧 B0～B2（注册登录、Device Flow、`/account/devices`、撤设备令牌失效）**已完成并有测试**。Susan 侧对接 **未做**（仍连 `ollama.com`）。
>
> 平台接口以平台对比表第七章 + `auth_device.py` 为准；平台改接口时必须同步更新本节。

##### 术语约定（先读）

| 本节用词 | 指什么 | 属于 | 运行在哪 | 代码在哪 |
|---|---|---|---|---|
| **本地 daemon** | `susan serve`（Desktop 也会拉起），`127.0.0.1:14343` | 客户端 | 用户电脑 | 本仓库 `server/`、`cmd/`、`app/` |
| **Susan 平台** | 账号 / Device Flow / Hub 的云端后端与官网 | 云端 | 服务器或本机联调 | `susan-platform` |

- 本仓库包名 `server` = **本地 daemon**，不是平台。
- CLI/Desktop 只打本地 daemon；**只有 daemon** 请求 `{SUSAN_CLOUD_HOST}/...`。

##### 任务总览（批次 3 · Susan 侧全量待办）

> **禁止再犯的错误**：不得以「最小实现 / 薄切片」为名，砍掉本地 `POST/GET /api/signin/device`、流程管理器、`user_code` 展示、Desktop 正式入口、或平台协议字段对齐。完整交付见各 3.3.x 正文；总览下列「必须包含」为验收底线。

###### A. 责任与文档（Plan）

| 编号 | 任务 | 必须包含 | 状态 |
|---|---|---|---|
| **P0** | Plan 与平台对比表对齐 | ① 平台第七章为 Device Flow **平台协议**权威；② 本文件写清两层 API（平台 vs 本地 daemon）；③ `verification_uri_complete` = `/device?user_code=`（废止 `connect?device_code=`）；④ 方案一：`public_key` 申请时上送；⑤ 与 `product/Susan_Web_Cloud_Ollama_CompareImplementationTable.txt`、平台仓 PRD 保持同步，平台改接口则回写本节 | **已对齐**（可再微调措辞，不减范围） |

###### B. 联调前置（必须先绿，再写对接代码）

| 编号 | 任务 | 必须包含（缺一不算完成） | 状态 |
|---|---|---|---|
| **3.3.0** | 本地跑起 susan-platform | ① PostgreSQL + `alembic upgrade head`；② API `http://localhost:8000`（Swagger `/docs` 可打开）；③ Web `http://localhost:3000`；④ 能 `/signup` 注册测试号；⑤ 用 Swagger 或 curl 打通一次 `POST /auth/device`（带 `public_key`）→ 浏览器 `/device` 允许 → `POST /auth/token` 拿到令牌；⑥ 记下联调环境变量：daemon 进程设 `SUSAN_CLOUD_HOST=http://localhost:8000`（Desktop 自拉起时用**用户级**环境变量并完全退出重启） | **准备工作 · 未勾验收** |

启动命令摘要（细节见下文「3.3.0」）：

```bash
# susan-platform/backend
alembic upgrade head
uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

# susan-platform/web（另开终端）
npm run dev
```

###### C. 实现（3.3.1～3.3.7 · 本仓代码，完整交付）

| 编号 | 任务 | 对比表 | 必须包含（禁止漏项） | 依赖 | 状态 |
|---|---|---|---|---|---|
| **3.3.1** | Cloud Host 可配置，Cloud≠Registry | B0-1 | `CloudHost()`；改 cloud_proxy / Remotes / 签名 hostname 等 **Cloud 侧**；**不动** `registry.ollama.ai`；坑 1 回归测试（设 CLOUD_HOST 后 pull 仍走 Registry） | 无 | 未做 |
| **3.3.2** | 保留字 namespace | B0-3 | `IsReservedNamespace`：精确黑名单 ∪ `^susan`/`^official`；`PushHandler` 403；与平台名单一致 | 无（可并行） | 未做 |
| **3.3.3** | 设备密钥/签名单测 | B0-4 | `auth/auth_test.go`：生成密钥、Sign 可校验；Windows 跳过 POSIX 权限位断言 | 可与 3.3.4 并行 | 未做 |
| **3.3.4** | 平台客户端 + **完整本地 Device Flow API** | B2-3 | ① `auth/platform`：`StartDeviceFlow`/`PollToken`/`auth.json`/Bearer `Do`/`GetProfile`/刷新；② 登录流程管理器（单例、状态机、后台按 `interval` 轮询、`force`）；③ 处理全部 RFC 错误：`authorization_pending`/`slow_down`/`access_denied`/`expired_token`/`invalid_grant`；④ **`POST /api/signin/device` + `GET /api/signin/device`**；⑤ `api.Client`：`StartDeviceSignin`/`DeviceSigninStatus`；⑥ 本地响应**永不**含 `device_code`/令牌；⑦ 字段对齐平台七章 + `auth_device.py`（含 `public_key`） | 3.3.0、3.3.1 | 未做 |
| **3.3.5** | Whoami / Signout / `signin_url` | B2-3 | 重写 `WhoamiHandler`→平台 profile；`SignoutHandler` 删 `auth.json`；`signinURL()` 返回进行中流程的 `verification_uri_complete`；保留路径 `/api/me`；401/503 语义见正文 | 3.3.4 | 未做 |
| **3.3.6** | CLI `login` / `logout` | B2-3 | `login`/`logout` 为主命令（`signin`/`signout` 别名）；调 **本地** Device Flow API；打印 `user_code`；打开 `verification_uri_complete`；轮询本地 status/`/api/me`；区分拒绝/过期；`--force`；可选 `--revoke` | 3.3.4～5 | 未做 |
| **3.3.7** | Desktop 登录入口 / 状态 / 托盘 | B2-3 | Sign In 走同一套本地 Device Flow API（代理 `POST/GET /api/signin/device`）；展示验证码或打开完整链接；登录状态与 `/api/me` 一致；去掉硬编码 `ollama.com/connect`；与 Web `/device`、`/account/devices` 同一账号可见设备 | 3.3.4～5 | 未做 |

###### D. 决策项与验收

| 编号 | 任务 | 说明 | 状态 |
|---|---|---|---|
| 3.3.8 | B2 控制面用 Bearer；设备签名细节 → B4 | 对比表方案一；本批无额外实现工作 | 已决策 |
| **3.3.9** | 端到端联调验收 | 见下方验收清单；CLI + Desktop + 平台 Web `/device` + `/account/devices` 全绿 | 未做 |

###### 建议顺序（不可跳过 3.3.0 / 不可合并砍掉 3.3.4 本地 API）

```text
【已完成】3.1、3.2
【Plan】P0 已对齐（微调不减范围）
    → 3.3.0 平台本机跑通并手测一次完整 Device Flow     ← 联调前置，必须先做
    → 3.3.1 CloudHost（解锁所有平台请求）
    → 3.3.4 完整：平台客户端 + POST/GET /api/signin/device   ║  3.3.2、3.3.3 可穿插
    → 3.3.5 Whoami / Signout / signin_url
    → 3.3.6 CLI login/logout
    → 3.3.7 Desktop
    → 3.3.9 三端联调验收
```

###### 两层 API（和 Web 联通的前提）

| 层 | 谁实现 | 谁调用 | 路径 | 权威 |
|---|---|---|---|---|
| 平台 Device Flow | susan-platform **已完成** | **仅 daemon** | `{CLOUD}/auth/device`、`/auth/token` | 平台对比表第七章 + `auth_device.py` |
| Web 授权页 | susan-platform **已完成** | 用户浏览器 | `{web}/device?user_code=...` | 平台返回的 URI，禁止自拼 |
| 本地 Device Flow API | **Susan 本仓待做** | CLI / Desktop | `127.0.0.1:14343/api/signin/device` | 本文件 3.3.4 |
| 本地会话 | **Susan 本仓待做** | CLI / Desktop | `/api/me`、`/api/signout` | 本文件 3.3.5 |

```
CLI / Desktop
    │  本机
    ▼
Susan daemon  ──POST/GET /api/signin/device──►（内部）──►  susan-platform API
    │                                                    POST /auth/device
    │                                                    POST /auth/token
    │                                                    GET  /api/user/profile
    └─ ~/.susan/auth.json + id_ed25519*

浏览器 ──► susan-platform Web /device  （与 /account/devices 同一账号）
```

###### 完整交付清单（对照用 · 实现时逐项打勾）

| # | 交付物 | 归属 |
|---|---|---|
| 1 | `SUSAN_CLOUD_HOST` / `CloudHost()` + Cloud 侧接线；Registry 不动 | 3.3.1 |
| 2 | `IsReservedNamespace` + Push 403 | 3.3.2 |
| 3 | `auth` 包密钥/签名单测 | 3.3.3 |
| 4 | `auth/platform` 全套 + `auth.json`（含 `cloud_host`、权限） | 3.3.4 |
| 5 | 登录流程管理器（状态机 + 后台轮询 + force） | 3.3.4 |
| 6 | **`POST /api/signin/device`**、**`GET /api/signin/device`** | 3.3.4 · **不可省略** |
| 7 | `StartDeviceSignin` / `DeviceSigninStatus`（api 包） | 3.3.4 |
| 8 | 重写 `/api/me`、`/api/signout`；`signin_url`→`verification_uri_complete` | 3.3.5 |
| 9 | CLI `susan login`/`logout`：展示 `user_code`、打开链接、等结果 | 3.3.6 |
| 10 | Desktop Sign In / 状态 / 托盘；去 ollama connect 硬编码 | 3.3.7 |
| 11 | 本地响应永不泄露 `device_code`/令牌 | 全程 |
| 12 | 3.3.9 验收清单全绿 | 3.3.9 |

###### 明确拒绝（曾导致漏项的做法）

- ❌ 只改 Whoami「内部偷偷 Device Flow」、**不上** `POST/GET /api/signin/device`
- ❌ CLI 只打开浏览器、不打印 `user_code`、不等待授权结果
- ❌ Desktop「以后再说」或只改文案不接线
- ❌ 全局把 `registry.ollama.ai` 改成 `api.susan.com`
- ❌ CLI/Desktop 直连平台 `/auth/device`（绕过 daemon）
- ❌ 自拼 `connect?device_code=` 链接

###### 约束（仍有效）

- Cloud ≠ Registry；`SUSAN_CLOUD_HOST` 设在 **daemon 进程**环境
- 3.3.8：B2 控制面 Bearer；设备签名细节 → B4
- B1（官网 Hub 只读）在平台仓，Susan 无 B1 任务

##### 核心设计：登录由 daemon 完成，CLI 和 Desktop 只是入口

- 对比表第七章 **方案一**：Device Flow = 账号 ↔ 设备公钥绑定；与 Web `/account/devices` 同源。
- 平台协议对齐平台对比表；**本地完整 API** 由 daemon 提供（见交付清单 #6）。
- 令牌：`~/.susan/auth.json`（对比表 3.4），与 `id_ed25519` 同目录。
- 浏览器打开平台返回的 `verification_uri_complete`（`/device?user_code=...`）。
- 安全：本地接口永不返回 `device_code` / 令牌。

```
susan login / Desktop
        ├─► POST/GET /api/signin/device  → daemon → 平台 /auth/device|/auth/token → auth.json
        └─►（兼容）POST /api/me → 401 + signin_url → 浏览器 → 轮询 /api/me

浏览器 ──► 平台 Web /device（与 Account 设备列表联通）
```

---

##### 3.3.0 本地跑起 Susan 平台（联调前置）

> **在任务总览中的位置**：B 组 · 联调前置。未完成本节验收前，不要宣称 3.3.4～3.3.7「已联调通过」。

| 服务 | 本地地址 | 正式地址（域名就绪后） | 用途 |
|---|---|---|---|
| 平台 API | `http://localhost:8000` | `https://api.susan.com` | Susan 调用的所有接口 |
| 平台 Web | `http://localhost:3000` | `https://susan.com` | 用户在浏览器里登录、确认设备授权、管理设备 |

启动方式（在 `susan-platform` 仓库执行，详见该仓库 README）：

```bash
# 后端（需要本机 PostgreSQL）
cd backend
alembic upgrade head
uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

# 前端（另开一个终端）
cd web
npm run dev
```

- 打开 `http://localhost:3000/signup` 注册一个测试账号。
- `http://localhost:8000/docs` 是后端自动生成的接口文档（Swagger），可以直接在页面上试调接口。
- 联调时给 daemon 设置 `SUSAN_CLOUD_HOST=http://localhost:8000`（3.3.1 实现后生效）。**这个变量必须设置在 daemon 进程的环境里**。Windows 上 Desktop 会自己拉起 daemon，所以要设成用户级环境变量，然后完全退出并重启 Desktop。

**3.3.0 验收清单（全部勾完才算前置完成）**

- [ ] PostgreSQL 可用，`alembic upgrade head` 成功
- [ ] `http://localhost:8000/docs` 可打开
- [ ] `http://localhost:3000` 可打开，`/signup` 能注册测试号
- [ ] 手测一轮平台 Device Flow：`POST /auth/device`（带 `public_key`）→ 浏览器 `/device` 点允许 → `POST /auth/token` 拿到 access/refresh
- [ ] 浏览器 `/account/devices` 能看到刚绑定的设备
- [ ] 已明确：后续 Susan daemon 使用 `SUSAN_CLOUD_HOST=http://localhost:8000`（用户级环境变量 + 重启 Desktop）

Susan 需要调用的平台接口（其余平台接口只给 Web 页面用，Susan 不需要调）：

| 方法与路径 | 鉴权 | 用途 | 详见 |
|---|---|---|---|
| `POST /auth/device` | 无 | 申请设备码 | 3.3.4 |
| `POST /auth/token` | 无 | 轮询换令牌；用 refresh_token 换新令牌 | 3.3.4 |
| `GET /api/user/profile` | `Authorization: Bearer <access_token>` | 查当前登录用户，给 `/api/me` 用 | 3.3.5 |
| `GET /api/user/devices` | 同上 | 当前账号的设备列表（可选） | 3.3.6 |
| `DELETE /api/user/devices/{id}` | 同上 | 移除设备（可选，给 `logout --revoke` 用） | 3.3.6 |

注意：
- 路径前缀不统一：`/auth/...` **没有** `/api` 前缀，`/api/user/...` 有。
- 所有请求体都是 JSON（`Content-Type: application/json`），**不是** RFC 8628 原文里的表单编码（`application/x-www-form-urlencoded`）。

---

##### 3.3.1 Cloud Host 可配置，Cloud 与 Registry 硬拆分（B0-1）

**目标**：新增 `SUSAN_CLOUD_HOST`，把"账号 / 云端推理 / Hub 元数据"这类请求的目标地址从写死的 `ollama.com` 改为可配置；模型下载（pull/push 用的 Registry）完全不受影响。

**为什么必须硬拆分**（对比表 8.4「坑 1」）：上游的 `ollama.com` 同时承担两种职责——云端 API，以及模型仓库（Registry，实际域名是 `registry.ollama.ai`）。如果全局把 `ollama.com` 替换成 `api.susan.com`，`susan pull` 会去 API 服务器上找模型文件，全部 404。**禁止全局搜索替换。**

**要做的**：

1. `envconfig/config.go` 新增 `CloudHost() *url.URL`：
   - 读 `SUSAN_CLOUD_HOST`，必须带 scheme，例如 `https://api.susan.com`、`http://localhost:8000`。
   - 非本机地址必须用 `https`（令牌不能明文传输）；`localhost`、`127.0.0.1`、`::1` 允许 `http`，方便联调。校验逻辑可参考已有的 `resolveCloudProxyBaseURL`（`server/cloud_proxy.go:396`）。
   - 不允许带 path、query、fragment、userinfo。
   - 未设置时默认 `https://ollama.com`，即现网行为不变（已决策，见文末「已决策事项」第 3 条）；域名上线后再把默认值改为 `https://api.susan.com`。
   - 加进同文件 `AsMap()` 的环境变量说明表（约 338 行附近）。

2. 下列写死的地址按"归属"分别改造：

| 位置 | 现在的值 | 归属 | 怎么改 |
|---|---|---|---|
| `server/cloud_proxy.go:29-30` | `https://ollama.com:443`，签名域名 `ollama.com` | Cloud | 默认值取 `CloudHost()`，签名域名取它的 hostname。现有的 `SUSAN_CLOUD_BASE_URL`（发布模式下只允许本机地址）保留为开发用覆盖项，或并入 `SUSAN_CLOUD_HOST` |
| `server/routes.go:2200`、`2263`（`WhoamiHandler`、`SignoutHandler`） | `https://ollama.com` | Cloud | 在 3.3.5 里整体重写 |
| `server/routes.go:57`（`signinURLStr`） | `https://ollama.com/connect?name=...&key=...` | 废弃 | 改为 Device Flow 链接，见 3.3.5 |
| `api/client.go:119`、`184` | `c.base.Hostname() == "ollama.com"` 时对请求签名 | Cloud | 改为与 `CloudHost().Hostname()` 比较 |
| `envconfig/config.go:170`（`Remotes()` 默认值） | `ollama.com` | Cloud | 默认值改为 `CloudHost().Hostname()`；显式设置了 `SUSAN_REMOTES` 时仍以它为准 |
| `envconfig/config.go:30`（`Host()` 对 `SUSAN_HOST=ollama.com` 的特例） | `ollama.com` | 上游兼容 | 保持不动 |
| `app/ui/ui.go:47-51` 的 `OllamaDotCom`（读 `SUSAN_DOT_COM_URL`），被 `doSelfSigned` 用来调 `/api/me`（459 行）、`/api/tags`（1661 行） | `https://ollama.com` | Cloud | 这两处改用 `CloudHost()`；`/api/me` 在 3.3.7 改为复用 daemon |
| `app/ui/ui.go:474`（拼接头像地址） | `OllamaDotCom` | Web 站点 | 继续用 `SUSAN_DOT_COM_URL`，含义明确为"官网地址" |
| `app/ui/ui.go:552`（`/v2/{name}/manifests/{tag}`） | `OllamaDotCom` | **Registry** | 这是模型清单请求，**不能**改成 Cloud，应指向 Registry 地址 |
| `app/cmd/app/app.go:475-478`（connect 登录链接） | `https://ollama.com` | 废弃 | 在 3.3.7 改为 Device Flow |
| `anthropic/anthropic.go:1217`、`1257`（web_search） | `https://ollama.com/api/web_search` | Cloud | 接到 `CloudHost()` 即可；平台暂未提供该接口（B4 之后） |
| `types/model/name.go:39`（`defaultHost`） | `registry.ollama.ai` | Registry | **不动** |
| `x/imagegen/manifest/manifest.go:74` | `registry.ollama.ai` | Registry | **不动** |
| `app/updater/updater.go:33`、`internal/proxy/codex_desktop_errors.go:13` | ollama.com 的更新检查 / 升级链接文案 | 更新服务 / 官网 | 不属于 B0～B2，随后续 H6 / 文案批次处理 |

3. Registry 地址：B0 阶段**不新增** `SUSAN_REGISTRY_HOST`，继续用 `types/model/name.go` 的 `defaultHost = registry.ollama.ai`（对比表允许"沿用 defaultHost"）。等 `registry.susan.com` 就绪（阶段 3 的 H3）再加这个变量；届时要处理本地已下载模型目录 `~/.susan/models/manifests/registry.ollama.ai/...` 的兼容。

**测试**（`go test ./envconfig ./server ./types/model`）：
- `CloudHost()`：未设置 → 默认值；`https://api.susan.com` → 正确解析；`http://api.susan.com` → 拒绝（非本机却用 http）；`http://localhost:8000` → 允许；带 path 或 query → 拒绝。
- 坑 1 回归：`t.Setenv("SUSAN_CLOUD_HOST", "https://api.susan.com")` 之后，`model.ParseName("qwen3").Host` 仍是 `registry.ollama.ai`；pull 构造出的 manifest 请求地址的 host 仍是 `registry.ollama.ai`。
- 未设置 `SUSAN_CLOUD_HOST` 时，`Host()` 仍是 `127.0.0.1:14343`。
- cloud_proxy：设置 `SUSAN_CLOUD_HOST` 后，代理请求的目标是配置值（`server/cloud_proxy_test.go` 已有类似用例可参考）。

**验收**（对比表 B0 验收前两条）：未设置时本地行为与现网一致；设置后 Cloud 请求指向配置值，且 `susan pull` 仍走 Registry。

---

##### 3.3.2 保留字 namespace 拦截（B0-3）

**目标**：push 时，如果目标模型的 namespace（即 owner，`alice/mymodel` 里的 `alice`）是保留字，直接拒绝并返回 HTTP 403，防止用户抢注 `susan`、`official` 这类官方名字（对比表 8.7.3「坑 2」）。平台在注册用户名时用的是同一套规则（平台侧已完成），**两边名单必须一致**。

**规则**（大小写不敏感）：
- 精确黑名单，共 28 个，与平台 `backend/app/core/namespace_validator.py` 完全一致：
  `library` `models` `search` `featured` `latest` `trending` `cloud` `local` `hf` `api` `admin` `account` `docs` `blog` `signin` `signup` `device` `download` `system` `susan` `official` `susanofficial` `pricing` `settings` `support` `connect` `susancloud` `chat`
- 前缀正则：`(?i)^susan`、`(?i)^official`。
- 例子：
  - 保留：`susanx`、`SuSaN-foo`、`official-ai`、`OFFICIAL_bar`、`Library`
  - 不保留：`mysusan`、`myofficial`、`alice`
  - 空字符串：函数返回不保留（由调用方处理）

**要做的**：
1. 新增 `IsReservedNamespace(ns string) bool`，建议放在 `types/model/`（例如 `types/model/reserved.go`），server 和 cmd 都能引用。
2. 在 `server/routes.go` 的 `PushHandler`（1168 行）中，解析出模型名之后、启动推送 goroutine **之前**检查 `name.Namespace`。保留字则返回：
   `c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("namespace %q is reserved", ns)})`
   - 注意：`susan push mymodel`（不写 namespace）会被 `ParseName` 补成 `library/mymodel`，`library` 是保留字，所以会被拒——这是期望行为。
3. CLI 的 `PushHandler`（`cmd/cmd.go:990`）不需要单独判断，把本地 daemon 返回的错误信息打出来即可。
4. 以后保留字名单有改动，平台与 Susan 两边要同时改。平台也提供了查询接口 `GET /api/namespace/reserved`，Susan 不需要在运行时调用它，仅供核对。

说明：MVP 阶段 push 的目标还是上游 `registry.ollama.ai`，真正开放用户 push 在对比表 B7。这里是提前把本地防线做好。

**测试**：表驱动单测，覆盖 28 个精确保留字的原样、全大写、首字母大写三种写法；前缀变体；非保留样例；`PushHandler` 对 `susan/x`、`official-ai/x`、`library/x` 返回 403，对 `alice/x` 不返回 403（实际推送可以 mock 掉）。

---

##### 3.3.3 设备密钥与签名函数补单测（B0-4）

**现状**：功能已有，缺测试。
- 密钥生成：`cmd/cmd.go:2033` 的 `initializeKeypair()`，在 `susan serve` 启动时调用，生成：
  - `~/.susan/id_ed25519`：私钥，写入时权限 0600
  - `~/.susan/id_ed25519.pub`：OpenSSH 格式公钥，形如 `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...`
- 签名：`auth/auth.go` 的 `Sign(ctx, data)`，输出 `<公钥 base64>:<签名 base64>`，其中公钥部分是 `.pub` 文件的第二段（去掉 `ssh-ed25519 ` 前缀）。
- `auth/` 目录下目前没有任何 `_test.go`。

**要做的**（新建 `auth/auth_test.go`）：
- 用 `t.TempDir()` 作为 home。Windows 上 `os.UserHomeDir()` 读 `USERPROFILE`，其他系统读 `HOME`，两个都要设置。
- 生成密钥后断言：两个文件都在 `~/.susan/` 下；`.pub` 以 `ssh-ed25519 ` 开头；`GetPublicKey()` 的返回值与 `.pub` 文件内容（去掉首尾空白）一致。
- `Sign` 的输出能按 `:` 拆成两段，两段都是合法 base64；第一段等于公钥的第二段；用 `ssh.PublicKey.Verify` 校验签名通过；把被签名的数据改一个字节后校验失败。
- 可选：把 `initializeKeypair` 挪到 `auth` 包（例如 `auth.EnsureKeypair()`），方便测试，也方便 3.3.4 复用。
- 权限位断言在 Windows 上跳过（Windows 不支持 POSIX 权限位，见文末「已知 Pre-existing 技术债」P1）。

---

##### 3.3.4 daemon 内的平台客户端：Device Flow、`auth.json`、自动刷新（B2-3 基础）

> **交付口径（定稿）**：**完整本地 Device Flow API** + **平台协议对齐**。
>
> - 平台请求/响应/错误码：以  
>   `D:/projects/susanAssist/susanPlatform/susan-platform/document/prd/Susan_Web_Cloud_Ollama_CompareImplementationTable.txt`  
>   **第七章（3.1 / 3.5 / 3.6）** 为准，并以平台仓 `backend/app/api/auth_device.py`、`backend/app/schemas/device.py` 为实现核对。
> - 本仓库额外交付：`auth/platform`、登录流程管理器、**`POST/GET /api/signin/device`**、`api/client` 方法。
> - CLI/Desktop 只打本地 daemon；与 Web 联通靠平台同一账号、同一 `public_key` 绑定（方案一）。

建议新建包 `auth/platform`（名字可调整），只在 daemon 里使用，对外提供：
- `StartDeviceFlow(ctx, clientName) (*DeviceCode, error)`：申请设备码
- `PollToken(ctx, dc) (*Token, error)`：阻塞轮询直到出结果，处理全部错误码
- `LoadCredentials()` / `SaveCredentials()` / `DeleteCredentials()`：读写删 `auth.json`
- `Do(ctx, req)`：自动带 `Authorization: Bearer`，令牌快过期或收到 401 时自动刷新
- `GetProfile(ctx) (*Profile, error)`：查当前用户

另外需要一个 daemon 内的"登录流程管理器"（单例 + 互斥锁），负责：同一时间最多一个进行中的 Device Flow；后台 goroutine 轮询；记录当前状态（`idle` / `pending` / `authorized` / `denied` / `expired` / `failed`）；成功后写 `auth.json`。3.3.5 的 `signin_url`、3.3.6 的 `susan login`、3.3.7 的 Desktop **都复用它**；显式状态查询走 `GET /api/signin/device`。

以下是必须严格遵守的**平台**接口约定（与平台对比表第七章 + 平台实现对齐；有出入时以平台仓代码为准并回写对比表）。

###### 0. 与平台对比表的对齐要点（联通 Web 必读）

| 项 | 平台对比表 / 实现 | Susan 客户端怎么做 |
|---|---|---|
| 端点 | `POST /auth/device`、`POST /auth/token` | 仅 daemon 调用；base = `SUSAN_CLOUD_HOST` |
| 方案一 | 申请时上送 `public_key`（`id_ed25519.pub`），授权后绑定到用户 | 必送；与 `/account/devices` 列表同源 |
| `expires_in` / `interval` | 900 / 5；`slow_down` 时 interval+5 | 严格按响应里的值睡够再轮询 |
| 授权页 | `verification_uri` = `{web}/device` | **用平台返回值**，禁止自拼 ollama connect 链接 |
| 一键链接 | `verification_uri_complete` = `/device?user_code=XXXX-XXXX` | 优先打开；**禁止**把 `device_code` 放进 URL（旧稿 `connect?device_code=` 已废止） |
| 轮询错误 | 400 + 顶层 `error`（`authorization_pending` 等） | 见下表；与 RFC 8628 / 对比表 3.6 一致 |
| 令牌落盘 | `~/.susan/auth.json`（对比表 3.4） | daemon 用户目录；含 `cloud_host` |
| 本地 `/api/me` | 对比表勘误：客户端用 `/api/me` 不是 `/auth/me` | 保留路径；内部改调平台 profile |

###### 1. 申请设备码：`POST {CLOUD}/auth/device`

请求：

```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...（id_ed25519.pub 的完整内容）",
  "device_name": "DESKTOP-ABC123",
  "client_name": "Susan CLI"
}
```

- `public_key`：必填。直接读 `id_ed25519.pub`，或用 `auth.GetPublicKey()`。必须是 `ssh-ed25519` 类型；末尾的注释可有可无，平台会去掉。格式不对时平台返回 **HTTP 422**，body 是 FastAPI 的校验错误 `{"detail": [...]}`（不是下面的 OAuth 错误格式）。
- `device_name`：可选，最长 255 字符，建议用 `os.Hostname()`。会显示在网页的授权确认页和设备列表里。
- `client_name`：可选，显示在授权确认页。建议 `Susan CLI` 或 `Susan Desktop`。
- 其他字段（如 `client_id`）平台会忽略。

成功响应（HTTP 200）：

```json
{
  "device_code": "dc_susan_Q2x...（一长串随机字符）",
  "user_code": "K7PX-3MQD",
  "verification_uri": "http://localhost:3000/device",
  "verification_uri_complete": "http://localhost:3000/device?user_code=K7PX-3MQD",
  "expires_in": 900,
  "interval": 5
}
```

- `device_code`：换令牌的凭据，**保密**。只留在 daemon 内存里，不打印、不写日志、不通过 daemon 本地接口返回。
- `user_code`：给人看的验证码，格式 `XXXX-XXXX`，字符集 `23456789ABCDEFGHJKMNPQRSTUVWXYZ`（去掉了易混的 0、O、1、I、L）。平台也接受小写或不带连字符的输入。
- `verification_uri` / `verification_uri_complete`：**Web 站点**的地址，不是 API 地址（本地分别是 `:3000` 和 `:8000`）。一律使用平台返回的值，不要自己拼。优先打开 `verification_uri_complete`（验证码已经带在链接里，用户不用手输）。
- `expires_in`：900 秒（15 分钟）后设备码失效。
- `interval`：首次轮询间隔，单位秒。

用户在浏览器里看到的流程：打开链接 → 没登录就先跳到登录页，登录后自动回到授权页 → 页面显示设备名、客户端名、公钥前几位 → 点「允许」或「拒绝」。

###### 2. 轮询换令牌：`POST {CLOUD}/auth/token`

请求：

```json
{
  "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
  "device_code": "dc_susan_Q2x..."
}
```

成功（HTTP 200，响应头带 `Cache-Control: no-store`）：

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "bearer",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 3600
}
```

失败一律返回 **HTTP 400**，错误字段在 body 的**顶层**（不是包在 `detail` 里）：

```json
{"error": "authorization_pending", "error_description": "User has not yet authorized the device", "interval": 5}
```

| `error` | 含义 | 客户端怎么做 |
|---|---|---|
| `authorization_pending` | 用户还没点「允许」或「拒绝」 | 等 `interval` 秒后继续轮询（body 里带当前 `interval`，以它为准） |
| `slow_down` | 轮询太快 | 把间隔改成返回的 `interval`（平台已经加了 5 秒并记住），继续轮询 |
| `access_denied` | 用户点了「拒绝」；或者授权后、换令牌前，该设备已在网页上被移除 | 停止，提示"授权被拒绝" |
| `expired_token` | 超过 15 分钟 | 停止，提示"验证码已过期，请重新登录" |
| `invalid_grant` | 设备码不存在，或已经被用过（成功换过一次就作废） | 停止并报错 |
| `invalid_request` / `unsupported_grant_type` | 请求缺字段，或 `grant_type` 写错 | 程序 bug，停止并报错 |
| 网络错误 / HTTP 5xx | 临时故障 | 退避重试，直到 `expires_in` 用完 |

轮询节奏的硬性要求：
- 平台判断 `slow_down` 的规则是：**本次请求到达时间 − 上次请求到达时间 < 当前 interval**。所以每次收到响应后，先睡满 `interval` 秒（建议再加 1 秒余量）再发下一次。
- 同一个设备码只能有一个轮询者，不能并发轮询。
- 设备码成功换到令牌后会被平台删除，再换会得到 `invalid_grant`。

账号绑定规则（排查问题时要知道）：
- 同一个公钥已经绑定在**同一个**账号下：重复登录会复用原来的设备记录，网页设备列表不会多出一条。
- 公钥已经绑定在**另一个**账号下：用户在网页点「允许」时，平台返回 403（"public_key already bound to another user"），客户端这边只会一直收到 `authorization_pending`，直到过期。客户端的过期提示里建议加一句："如果这台设备之前登录过别的账号，请先在原账号的设备页面移除它。"

###### 3. 令牌的使用与刷新

- 两个令牌都是 JWT，但**客户端当作不透明字符串处理**，不需要、也无法校验签名。过期时间按 `expires_in` 自己计算。（仅供排查：payload 里的 `sub` 是用户 ID，`did` 是设备 ID，`typ` 是 `device_access` 或 `device_refresh`。）
- `access_token` 有效期 60 分钟；`refresh_token` 有效期 30 天。
- 调用受保护接口时带：`Authorization: Bearer <access_token>`。
- 以下情况平台返回 **HTTP 401**（body 形如 `{"detail": "Could not validate credentials / 无法验证凭据"}`，响应头 `WWW-Authenticate: Bearer`）：
  - access_token 过期或无效；
  - 该设备已在网页上被移除；
  - 误把 refresh_token 当作 access_token 使用。
- 刷新令牌：`POST {CLOUD}/auth/token`

  ```json
  {"grant_type": "refresh_token", "refresh_token": "eyJ..."}
  ```

  成功时返回与上面相同结构的**一对新令牌**，两个都要覆盖保存。失败时返回 400 `invalid_grant`，`error_description` 为 "Invalid or expired refresh token" 或 "Device has been removed; run susan login again"。此时删除本地令牌，视为已登出。
- 刷新策略：
  - 发请求前，如果 access_token 剩余有效期不到 60 秒，先刷新；
  - 请求收到 401 时，刷新一次并重试一次；
  - 刷新返回 `invalid_grant`，视为已登出。
- 平台目前不会作废旧的 refresh_token，所以即使并发刷新也不会出错；但写文件必须是原子的（见下文），并建议在 daemon 内用互斥锁把刷新串行化。

###### 4. 令牌文件 `~/.susan/auth.json`

位置：daemon 用户的 `~/.susan/auth.json`，与 `id_ed25519` 同目录。

建议的文件内容：

```json
{
  "version": 1,
  "cloud_host": "http://localhost:8000",
  "token_type": "bearer",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_at": "2026-09-23T09:30:00Z",
  "device_name": "DESKTOP-ABC123"
}
```

- `cloud_host` 记录令牌是从哪个平台拿到的。读取时如果与当前 `CloudHost()` 不一致，视为未登录（避免把令牌发给别的服务器）。
- 原子写入：先写到同目录的临时文件 → 设置权限 → 重命名覆盖。Go 的 `os.Rename` 在 Windows 上也会覆盖已有文件。
- 权限要求（对比表 3.4）：
  - Linux / macOS：目录 0700，文件 0600。读取时如果发现文件权限比 0600 宽（例如 0644），自动 `chmod 0600`；改不了就拒绝使用并打印警告。
  - Windows：用 `golang.org/x/sys/windows`（`go.mod` 里已有 v0.37.0）给文件设置**受保护的 DACL**，只保留当前用户 SID 和 SYSTEM 的完全控制权，去掉继承来的 Everyone / Users 读取权限。可以先用 `windows.ACLFromEntries` 构造 ACL，再调用 `windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)`。设置失败只记警告、不阻塞登录（对比表的要求是"尽力"）。
- 相关测试在 Windows 上不要断言 POSIX 权限位（参考技术债 P1），改为读取 DACL 进行校验。

###### 5. daemon 新增的本地接口

| 方法与路径 | 用途 | 返回 |
|---|---|---|
| `POST /api/signin/device` | 发起（或复用进行中的）一次 Device Flow，daemon 在后台轮询 | `{"user_code", "verification_uri", "verification_uri_complete", "expires_in"}` |
| `GET /api/signin/device` | 查询当前这次登录的进展 | `{"state": "idle|pending|authorized|denied|expired|failed", "user_code": "...", "error": "..."}` |

- `POST` 请求体可选 `{"client_name": "Susan CLI", "force": true}`。`client_name` 转给平台；`force` 表示放弃进行中的流程、重新申请。
- 默认行为：已有未过期的进行中流程就直接复用，返回同一个验证码。
- 轮询成功后写入 `auth.json`，状态变为 `authorized`，此后 `/api/me` 返回用户信息。
- 平台不可达或返回错误时，`POST` 返回 502 并带错误信息；`SUSAN_CLOUD_HOST` 没有指向 Susan 平台（例如还是默认的 `https://ollama.com`）时返回 400，错误信息提示"请设置 SUSAN_CLOUD_HOST 指向 Susan 平台"。
- 在 `api/client.go` 里加对应的客户端方法（例如 `StartDeviceSignin`、`DeviceSigninStatus`），供 CLI 使用。
- 在 `server/routes.go` 注册路由，参考 1904–1906 行 `/api/me`、`/api/signout` 的注册方式。

**测试**：用 `httptest.Server` 模拟平台，覆盖：
- 正常授权；`authorization_pending` 若干次后成功；`slow_down` 之后轮询间隔变大；`access_denied`；`expired_token`；`invalid_grant`；
- 刷新成功后两个令牌都被覆盖；刷新返回 `invalid_grant` 后令牌被删除；
- `cloud_host` 不一致时视为未登录；`auth.json` 的权限；
- 进行中的流程被复用，`force` 时重新申请；
- daemon 本地接口的任何响应里都不出现 `device_code` 或令牌。

---

##### 3.3.5 `/api/me`、`/api/signout`、`signin_url` 改为对接 Susan 平台（B2-3）

`/api/me`（POST）这个本地路径**必须保留**：Desktop 前端、Desktop 主进程、CLI 都在调用它（对比表第七章勘误）。

###### 1. `WhoamiHandler`（`server/routes.go:2198`）重写为

1. 读 `auth.json`。没有，或者 `cloud_host` 不一致 → 返回 `401 {"error": "unauthorized", "signin_url": "..."}`（`signin_url` 的生成见下面第 3 点）。这一步**不要**请求平台的 profile 接口。
2. 用 3.3.4 的客户端调 `GET {CLOUD}/api/user/profile`（带自动刷新）。平台返回：

   ```json
   {"id": "3f6c1d2e-...-uuid", "email": "tester@example.com", "username": "tester", "created_at": "2026-09-20T08:00:00Z"}
   ```

3. 映射到现有的 `api.UserResponse`（`api/types.go:975`）：`ID ← id`、`Email ← email`、`Name ← username`、`Plan ← "free"`（平台还没有套餐，等 B4 计费再接），其余字段留空，返回 200。
   - Desktop 主进程的 `checkUserLoggedIn`（`app/cmd/app/app.go:427`）要求 `id` 和 `name` 都非空，这个映射满足要求。
4. 刷新后仍然 401（设备被移除、refresh_token 过期）→ 删除本地令牌，按第 1 步返回 401。
5. 网络错误、平台 5xx → 保持现有语义，返回 `503 {"error": "account unavailable"}`，**不要**当作已登出处理。

###### 2. `SignoutHandler`（`server/routes.go:2252`）重写为

有 `auth.json` → 删除，返回 200；没有 → 返回 `401 {"error": "you are not currently signed in"}`（CLI 的 `SignoutHandler` 已经按这个 401 打印"未登录"）。不再调用 ollama.com 的 disconnect 接口。

###### 3. `signin_url` 改为 Device Flow 的授权链接（推荐的兼容做法）

现状：`signinURL()`（`server/routes.go:238`）用 `signinURLStr`（57 行）拼出 ollama.com 的 connect 链接。所有需要登录的 401 响应都会带上它：`WhoamiHandler`、生成/对话接口的 401 分支（约 392、2609 行），以及 cloud_proxy 的 `writeCloudUnauthorized`（`server/cloud_proxy.go:362`，通过 `cloudProxySigninURL` 变量调用）。

客户端里至少有这些地方依赖 `signin_url` 打开浏览器，然后轮询 `/api/me` 等待登录完成：
- `cmd/cmd.go:657`、`905`、`949`、`1010`（运行云端模型、signin、push 等）
- `cmd/agent_tui.go:531`
- `cmd/tui/chat/cloudauth.go:91`、`134`
- `cmd/launch/models.go:200-227`（经由 `cmd/cmd.go:80` 的 `tui.RunSignIn`）
- Desktop 前端 `app/ui/app/src/api.ts:91` 的 `fetchConnectUrl()`

**做法**：保持 `signinURL() (string, error)` 的函数签名不变，只改实现——调用 3.3.4 的登录流程管理器，复用进行中的流程（没有就新发起一次），返回 `verification_uri_complete`（例如 `http://localhost:3000/device?user_code=K7PX-3MQD`）。

效果：上面这些客户端**基本不用改代码**。它们照旧打开 `signin_url`，用户在网页上确认；daemon 后台轮询成功后写入 `auth.json`；客户端轮询 `/api/me` 拿到 200，流程结束。函数签名不变，也减少了与上游合并时的冲突。

注意事项：
- 生成失败（平台不可达、`SUSAN_CLOUD_HOST` 未指向 Susan 平台）时返回 error，调用方已有"`signin_url` 为空就不带"的分支，保持即可。
- Desktop 在未登录时会周期性调用 `/api/me`，每个设备码 15 分钟过期后会再申请一个新的，属于可接受的开销。如果嫌多，可以只在"用户主动发起登录"的场景（CLI 命令、Desktop 点登录按钮）生成新流程，`/api/me` 只返回已有的进行中流程链接。
- 各处提示文案里的 "susan.com" 字样按需调整；链接里已包含验证码，展示时建议同时把验证码打印出来，方便用户核对。
- `cmd/tui/signin_test.go` 里对 ollama.com 链接的断言要同步更新。

**测试**：`httptest` 模拟平台的 profile 接口，覆盖：无令牌 → 401、带 `signin_url`、且没有请求 profile 接口；正常 → 200 且字段映射正确；access_token 过期 → 自动刷新后 200；刷新失败 → 401 且令牌被删除；平台 500 → 503；signout 的两种情况；连续两次 401 返回的 `signin_url` 相同（复用了进行中的流程）。

---

##### 3.3.6 `susan login` / `susan logout`（B2-3）

**现状**：`cmd/cmd.go:2399-2430` 已注册 `signin`（可见）、`login`（隐藏别名）、`signout`（可见）、`logout`（隐藏别名），都带 `PreRunE: checkServerHeartbeat`（要求 daemon 已在运行，这正好符合"登录由 daemon 完成"的设计）。`SigninHandler`（936 行）调 daemon 的 `/api/me`，从 401 里取 `signin_url` 打开浏览器。

3.3.5 完成后，现有的 `SigninHandler` 就已经能走通 Device Flow，但体验不完整（不显示验证码，不等待结果，不区分拒绝和过期）。**要做的**：

1. `login` 改为可见的主命令，`signin` 作为别名（已决策）；`logout` 同理为主命令，`signout` 作为别名。帮助文案改为 "Sign in to Susan" / "Sign out of Susan"。
2. `susan login` 的流程：
   1. 调 daemon 的 `/api/me`：已登录就打印 `已登录：tester（tester@example.com）`，退出码 0。提供 `--force` 参数强制重新登录。
   2. 调 daemon 的 `POST /api/signin/device`（`--force` 时带 `"force": true`）。
   3. 打开浏览器（`browser.OpenURL`，`cmd.go` 里已经在用），同时打印：

      ```
      请在浏览器中完成授权：
          http://localhost:3000/device?user_code=K7PX-3MQD
      验证码：K7PX-3MQD（15 分钟内有效）
      如果浏览器没有自动打开，请手动访问上面的地址。
      等待授权中…（Ctrl+C 取消）
      ```

   4. 每 2 秒查一次 daemon 的 `GET /api/signin/device`（这是本机调用，不会触发平台的 `slow_down`），直到状态不再是 `pending`。
   5. 按结果输出：
      - `authorized` → 调 `/api/me`，打印 `登录成功：tester（tester@example.com）`，退出码 0
      - `denied` → `授权被拒绝`
      - `expired` → `验证码已过期，请重新运行 susan login`，并附上"设备可能已绑定其他账号"的提示
      - `failed` → 打印错误信息
      - 除成功外，退出码都不为 0
   6. Ctrl+C：CLI 直接退出；daemon 那边的流程到期后自然结束。
3. `susan logout`：调 daemon 的 `/api/signout`（3.3.5），成功后打印：

   ```
   已登出（已删除本机令牌）。
   如需让这台设备在 Susan 平台上也失效，请到 http://localhost:3000/account/devices 移除该设备。
   ```

   地址取平台 Web 站点地址。可选 `--revoke` 参数：登出前先调平台的 `DELETE /api/user/devices/{id}`，把设备从账号上移除（设备 ID 可以从 `GET /api/user/devices` 的结果里按公钥匹配得到）。这需要 daemon 再提供一个本地接口来代为调用，因为令牌只在 daemon 手里。平台目前没有"只吊销当前令牌"的接口。

**测试**：CLI 层用假的 daemon（`httptest`）覆盖：已登录直接返回；授权成功；拒绝；过期；`--force`；logout 成功；未登录时 logout 的提示。

---

##### 3.3.7 Desktop 登录入口、登录状态、托盘菜单（B2-3）

对比表要求：托盘「登录云端」唤起浏览器走同一套流程；托盘显示登录状态；状态与 `/api/me` 一致。

**Desktop 结构速览**：
- Desktop 主进程在 `app/cmd/app/`（Go），内嵌一个 UI 服务 `app/ui/ui.go`。
- React 前端 `app/ui/app/src/` 只和这个 UI 服务通信。
- UI 服务把 `/api/me`、`/api/signout` 等请求原样代理给 daemon（`app/ui/ui.go:302-309` 的 `ollamaProxy`）。
- 所以 3.3.5 完成后，Desktop 看到的登录状态自然与 `/api/me` 一致；`fetchConnectUrl()` 拿到的 `signin_url` 也已经是 Device Flow 链接。

**要做的**：

1. `app/ui/ui.go:458` 的 `UserData()` 目前自己签名请求 ollama.com 的 `/api/me`，改为复用 daemon 的 `/api/me`，避免 Desktop 出现两套登录判断。
2. `app/cmd/app/app.go:467` 的 `handleConnectURLScheme()`（由 `susan://` 协议触发）：
   - 已登录 → 打开 UI（保持不变）；
   - 未登录 → 调 daemon 的 `/api/me`，用 401 里的 `signin_url` 调 `openInBrowser`；不再调用 `auth.BuildConnectURL("https://ollama.com")`，也不再兜底打开 `https://ollama.com/connect`。
3. 托盘菜单（Windows 在 `app/wintray/menus.go`，目前没有任何登录相关的菜单项）：
   - 未登录时显示"登录 Susan 云端…"，点击后走第 2 点的流程。
   - 已登录时显示"已登录：tester"（不可点击）和"登出"；点"登出"调 daemon 的 `/api/signout`。
   - 状态来源用现有的 `checkUserLoggedIn()`（`app.go:427`），在菜单打开时或每隔几分钟刷新一次。
   - macOS 托盘同理，可以放在 Windows 之后做。
4. 前端体验改进（建议做）：
   - 在 UI 服务里再代理 `POST /api/signin/device`、`GET /api/signin/device`（加在 `app/ui/ui.go:308` 附近，同样走 `ollamaProxy`）。
   - 发起登录的 4 处调用方：`hooks/useUser.ts:31`、`routes/onboarding.tsx:97`、`components/DisplayLogin.tsx:51`、`components/Settings.tsx:442`。在打开浏览器的同时，在界面上显示验证码（例如"请在浏览器中确认验证码 K7PX-3MQD"），并能显示"已拒绝 / 已过期"。
   - `fetchConnectUrl()` 会给链接追加 `launch=true` 参数，平台的 `/device` 页面会忽略它，不影响使用。
   - `Onboarding.test.tsx` 里 `susan.com/connect?...` 相关的断言随之修改。
5. 头像：平台没有头像字段；`api.ts:77` 只在有 `avatarurl` 时才处理，无需修改。

**测试**：前端单测覆盖"发起登录后显示验证码""登录成功后切换为已登录状态"；手动验证托盘菜单两种状态的切换。

---

##### 3.3.8 云端请求的鉴权方式（方案一：设备签名还是 Bearer）——已决策

**背景**：Susan 现有的云端请求用设备私钥签名，平台目前只认 Bearer 令牌，两边格式对不上：

| | Susan 现状 | 平台现状 |
|---|---|---|
| 放在哪个请求头 | `Authorization: <公钥 base64>:<签名 base64>`（没有 scheme） | `Authorization: Bearer <access_token>` |
| 签名的内容 | `"<METHOD>,<path>?ts=<unix 秒>"`（`api/client.go:121`、`server/cloud_proxy.go:388`） | 有一个没有接入任何接口的 `DeviceAuth.verify_device_signature`，签名内容是 `"<完整公钥>:<时间戳>"`，时间窗口 ±300 秒 |
| 实际在用的地方 | 发往 `ollama.com` 的请求 | 无 |

两者都用 `Authorization` 头，不能同时携带。

**决定（2026-09-23）**：
- **B2 只用标准 Bearer 令牌**：`Authorization: Bearer <access_token>`（3.3.4、3.3.5 已覆盖）。设备签名逻辑推迟到 B4（GPU 推理 MVP）之前再定。
- 依据：
  - B2 / B3 的重点是 Device Flow 授权和本地 daemon 登录，Bearer 令牌已经足够满足控制面鉴权，不必现在增加云端中间件的调试成本。
  - 对比表 B2 的验收项"移除设备后旧凭证失效"已经通过令牌里绑定的设备 ID 实现，平台侧已验证。
- 后续演进路线（已确定）：B4 如果要启用设备级签名，采用"职责分离"的规范：
  - `Authorization` 头继续只放 Bearer 令牌；
  - 设备签名改放在自定义请求头（例如 `X-Susan-Signature`），不再占用 `Authorization`；
  - 签名内容、时间窗口等细节在 B4 前与平台一起定。
- 对本节的影响：本地 daemon 发往 Susan 平台的请求（3.3.4 的平台客户端、`server/cloud_proxy.go`）一律带 Bearer，**不再**把设备签名放进 `Authorization`。令牌只在本地 daemon 手里，CLI 不直接请求 Susan 平台。发往 `ollama.com`（默认值）的请求保持现有行为不变。

---

##### 3.3.9 端到端联调验收清单

前置：按 3.3.0 跑起平台并注册测试账号；daemon 的环境变量设置 `SUSAN_CLOUD_HOST=http://localhost:8000` 后启动。

| # | 操作 | 期望结果 | 对应 |
|---|---|---|---|
| 1 | 不设置 `SUSAN_CLOUD_HOST`，启动 `susan serve` | 仍监听 `127.0.0.1:14343`，本地模型照常可用 | B0-1 |
| 2 | 设置 `SUSAN_CLOUD_HOST` 后执行 `susan pull qwen3:0.6b` | 从 `registry.ollama.ai` 正常下载 | B0-1 坑 1 |
| 3 | `susan cp <本地模型> susan/test`，再 `susan push susan/test` | 报错，本地 daemon 返回 403 "namespace ... is reserved" | B0-3 |
| 4 | `susan login` | 打印链接和验证码并打开浏览器；在网页登录并点「允许」后，CLI 显示登录成功 | B2-3 |
| 5 | 查看 `~/.susan/auth.json` | 文件存在；Linux/macOS 权限为 0600；Windows 上只有当前用户和 SYSTEM 有权限 | B2-3 |
| 6 | 浏览器打开平台的 `/account/devices` | 能看到这台机器（设备名为主机名） | B2 验收 |
| 7 | `susan login --force` 再登录一次 | 成功；设备列表**没有**多出一条 | B2-3 |
| 8 | 看 Desktop 设置页和托盘 | 显示已登录的用户名，与 `curl -X POST http://127.0.0.1:14343/api/me` 的结果一致 | B2 验收 |
| 9 | 运行一个需要登录的云端功能（未登录状态下） | 自动打开 Device Flow 授权页，确认后命令继续执行 | B2-3 |
| 10 | 再发起一次 `susan login --force`，在网页上点「拒绝」 | CLI 显示授权被拒绝，退出码不为 0 | B2 错误码 |
| 11 | 发起登录后不做任何操作，等 15 分钟 | CLI 显示验证码已过期 | B2 错误码 |
| 12 | 在网页上移除该设备，再调 `/api/me` | 返回 401；Desktop 刷新后变为未登录 | B2 验收 |
| 13 | 移除后再次 `susan login` | 可以重新绑定，设备列表里重新出现 | B2-3 |
| 14 | `susan logout` | `auth.json` 被删除；`/api/me` 返回 401 | B2-3 |
| 15 | `go test ./envconfig ./auth/... ./server ./types/model ./cmd/...` | 全部通过（Windows 上已知的技术债 P1 除外） | 回归 |

---

##### 已决策事项（2026-09-23）

1. **登录流程由谁做**：由本地 daemon 做（申请设备码、轮询、保存 `auth.json`），CLI 和 Desktop 只负责展示链接和验证码。见「核心设计」。
2. **`signin_url` 怎么处理**：本地 daemon 把 Device Flow 的授权链接填进 `signin_url`，现有依赖它的调用方基本不改。见 3.3.5 第 3 点。
3. **`SUSAN_CLOUD_HOST` 的默认值**：域名上线前保持 `https://ollama.com`（现网行为不变，符合对比表 B0"不改默认 Host"）。此时 Device Flow 不可用（ollama.com 没有这些接口），`susan login` 和 `signin_url` 生成直接提示"请设置 SUSAN_CLOUD_HOST 指向 Susan 平台"，不再走 ollama.com 的 connect 流程。域名上线后把默认值改为 `https://api.susan.com`（一行改动）。
4. **命令名**：`login` 为主命令，`signin` 为别名；`logout` 为主命令，`signout` 为别名（对比表写的是 `susan login`）。
5. **未登录时 `/api/me` 是否自动发起 Device Flow**：有进行中的就复用，没有就新发起。有问题再收紧为"只在用户主动登录时发起"。
6. **云端请求鉴权**：B2 只用 Bearer；B4 如启用设备签名，放在 `X-Susan-Signature` 等自定义头，`Authorization` 继续只放 Bearer。见 3.3.8。
7. **`SUSAN_HOME`**：本节不做，需要时单独排期。

##### 不在本节范围

- Web 页面、平台后端的任何改动（平台 B0～B2 已完成）。
- 云端推理的 `:cloud` 路由、`/v1` 接口契约测试（对比表 B3 / B4）。
- 自建 `registry.susan.com` 与 `SUSAN_REGISTRY_HOST`（阶段 3 的 H3）。
- 开放用户 push（对比表 B7）。

---

### 批次 4：构建脚本 + 环境变量联动（**已做**）
- A1：**不做。** go.mod / import 路径保持 `github.com/ollama/ollama`，为了保持和上游的合并。
- G：构建脚本产物名、iss 内容、install.ps1、Dockerfile、CI 镜像名已联动。ldflags 的 module 路径不改。
- I：`OLLAMA_HOST=http://localhost:11434` → `SUSAN_HOST=http://localhost:14343`。合上游时这里会有冲突。

### 批次 5：Cloud 后端
- D1-D3：cloud_proxy.go URL 替换（Cloud Host 可配置，见对比表 P0-a）
- H4：自建云推理代理服务
- 路由边界：直连 `api.susan.com` 不强制 `model` 带 `:cloud`；`:cloud` 仅用于本机 Daemon 转发（见对比表 8.7.2）
- Usage MVP：进程内累计 + 断开强制结算 + PostgreSQL；**本批次不引入 Redis**

### 批次 6：官网 + Model Hub（新仓库 susan_web）
- H1、H2、H5、H6
- H3 Registry 后端延后（先用 ollama registry）

### 批次 7+ / 对齐对比表 B5-2（后续）：Redis 与多节点增强
触发条件：第二台 API 节点，或需要跨实例计费/限流/Key 撤销广播时。
（对比表第一部分编号：B5-2；第二部分【批次 5】5-2）
- Redis 基础设施
- 流式用量分桶增量上报（对比表第十五章）
- API Key 二级缓存 + Pub/Sub 撤销
- （可选）网关限流跨节点聚合

**MVP / B0～B4 明确不做 Redis。**

### 批次 8+ / 对齐对比表 B6（后续）：Susan Web Chat（chat.susan.com）
对标 DeepSeek 式网页云端对话 + 可分享会话链接。
触发条件：B2 登录 + B4 推理 + B4-3 Usage 可用之后。
（对比表第一部分编号：B6-1～B6-6；第二部分【批次 6】）
- 子域 `chat.susan.com`
- 会话 URL：`/c/{uuid}`；分享：`/s/{shareId}`
- 调用 `api.susan.com` 流式推理；第一期不与 Desktop 双向同步

**B1 明确不做 Web Chat。**

### 批次 9+ / 对齐对比表 B7（后续）：模型 Push / 发布管理
触发条件：自建 `registry.susan.com` 写入端就绪 + B2 登录可用。
（对比表第一部分编号：B7-1～B7-5；第二部分【批次 7】）
- CLI `susan push` 开放写入
- Web `/account/models`：我发布的模型列表、下架、改元数据
- License / 可见性 / Hub 收录联动
- 一期大文件仍以 CLI 为主；网页辅助上传可选二期

**B0～B6 明确不做真实用户 Push。**

---

## 已知 Pre-existing 技术债（批次 2 验证期间发现，非本次改动引入）

下列两项问题在批次 2 测试联动阶段暴露，经 `git stash` 回到批次 2 之前的 HEAD 重跑同一用例复现相同错误，确认是**批次 2 之前就存在的 bug**，与品牌改造（数据目录改名 / CLI 改名 / 端口改造）无关，留待后续单独修复。批次 2 不处理，以避免越界改动无关代码。

| 编号 | 测试用例 | 现象 | 根因摘要 | 建议修复方向 |
|---|---|---|---|---|
| P1 | `TestCodexAppManagedAuthLifecycle` | `auth mode = 666, want 600` | Windows 文件系统不实现 POSIX 权限位，`os.Chmod(path, 0o600)` 为 no-op，`os.Stat().Mode()` 返回默认 `0666` | 测试内对 `runtime.GOOS == "windows"` 做 skip 或调整断言；或改用平台感知的权限校验（Windows 走 ACL，POSIX 走 mode） |
| P2 | `app/ui/app/src/components/__tests__/Onboarding.test.tsx`（5 个用例） | `expected '<main...' to contain 'Use Susan models in Claude Desktop'`、`No apps found` | 测试渲染 Onboarding 组件时 `apps` 数组为空，相关分支未渲染；组件 mock 上下文/数据源缺失 | 在测试 setup 中补齐 `apps` mock 数据；或调整组件默认值，使无 apps 时也能渲染兜底文案 |

### P1 Windows 不支持 POSIX 文件权限位

- **测试用例**：`TestCodexAppManagedAuthLifecycle`
- **现象**：断言失败 `auth mode = 666, want 600`
- **根因**：
  - 测试期望私钥文件 `~/.susan/id_ed25519` 的权限位是 `0o600`
  - 代码通过 `os.Chmod(path, 0o600)` 设置权限，但 Windows 文件系统不实现 POSIX 权限位，该调用为 no-op
  - Windows 上 `os.Stat().Mode()` 返回默认 `0666`，断言恒为 `666 != 600`
- **关联代码**：[auth/auth.go](file:///d:/projects/susanAssist/susanPlatform/Susan/auth/auth.go)（私钥写入与 chmod）

### P2 前端 Onboarding 组件测试 mock 缺失

- **测试用例**：`app/ui/app/src/components/__tests__/Onboarding.test.tsx`（共 5 个用例）
- **现象**：
  - `expected '<main...' to contain 'Use Susan models in Claude Desktop'`
  - `No apps found`
- **根因**：
  - 测试渲染 Onboarding 组件时 `apps` 数组为空，导致相关分支未渲染
  - 是组件 mock 上下文/数据源缺失，与品牌字符串替换（`ollama` → `susan`）无关
- **关联代码**：[app/ui/app/src/components/Onboarding.tsx](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/app/src/components/Onboarding.tsx)
