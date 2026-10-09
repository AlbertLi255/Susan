# Susan 合并分支全量回归测试报告

- 测试日期：2026-09-29 ~ 2026-09-30
- 被测分支：`feature/sync-ollama-upstream`（合并提交 `ab7a3494`，基线修复提交 `6cb12912`，本轮修复未提交）
- 测试平台：Windows 11 / Go（PowerShell），工作目录 `d:\projects\susanAssist\susanPlatform\Susan`
- 测试范围：合并 ollama/main 后 Susan 品牌与平台改造的完整性，覆盖核心场景、边界条件、异常路径
- 测试方法：干净环境基线复现 → 逐项归因（合并新增 / 上游既有 / 环境污染）→ 修复 → 全量重跑 → 静态硬约束核查 → 前端单测 → 真机抽测

---

## 1. 总体结论

**Susan 功能改造在合并后整体保持完整，发现的问题已全部修复或明确归因，最终全量回归通过。**

| 维度 | 结论 |
|---|---|
| Go 后端单测（除 macOS 专属 x/ 模块） | ✅ 67 个包全部通过，0 失败 |
| 合并新增代码缺陷 | 🔴 发现 2 个，**均已修复**：① darwin 桌面端引用不存在的 `appui.SusanDotCom`（macOS 构建必断）；② 上游 #17943 合并丢失 `IncludeIntermediateMetrics` 三处代码（llm 包编译失败） |
| 上游在 Windows 的既有缺陷 | 🟡 2 个，已修复/规避：glob 无匹配返回 ERROR_INVALID_NAME；host:port 无法作为 NTFS 目录（POSIX 继续覆盖） |
| Susan 品牌/配置测试资产 | 🟡 合并后测试仍断言旧品牌、设置旧 `OLLAMA_*` 变量，共修复 60+ 处，未发现生产代码品牌回退 |
| 前端 Vitest | ✅ 22 文件 / 228 用例全部通过 |
| 真机抽测 | ✅ 端口 14343、品牌路由文案、CLI、Device Flow 行为均符合代码契约 |
| 与实测清单的差异 | 1 项：清单预期 Device Flow 未配置时返回 503，实现与单元测试契约为 **400 + 可操作错误信息**（见 §6-D1），建议更新清单，未改代码 |

---

## 2. 环境与证据

### 2.1 环境注意事项

- 用户全局环境存在污染：`SUSAN_CLOUD_HOST=http://localhost:8000`；所有测试命令均在执行前 `Remove-Item Env:SUSAN_CLOUD_HOST`。
- 用户真实 `~/.susan/server.json` 内容为 `{"disable_ollama_cloud": true}`（云端功能在本机处于禁用状态）。该文件使两个缺少 home 隔离的上游测试在本机 403（已补隔离）。
- Go module 名仍为 `github.com/ollama/ollama`（有意保留，避免大面积导入路径风险）。

### 2.2 日志与产物

| 文件 | 说明 |
|---|---|
| `logs/go-test-all-baseline.log` | 修复前干净环境全量基线（8 包失败 + 2 个 x/ setup failed） |
| `logs/go-test-all-final.log` / `-summary.log` | 修复后全量（仅 x/ 2 包 setup failed + 日志目录自身临时文件导致的 build failed，后者已消除） |
| `logs/go-test-all-final-nox.log` / `-summary.log` | 排除 x/ 后全量：**67 ok / 27 no-test-files / 0 FAIL** |
| `logs/retest-llm-parser-middleware-create.log`、`retest-launch.log`、`retest-cmd.log`、`retest-appui.log`、`retest-server.log`、`retest-server-2.log` | 分包重跑证据 |
| `logs/vitest-final.log` | 前端 228/228 通过 |
| `logs/susan-smoke-serve.log` | 真机 `susan serve` 运行日志 |
| `logs/upstream-855f4bf9-llm.diff`、`upstream-parser.go.txt`、`susan-pre-merge-parser.go.txt` | 上游溯源对比证据 |

---

## 3. Susan 硬约束专项静态核查

