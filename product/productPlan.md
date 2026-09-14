# Susan MVP 改造计划

> 目标：把 Ollama 全套改造为独立品牌 Susan，形成可发布的产品。
> 原则：Susan 是**全新独立软件**，不承接 Ollama 老用户数据，数据目录直接改名，老用户全新开始。

## 已定关键决策

| 项 | 值 |
|---|---|
| GitHub 仓库 | https://github.com/AlbertLi255/Susan （当前 public，稍后改 private） |
| Go module 路径 | `github.com/AlbertLi255/Susan` |
| 数据目录 | `%LOCALAPPDATA%\Susan`（直接改名，**不做迁移**） |
| 域名 | 自行购买 `susan.com`（官网 www / 文档 docs / cloud api / registry） |
| Model Hub 就绪前 | **继续使用 ollama registry**（registry.ollama.com），不急于切换 |
| Web 仓库 | 独立新仓库 `susan_web`（private） |
| 官网定位 | 对标 ollama.com，UI/功能一致，仅品牌替换为 Susan；**自行开发、自托管**，不 fork 也不依赖第三方平台 |
| 服务器 | **自行购买实体服务器**，不使用云服务，全部自托管 |
| 对象存储 | **开源 MinIO** 自托管 + 免费 CDN（如 Cloudflare 免费版） |

---

## 阶段 1：仓库内 brand 改造（无新基础设施依赖，可立即开始）

按依赖顺序：

### 1. A1 模块路径改名
- `go.mod` 第 1 行：`module github.com/ollama/ollama` → `module github.com/AlbertLi255/Susan`
- 全仓所有 import 路径 `github.com/ollama/ollama/...` → `github.com/AlbertLi255/Susan/...` 联动替换
- 注意：这一步会影响所有 .go 文件的 import，必须最后做，否则中途编译失败

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

### 3.5. 端口改造（11434 → 14343，默认监听 0.0.0.0）
- [envconfig/config.go:23](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go#L23)：`defaultPort := "11434"` → `"14343"`
- [envconfig/config.go:42](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go#L42)：默认 host `"127.0.0.1"` → `"0.0.0.0"`（允许外部 IP 访问）
- [envconfig/config.go:21, 322](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go#L21)：注释/环境变量说明同步更新
- 测试文件批量替换 `11434 → 14343`：
  - `envconfig/config_test.go`、`api/client_test.go`、`cmd/cmd_test.go`
  - `internal/proxy/*_test.go`、`integration/utils_test.go:139`、`internal/modelref/modelref_test.go`
- 文档站 `docs/` 下所有 .mdx 中的 `11434` → `14343`（API 示例、curl 命令）
- **已验证**：14343 端口在本机空闲可用
- 注意：改完后默认监听 `0.0.0.0:14343`，需配合 Windows 防火墙放行 14343 入站规则（install.ps1 或安装包中添加）

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
- [scripts/build_windows.ps1:904-994](file:///d:/projects/susanAssist/susanPlatform/Susan/scripts/build_windows.ps1#L904-L994)：产物名 `windows-ollama-app-${arch}.exe` → `windows-susan-app-${arch}.exe`；ldflags 路径联动 A1
- [app/ollama.iss](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ollama.iss)（建议改名 `susan.iss`）：`MyAppURL`、打包文件名联动
- [scripts/install.ps1](file:///d:/projects/susanAssist/susanPlatform/Susan/scripts/install.ps1)：下载域名 `ollama.com/download` → `susan.com/download`；签名组织 `O=Ollama Inc.` → 新证书主体
- [CMakeLists.txt:3-55](file:///d:/projects/susanAssist/susanPlatform/Susan/CMakeLists.txt#L3-L55)：project 名（可选，纯内部不影响产物）
- Dockerfile / .github/workflows/：镜像名 `ollama` → `susan`

### 8. I 环境变量前缀改造（OLLAMA_ → SUSAN_）
- 当前代码使用 `OLLAMA_HOST`、`OLLAMA_MODELS`、`OLLAMA_ORIGINS`、`OLLAMA_KEEP_ALIVE`、`OLLAMA_DEBUG` 等环境变量
- 统一改为 `SUSAN_` 前缀：`SUSAN_HOST`、`SUSAN_MODELS`、`SUSAN_ORIGINS`、`SUSAN_KEEP_ALIVE`、`SUSAN_DEBUG` 等
- 改动点：[envconfig/config.go](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go) 中所有 `Var("OLLAMA_*")` 调用，以及 [config.go:322](file:///d:/projects/susanAssist/susanPlatform/Susan/envconfig/config.go#L322) 的环境变量说明表
- 联动：测试文件里的 `t.Setenv("OLLAMA_*", ...)` → `t.Setenv("SUSAN_*", ...)`、文档站里的环境变量说明
- 改后用户配置示例：`SUSAN_HOST=0.0.0.0:14343`

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

### 12. H3 Registry 后端（延后）
- 实现 `/v2/...` manifest/blobs 接口
- **就绪前：pull/push 继续走 `registry.ollama.com`**（不改 E 项代码）
- 等 Model Hub 后端稳定后，再切到 `registry.susan.com`

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
- 端口改造：11434 → 14343，默认监听 0.0.0.0（含测试文件联动）

### 批次 3：模块路径 + 构建脚本 + 环境变量联动（最后做）
- A1：go.mod + 全仓 import 路径替换
- G：构建脚本产物名、iss、install.ps1、Dockerfile、CI 联动
- I：环境变量前缀 OLLAMA_ → SUSAN_

### 批次 4：Cloud 后端
- D1-D3：cloud_proxy.go URL 替换
- H4：自建云推理代理服务

### 批次 5：官网 + Model Hub（新仓库 susan_web）
- H1、H2、H5、H6
- H3 Registry 后端延后（先用 ollama registry）