| 约束 | 核查方式 | 结果 | 证据 |
|---|---|---|---|
| 品牌一律 Susan | 全仓 grep 生产代码用户可见字符串；CLI/路由/TUI/welcome | ✅ | CLI 帮助 `susan [flags]`；`GET /` → `Susan is running`；`cmd/tui/welcome.go`、`cmd/welcome.go` 补漏 2 处 |
| 端口 14343 | grep `11434`/`14343`，真机监听确认 | ✅ | 默认 `127.0.0.1:14343`（真机 Get-NetTCPConnection 证实）；残留 11434 仅存在于"旧配置迁移"与第三方 App（VSCode/Cline/Codex）字段夹具中，属有意保留 |
| 数据目录 `~/.susan` | grep `.ollama` 生产路径 | ✅ | 生产代码零残留；测试全部隔离到 temp HOME |
| 环境变量 `SUSAN_` 前缀 | grep 生产代码 `OLLAMA_` 读取 | ✅ | 仅修复 1 处日志键（`server/sched.go`）；保留项：`envconfig/config_test.go:57` 负向用例（验证旧变量被忽略）、integration 包 CI 变量（`OLLAMA_BIN`/`OLLAMA_TEST_*`，包默认跳过）、`server.json` 字段名 `disable_ollama_cloud`（上游文件格式兼容） |
| UA 用 susan | 静态核查 | ✅ | `api/client.go` 两处 `susan/<ver> (...)`；`app/updater` 同为 susan |
| `susan://` 协议，兼容 `ollama://` | 静态核查 + app 单测 | ✅ | Windows 注册表注册 `susan://`（`app_windows.go`）；`app.go` 同时接受两种 scheme；单测两套均覆盖 |
| namespace 默认 library | 静态核查 | ✅ | `types/model/name.go:40 defaultNamespace = "library"` |
| Device Flow（RFC 8628） | 静态核查 + 单测 + 真机 | ✅（1 项清单差异，见 §6-D1） | 路由 `POST/GET /api/signin/device`；`TestSigninDeviceAPI` 覆盖完整授权轮询；真机 idle/400 行为验证 |
| Cloud MVP 只读 registry.ollama.com | 静态核查 | ✅ | `defaultCloudHost = "https://ollama.com"` 按约束**有意保留**；registry 走 `SusanDotCom`（当前解析 ollama.com，注释说明域名上线计划） |

---

## 4. 用例执行记录（关键项）

### 4.1 基线失败项逐项闭环（对应实测清单 13 个已知失败）

| # | 用例/包 | 步骤 | 预期 | 基线实际 | 归因 | 处置 | 修复后实际 |
|---|---|---|---|---|---|---|---|
| 3A-1 | `app/ui TestUserAgent` | 解析客户端 UA | 含 susan 产品名 | 断言 ollama 失败 | 测试未随品牌更新 | 改测试断言 | ✅ pass |
| 3A-2 | `TestUserAgentTransport` | 校验 RoundTripper UA 透传 | susan UA | ollama | 同上 | 改测试 | ✅ pass |
| 3A-3 | `TestGetIntegrationStatuses` | 集成状态命令名 | Command=susan | 断言 ollama | 同上 | 改测试 | ✅ pass |
| 3A-4 | TUI Welcome / CLI welcome 文案 | 启动欢迎页 | Susan Cloud / Susan account | 残留 Ollama 文案 | **生产代码品牌遗漏** | 改生产代码 2 处 | ✅ 构建+测试通过 |
| 3B | middleware 3 用例（web_search×2、responses×1） | `t.Setenv(OLLAMA_HOST, httptest URL)` 后随访 | 请求打到测试服务器 | 旧变量被忽略→打真实 127.0.0.1:14343 拒绝连接 | 环境变量改名后测试未跟 | →`SUSAN_HOST` | ✅ pass |
| 3B | `create` 包 9 用例（IsSafetensorsLLMModel、manifest 7 处） | `OLLAMA_MODELS=temp` | 写临时目录 | 写真实 `C:\Users\albert\.susan\models\blobs` → Access denied | 同上 | →`SUSAN_MODELS` | ✅ pass（另见 3C-42） |
| 3B | `cmd/create_safetensors_test.go` 4 处 | 同上 | 隔离 | Access denied | 同上 | →`SUSAN_MODELS` | ✅ pass |
| 3B | `server/gguf_metadata_test.go` 13 处、`create_fuzz_test.go` 1 处 | 同上 | 隔离 | Access denied | 同上 | →`SUSAN_MODELS` | ✅ pass |
| 3B-37 | `server/model_thinking_test.go` 模板/上下文 6 处 | set GO_TEMPLATE/NO_CLOUD/CONTEXT_LENGTH/MODELS | 变量生效 | 旧变量被忽略；backfill 子用例失败 | 改名+缺配置 reload | →`SUSAN_*`，补 `envconfig.ReloadServerConfig()` | ✅ pass |
| 3B-38 | `TestRunThinkingNamesReachServer` | SUSAN_HOST 指测试服务器 | 请求到达 | dial 14343 拒绝 | 同上 | →`SUSAN_HOST`（前轮已改） | ✅ pass |
| 3B-39 | `cmd TestCreateHandlerRejectsForceForRemoteSafetensors` | SUSAN_CREATE_REMOTE 强制远端 | 拒绝本地小 safetensors | 走本地路径失败 | 同上 | →`SUSAN_CREATE_REMOTE`（前轮已改） | ✅ pass |
| 3C-llm | `llm` 包全部测试 | 编译并跑测 | 通过 | **build failed**：`CompletionRequest.IncludeIntermediateMetrics` 未定义 | **合并丢失上游 #17943（855f4bf9）3 处代码** | 恢复结构体字段、`TimingsPerToken` 透传、非最终块累计指标填充 | ✅ pass（3.2s） |
| 3C-parser | `parser TestCreateRequestFileGlobNoMatchUsesModelName` | Modelfile `MODEL model-*.gguf` 无匹配 | 回落为模型名（From） | Windows 报 `ERROR_INVALID_NAME` | **上游既有 Windows 缺陷**（parser.go 与上游逐字一致） | `expandPaths` 区分 glob 无匹配（返回空，回落模型名）；draft 无匹配显式报错；坏 pattern 仍返回 `ErrBadPattern` | ✅ pass |
| 3C-40 | `cmd/launch TestCodexAppManagedAuthLifecycle` | auth 文件权限 | 0600 | Windows 报 666 | 测试未区分平台 | 条件断言：POSIX 查 0600，Windows 跳过 mode 位检查（注释说明 Windows 需 ACL） | ✅ pass |
| 3C-41 | `TestLaunchCmdAutodiscoveryDefaultLaunchDoesNotForceConfigure` | 保存并复用自动发现配置 | 标签一致直接运行 | 存 "Ollama Cloud" 而 stub 返回 "Susan Cloud" → 触发重复 configure | 测试品牌标签陈旧 | 两处改 "Susan Cloud" | ✅ pass |
| — | `cmd/launch TestClaudeDesktopRestoreRemainsAvailableOnWindows` | 还原后再次可用 | 写测试 home | 写真实 `~/.susan/backup` Access denied | 测试缺 home 隔离 | 加 `setLaunchTestHome(t.TempDir())` | ✅ pass |
| 32 | `server TestCreateFromCloudSourceSuffix` | `:cloud` 后缀建模型并 Show | 200，RemoteHost=ollama.com:443 | 本机 `SUSAN_CLOUD_HOST`/server.json 污染→行为漂移、403 | 测试无环境隔离 | setTestHome + 钉 `SUSAN_CLOUD_HOST=""` + reload | ✅ pass |
| 36 | `server` model recommendations 系列 | 签名推荐请求 | URL/签名正确 | 本机污染导致目标漂移 | 同上 | setup 中统一钉 `SUSAN_CLOUD_HOST=""` | ✅ pass |

### 4.2 本轮新增发现（基线清单之外）

| ID | 用例/位置 | 场景 | 预期 | 实际 | 归因/处置 |
|---|---|---|---|---|---|
| N1 | `app/cmd/app/app_darwin.go`、`codex_app_darwin.go` 引用 `appui.SusanDotCom`×4 | macOS 桌面构建 | 编译通过 | **符号不存在（app/ui 仅定义 OllamaDotCom）→ darwin 目标必编译失败** | 合并提交 `ab7a3494` 中 darwin 侧品牌重命名做了一半。已将符号重命名为 `SusanDotCom`（取值按约束仍为 https://ollama.com，注释说明域名上线计划）。Windows 构建/app-ui 测试通过；darwin cgo 工具链本机不具备，已逐符号静态核对其余 Susan 符号均有定义 |
| N2 | `server/sched.go` 日志键 | 并发调试日志 | SUSAN_ 前缀 | `OLLAMA_MAX_LOADED_MODELS` | 改日志键（无行为影响） |
| N3 | `cmd/launch/codex_app.go`、`app/wintray/eventloop_test.go`、`app/cmd/app/app_darwin.go` | gofmt | 格式合规 | 品牌重命名后对齐空格错乱 | 已 gofmt（纯格式） |
| N4 | `create TestIsSafetensorsLLMModel/registry.example.com:5000/...` | host:port 模型名落盘 manifest | 建目录成功 | NTFS 文件名禁含 `:`，mkdir 失败 | 上游既有 Windows 平台限制（`Name.Filepath()` 与上游逐字一致）。Windows 跳过该子用例（注释说明），POSIX 继续覆盖；同时记录为产品限制（见 §6-R3） |
| N5 | `server TestGenerateChatRemote`、`TestCreateFromCloudSourceSuffix` | 远端推理/云后缀 | 200 | 读真实 `~/.susan/server.json`（disable_cloud=true）→ 403 | 上游测试缺 home 隔离，本机配置敏感。已补 setTestHome |

### 4.3 真机抽测（dist\susan.exe，51.8 MB，go build 产出）

| 步骤 | 预期 | 实际 | 结果 |
|---|---|---|---|
| `susan serve` 启动 | 监听 14343 | `127.0.0.1:14343` LISTEN | ✅ |
| `GET /` | 200 品牌文案 | `200 Susan is running` | ✅ |
| `GET /api/version` | 200 JSON | `200 {"version":"0.0.0"}`（无 ldflags 注入，符合预期） | ✅ |
| `GET /api/tags` | 200 模型列表 | 200，namespace/能力字段正常 | ✅ |
| `GET /api/signin/device` | 200 状态 | `200 {"state":"idle"}`，不含任何密钥字段 | ✅ |
| `POST /api/signin/device`（未配 CloudHost） | 清单写 503 | **400** + `Susan cloud is not configured; set the SUSAN_CLOUD_HOST ...` | ⚠️ 与清单不一致，见 §6-D1 |
| `susan --help` | susan 命令名 | `susan [flags]` / `susan [command]` | ✅ |

### 4.4 前端

`npx vitest run`（app/ui/app）：**Test Files 22 passed (22)，Tests 228 passed (228)**，6.15s。stderr 仅 react-test-renderer 弃用提示，无失败。

---

## 5. 缺陷清单（合并影响评估）

| 缺陷ID | 标题 | 分类 | 严重级 | 状态 |
|---|---|---|---|---|
| B-01 | darwin 桌面端引用未定义的 `appui.SusanDotCom`，macOS 构建中断 | **合并新增（品牌改造受合并影响）** | 高（macOS 交付阻断） | ✅ 已修复+静态核对 |
| B-02 | llm 包丢失上游 #17943 `IncludeIntermediateMetrics` 三处代码，包编译失败 | **合并新增（上游代码合入不完整）** | 高（编译阻断，流式中间指标功能缺失） | ✅ 已按上游 diff 恢复，llm 测试全绿 |
| B-03 | parser glob 无匹配在 Windows 走 Stat 返回 ERROR_INVALID_NAME，模型名回落失效 | 上游既有（Windows） | 中（边界场景） | ✅ 已修复，含回归测试 |
| B-04 | 60+ 处测试仍用 ollama 品牌断言/`OLLAMA_*` 变量，污染真实 home | 合并后测试资产漂移 | 中（测试可信度+本机数据污染） | ✅ 已全部修复 |
| B-05 | 两处云相关测试缺少 home/CloudHost 隔离 | 上游测试设计 | 中（环境敏感） | ✅ 已修复 |
| B-06 | TUI/CLI 两处用户可见 Ollama 文案遗漏 | Susan 改造遗漏 | 低（用户体验一致性） | ✅ 已修复 |
| D-01 | Device Flow 未配置 CloudHost 返回 400，与实测清单 503 不一致 | 需求契约待确认 | 低 | ⏸ 不改代码，待确认（见下） |
| R-03 | Windows 无法拉取/存储 host:port 自定义 registry 模型（manifest 目录含冒号） | 上游既有平台限制 | 中 | 记录（POSIX 覆盖，Windows 子用例跳过） |
| R-04 | 基线失败期间测试曾写入真实 `~/.susan`：`fuzz-create-gguf`、`test-draft`、`test-licenses`、`test-quantized`、`base-with-draft`、`base-with-replaced-draft` 6 个垃圾模型 | 环境污染（B-04 副作用） | 低 | 未擅自删除用户数据，建议确认后用 `susan rm` 清理 |

**D-01 说明**：`server/signin_device.go:31` 在 CloudHost 未配置时返回 `400`，错误体明确指引设置 `SUSAN_CLOUD_HOST`；单元测试 `TestSigninDeviceRequiresCloudHost` 明确断言 400 且要求消息含 SUSAN_CLOUD_HOST。代码与测试自洽且对调用方可操作，判定**实现侧契约成立、实测清单预期需更新**；如产品坚持 503（语义更贴近"服务未配置"），改码+改测试均很小，需产品确认后再动。

---

## 6. 通过率统计

| 测试集 | 通过 | 失败 | 跳过/不适用 | 通过率 |
|---|---|---|---|---|
| Go 全量（除 x/） | 67 包 ok | 0 | 27 包无测试文件；1 个 Windows 平台子用例 skip（R-03） | 100% |
| Go x/（macOS MLX 独立 module） | — | 2 setup failed | Windows 平台不适用（缺 x/mlxrunner 等 module 条目，macOS 预期） | N/A |
| 基线 → 修复对比 | 基线 8 个失败包 + 约 60 个失败/子失败用例 | 最终 0 | — | 全部闭环 |
| 前端 Vitest | 228 | 0 | 0 | 100% |
| 真机抽测 | 6 | 0 | 1 项契约差异（D-01，行为本身明确） | 见 §4.3 |

---

## 7. 残留风险与未测项

1. **x/ macOS MLX 模块**：`x/internal/mlxthreadtest`、`x/models/qwen4_exp` 在 Windows `go test ./...` 必然 setup failed（独立 Go module、缺平台条目），与基线一致，非合并回归；需 macOS CI 覆盖。
2. **macOS 桌面端**：B-01 已修复且 Windows 侧全绿，但本机无 darwin cgo（Cocoa/Sparkle）工具链，未能完成真实 macOS 链接与真机验证；建议 macOS runner 跑 `go build ./app/...` 与 app/cmd/app 测试。
3. **CMake 全构建、Windows 安装包、签名/更新链路**：本轮仅 `go build -o dist/susan.exe`，未跑 CMake/打包。
4. **integration 包**：默认跳过（需 OLLAMA_BIN/真机 server）；其中仍有 `OLLAMA_CREATE_REMOTE/OLLAMA_MODELS/OLLAMA_LIBRARY_PATH` 旧变量，若未来在 Susan 二进制上启用 integration CI，需再做一轮 SUSAN_ 改名（本轮按"低优、不动 CI 契约"保留）。
5. **Device Flow 浏览器 E2E（清单 42–50 剩余项）**：需 susan-web 平台端与人工浏览器授权，本轮仅验证端点状态码/契约/密钥不泄漏的服务端部分；未执行项不做伪造，标注为未测。
6. **真实模型推理**：未在真机发起 chat/generate（避免占用本机 GPU/下载）；llm server 层由单测覆盖（mock runner）。
7. 工作区另有两个先存改动与本轮无关：`.github/workflows/test-llamacpp-update.yaml`、`scripts/build_darwin.sh`（基线前即 modified），未触碰。

---

## 8. 本轮改动文件一览（未提交 git）

**生产代码（6）**：`llm/server.go`、`llm/llama_server.go`（恢复上游指标功能）；`parser/parser.go`（Windows glob）；`app/ui/ui.go`（SusanDotCom 重命名）；`cmd/tui/welcome.go`、`cmd/welcome.go`（品牌文案）；`server/sched.go`（日志键）；外加 3 个 gofmt 纯格式文件。

**测试代码（17）**：`app/ui/ui_test.go`、`app/wintray/eventloop_test.go`、`cmd/cmd_test.go`、`cmd/welcome_test.go`、`cmd/create_safetensors_test.go`、`cmd/launch/{command_test,codex_app_test,claude_desktop_test}.go`、`create/{create_test,manifest_test}.go`、`middleware/{web_search_test,responses_web_search_test}.go`、`server/{create_fuzz_test,gguf_metadata_test,model_thinking_test,model_recommendations_test,routes_create_test,routes_generate_test}.go`。

所有改动已通过 `gofmt -l`（零输出）与 `go build ./...`（除 x/ 预期错误外零错误）、`go vet` 范围编译验证；未执行 git commit/push。
