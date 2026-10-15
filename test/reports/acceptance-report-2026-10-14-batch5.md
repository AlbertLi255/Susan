# Susan 批次 5 验收报告：对接云端账号与 `:cloud`

- 日期：2026-10-14
- 分支：`feature/susan_task05`
- 对照提交：`4afba613`
- 测试人：AI 辅助测试
- 被测二进制：`susan.exe`（`go build`，2026-10-14 16:24，`susan version is 0.0.0`，未注入 ldflags）；第二轮补测另用当前工作区 `go build` 出的 CLI 与 Desktop（`app/cmd/app -dev`）
- 判定依据：`product/productPlan.md` 批次 5 节（5.1～5.5、AT-5.2/5.4、E2E-5.1～5.5）

---

## 一、环境

| 项 | 值 |
|---|---|
| Susan daemon | `127.0.0.1:14343`，启动时 `SUSAN_CLOUD_HOST=http://localhost:8000`，日志 `Susan cloud disabled: false` |
| susan-platform API | `http://localhost:8000`（FastAPI/uvicorn，`BILLING_MODE=mock`） |
| susan-platform Web | `http://localhost:3000` |
| 登录账号 | `susan_test@example.com`（用户名「测试一号」，plan=free，credit=1000） |
| 凭据 | `~/.susan/auth.json`，`cloud_host=http://localhost:8000`（原 access_token 已过期，由 daemon 自动刷新成功） |
| 云端开关 | `~/.susan/server.json` 由用户改为 `{"disable_ollama_cloud": false}`；原文件备份见 `server.json.bak-batch5` |

环境备注：

1. daemon 必须以**非沙箱**方式启动。沙箱内启动时，刷新 token 时写 `~/.susan/auth.json` 被拒，`/api/me` 会返回 `503 {"error":"account unavailable"}`。这与 `disable_ollama_cloud` 无关。
2. `~/.susan/models` 中残留大量回归测试夹具（`testcompat`、`example`、`laguna-xs.2`、`gemma4` 等），daemon 启动时提示 `corrupt manifests detected`，属既有状态，与批次 5 无关。
3. 平台 `CLOUD_RUNTIME_HOST` 为空时回退 `SUSAN_LOCAL_HOST=http://127.0.0.1:14343`，即本机 daemon 充当云端运行时。第二轮补测为此在本机拉取了 `qwen3:0.6b`（保留在 `~/.susan/models`，可用 `susan rm qwen3:0.6b` 删除）。
4. Desktop 用开发模式启动：`app/ui/app` 下 `npm run dev`，另起 `app.exe -dev --fast-startup`，`SUSAN_APP_DB_PATH` 指向临时库，不动正式 Desktop 数据。UI 服务在 `127.0.0.1:3001`，代理到本机 daemon。

---

## 二、自动化验收

| 编号 | 行为 | 结果 | 证据 |
|---|---|---|---|
| AT-5.2-a | profile 成功且 overview 返回 `pro` → `/api/me` 的 `plan` 为 `pro` | **通过**（修复 F1 后复验） | `TestWhoamiPlanFromOverview` PASS；真实平台复验见 F1 修复记录 |
| AT-5.2-b | overview 失败 → `/api/me` 仍 200，`plan` 为空 | 通过 | `TestWhoamiPlanEmptyOnOverviewFailure` PASS |
| AT-5.4-a | 402 → `usage_limit_upgrade` | 通过 | `TestGetError402ReturnsUsageLimitUpgrade` PASS；用例改用真实错误 `api.StatusError{402, "402 Payment Required", "Insufficient credit balance / 余额不足，请充值"}`，与 E2E-5.2 中 CLI 实际打印的串一致 |
| AT-5.4-b | `susancloud/*` 403 → 同一升级条 | 通过 | `TestGetErrorSusancloud403ReturnsUsageLimitUpgrade` PASS；用例改用真实错误 `403 Forbidden: Pro plan required for susancloud models / susancloud 模型需 Pro 档`；实测同样命中（见 E2E-5.3） |
| AT-5.4-c | 429、保留字 Push 403、云端关闭 403 → 不进升级条 | **通过**（缺口已闭合） | `TestGetErrorOther403DoesNotReturnUsageLimitUpgrade` 三个子用例 PASS，其中保留字用例已改用真实错误串 `403 Forbidden: namespace "susancloud" is reserved`；429 实测命中（见 E2E-5.4），保留字 Push 实测见 R1 |

执行的命令与结果：

```
go test ./server/ -run Whoami -count=1      -> ok   (7 个子用例全 PASS)
go test ./server/ -run 'Whoami|Signin|Signout|Reporting' -count=1 -> ok
go test ./server/ -run TestReporting -count=1 -v -> PASS（4 + 6 子用例）
go test ./app/ui/ -run TestGetError -count=1 -> ok   (5 个子用例全 PASS)
go test ./auth/platform/... -count=1         -> ok
npx vitest run  (app/ui/app)                 -> Test Files 22 passed (22)
                                                Tests 228 passed (228), 0 failed
                                                （含 Settings.test.ts 8 条、Settings.interaction.test.tsx 3 条）
```

第二轮（R4/R5 修复与 402/403 用例改真实串之后）复跑：

```
go test ./app/ui/ -run TestGetError -count=1 -v                      -> PASS（3 个用例 + 3 个子用例）
go test ./server/ -run 'Whoami|Signin|Signout|Reporting' -count=1    -> ok
go test ./auth/platform/ -count=1                                    -> ok
npm ci && npx vitest run   (app/ui/app)                              -> Test Files 22 passed, Tests 228 passed
npx tsc -b                 (app/ui/app)                              -> 0 错误
```

---

## 三、手工验收（E2E）

| 编号 | 操作 | 期望 | 结果 | 证据 |
|---|---|---|---|---|
| E2E-5.1 | 已登录，`susan run <模型>:cloud` | 先 show，再出回复 | **通过**（第二轮） | `susan run qwen3:0.6b:cloud "Reply with just the word hello." --think=false` → 输出 `Hello!`。链路：CLI → daemon `:cloud` 设备签名转发 → 平台 `/api/show`、`/api/chat` → 运行时（本机 daemon 的 `qwen3:0.6b`）。另见观察项 O1（cloud stub 警告）。首轮用 `gpt-oss:20b:cloud` 得到平台 404，已证明签名转发可用 |
| E2E-5.2 | 余额用尽后再跑 | Desktop 出现英文升级条；free 链 `/upgrade`，pro 链 `/settings/billing` | **通过**（第二轮） | 测试数据：`official/qwen3.price_cny_per_m_token=100`，测试账号 `credit_balance=0`。CLI：`Error: 402 Payment Required: Insufficient credit balance / 余额不足，请充值`。Desktop（free）：对话出现 "You've reached your usage limit, please upgrade to continue" + Upgrade，链接 `https://susan.com/upgrade`。把账号改为 `plan=pro`、有效期 30 天后再跑：同一升级条，链接 `https://susan.com/settings/billing` |
| E2E-5.3 | 非 Pro 跑 `susancloud/*` | 同一条升级条 | **通过** | 首轮：`POST /api/chat {"model":"susancloud/demo:cloud"}` → `403 {"error":"Pro plan required for susancloud models / susancloud 模型需 Pro 档"}`。第二轮经 Desktop UI 服务（`POST 127.0.0.1:3001/api/v1/chat/new`，走真实 `chat` 处理与 `getError`）→ 事件 `{"eventName":"error","error":"You've reached your usage limit, please upgrade to continue","code":"usage_limit_upgrade"}`。前提：运行时上要有该模型，测试时临时 `cp qwen3:0.6b susancloud/demo:latest`，测完已删除（原因见 O2） |
| E2E-5.4 | 打满限流 | 显示平台原文，没有升级条 | **通过** | 连续请求，第 59 次 → `429 {"error":"Rate limit exceeded / 请求过于频繁"}`，带响应头 `Retry-After: 31`；该串不含 402，也不含 `403`+`susancloud` → 走默认分支，不显示升级条 |
| E2E-5.5 | 对照网页 Billing 看设置页 | free 有 Upgrade，pro 没有；与 `/api/me` 的 `plan` 一致 | **通过**（第二轮补 pro） | free：`/api/billing/overview` 与 `/api/me` 均为 `plan=free`，设置页显示 Upgrade、Manage、Sign out。pro（库内 `plan=pro`、`plan_expires_at=now()+30d`）：`/api/me` → `"plan":"pro"`，设置页只剩 Manage、Sign out |

第二轮的平台测试数据改动已全部还原：测试账号 `plan=free`、`plan_expires_at=NULL`、`credit_balance=1000`；`official/qwen3.price_cny_per_m_token=NULL`；本机别名 `susancloud/demo:latest` 已删除。Pro 是直接改库构造的，没有走网页模拟购买，以免留下订单、订阅、流水记录。

---

## 四、问题项与修复记录

### F1（已修复）5.2 客户端计费接口路径写错

- 代码：`Session.GetOverview` 请求 `GET {BaseURL}/api/user/overview`（[http.go](file:///d:/projects/susanAssist/susanPlatform/Susan/auth/platform/http.go#L212-L238)）；
  平台实际端点是 `GET /api/billing/overview`（[billing.py](file:///d:/projects/susanAssist/susanPlatform/susan-platform/backend/app/api/billing.py#L45-L61)）。
- 实测（同一账号、同一 token、真实平台）：

```
GET /api/user/profile      -> 200 {"email":"susan_test@example.com","username":"测试一号"}
GET /api/billing/overview  -> 200 {"credit_balance":1000,"plan":"free","plan_expires_at":null,"billing_mode":"mock"}
GET /api/user/overview     -> 404                    # 客户端调用的路径
POST /api/me (daemon)      -> 200 {"id":...,"email":...,"name":...}   # 无 plan
```

- 传导链：`GetOverview` 对非 2xx 返回 `(nil, nil)` → `WhoamiHandler` 拿不到 overview → `plan = ""` →
  `api.UserResponse.Plan` 带 `omitempty`（[types.go](file:///d:/projects/susanAssist/susanPlatform/Susan/api/types.go#L1016-L1025)）→ 字段被整个省略。
- 后果：5.2（`plan` 取平台生效套餐）、5.3（Upgrade 按钮按套餐显隐）、5.5（free/pro 升级链接区分）在真实平台上全部失效；Pro 用户会看到 Upgrade 按钮并跳到 free 的升级页。
- 自动化为何没发现：[signin_device_test.go](file:///d:/projects/susanAssist/susanPlatform/Susan/server/signin_device_test.go) 的 mock 自行注册了 `/api/user/overview` 并默认返回 `{"plan":"free"}`，掩盖了路径不一致。
- **修复**：`GetOverview` 改指 `/api/billing/overview`（[http.go](file:///d:/projects/susanAssist/susanPlatform/Susan/auth/platform/http.go#L213-L217)）；mock 同步改为注册 `/api/billing/overview`（[signin_device_test.go](file:///d:/projects/susanAssist/susanPlatform/Susan/server/signin_device_test.go#L147)）；`Overview` 类型与函数的注释一并更正。
- **复验**：重新构建 `susan.exe` 并重启 daemon，`POST /api/me` →

```
200 {"id":"6d0c7421-0b16-4d97-be73-1bc643aab1cf","email":"susan_test@example.com","name":"测试一号","plan":"free"}
```

  与平台 `/api/billing/overview` 的 `plan=free` 一致；`go test ./server/ -run 'Whoami|Signin|Signout|Reporting'` 全 PASS。

### F2（已修复）上游回归测试文件被删除

- `server/routes_reporting_test.go` 在工作区被删除，留下未跟踪的 `server/routes_reporting_test.go.bak`。
- 该文件来自上游 #18829（`/api/balance`、`/api/usage` 透传及 `Retry-After`/`Cache-Control` 断言）。在本 fork 无法编译，原因有四处：`cloudProxySigningHost` → `cloudProxySigningHostOverride`、`writeTestOllamaPrivateKey` → `writeTestSusanPrivateKey`、`verifySignedOllamaRequest` → `verifySignedSusanRequest`，以及使用了 `OLLAMA_NO_CLOUD`。
- 建议：按改名后的符号回植并恢复，还原该回归覆盖。
- **修复**：按改名后的符号回植 `server/routes_reporting_test.go`（4 处改名 + `SUSAN_NO_CLOUD` + 补 `envconfig.ReloadServerConfig()` 以防 server.json 缓存干扰），并删除遗留的 `.bak`。
- **复验**：`go test ./server/ -run TestReporting -count=1 -v` → `TestReportingPassthrough`（balance/usage/upstream error/malformed query 共 4 个子用例）与 `TestReportingLocalErrors`（balance、usage 各 3 个子用例）全部 PASS。

---

## 五、残留风险

- **R1（已修复）**：`getError` 用 `strings.Contains(errStr,"403") && strings.Contains(errStr,"susancloud")`（[ui.go](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/ui.go#L641-L653)）区分「升级 403」与「其它 403」，但保留字 Push 的真实错误是 `403 Forbidden: namespace "susancloud" is reserved`（[IsReservedNamespace](file:///d:/projects/susanAssist/susanPlatform/Susan/types/model/reserved.go#L47-L58) 把任何 `susan*` 前缀视为保留），同样会命中该规则。已在规则上追加 `&& !strings.Contains(errStr, "is reserved")`（[ui.go](file:///d:/projects/susanAssist/susanPlatform/Susan/app/ui/ui.go#L648)），并把用例改为真实错误串。实测确认：`POST /api/push {"model":"susancloud/demo:latest"}` → `403 Forbidden` + body `{"error":"namespace \"susancloud\" is reserved"}`，现在不再触发升级条；而 `susancloud` 模型的 403 仍正常触发。
- **R2（已关闭）**：E2E-5.1 的「再出回复」与 E2E-5.2 的 402 已在第二轮补测通过（本机 daemon 充当运行时，临时定价与清零余额）。
- **R3**：`/api/balance`、`/api/usage` 透传到平台后返回 `404 {"detail":"Not Found"}`——本平台没有这两个原生路径（用量在 `/api/usage/summary`）。上游 #18829 的透传语义与本平台不匹配。不在批次 5 范围，不阻断。
- **R4（已修复）**：`DisplayUpgrade.tsx` 的 `plan` prop 已撤掉，恢复默认 `href = "https://susan.com/upgrade"`；链接仍由 `Chat.tsx` 按套餐传入。
- **R5（已修复）**：设置页 Upgrade 条件改回 `user?.plan === "free"`，`plan` 为空时不显示。同时把 Manage 链接恢复成 `https://susan.com/settings`（批次 5 写明不改链接）。
- **O1（观察，不阻断）**：`susan run <模型>:cloud` 会先尽力从 Registry 拉一个 cloud stub 清单，以便在 `susan ls` 中列出；Registry 写死 `registry.ollama.ai`，没有这些清单，所以 CLI 打出 `WARN failed to pull cloud stub ... file does not exist`，推理不受影响。属上游行为，批次 7 自建 `registry.susan.com` 并切换后可消除。
- **O2（已修复，作为 5.4 的补强）**：Desktop 的 `chat` 处理原本沿袭上游，`Show` 只要报错就转去 `Pull`。平台 `/api/show` 目前不做套餐检查，所以实际没出问题；但若日后平台在 show 上加检查，或 show 遇到 401、429、5xx，Desktop 都会误走 Pull，最后显示 `pull model manifest: file does not exist`。修复：
  - `ui.go` 的 `chat`：只有 `Show` 返回 404（或用户主动更新模型）才 `Pull`，其它错误直接交给 `getError` 显示。CLI 的 `showOrPullModel` 本来就只在 404 时拉取，未改。
  - `getError`：除了匹配错误文本，也看 `api.StatusError.StatusCode`。原因是非流式请求（如 `Show`）的错误文本里没有状态行，只有平台原文，比如 `Pro plan required for susancloud models / …`，仅靠字符串里的 "403" 会漏判。
  - 新增单测 `TestChatShowErrorOnlyPullsOnNotFound`，覆盖四种情况：show 返回 403 `susancloud`、402、429 时不拉取并给出正确事件，返回 404 时仍拉取。在修改前的代码上，前三种失败、404 通过；修改后全部通过。
  - 本机环境里运行时上没有该 `susancloud/*` 模型时，show 返回 404，仍会走 Pull，这是预期行为。
- 环境说明：本次未改动你的 `server.json`；`server.json.bak-batch5`（内容 `{"disable_ollama_cloud": true}`）是我此前尝试留下的备份，不需要可删除。
- **R6（已修复，按你的决定并入批次 5）**：从上游同步的测试里仍在用 `OLLAMA_*` 环境变量（`MODELS`、`HOST`、`DEBUG`、`GO_TEMPLATE`、`CREATE_REMOTE`、`LIBRARY_PATH`）。Susan 只读 `SUSAN_*`，所以这些设置不起作用，测试会写进真实的 `~/.susan/models`，或连到真实的 daemon。
  - 修复：在 `server/`、`manifest/`、`compatmigrate/`、`integration/`、`mlxrunner/` 的 17 个测试文件中改名 93 处，另有 `model_show_cache_test.go` 1 处、`cmd_test.go` 3 处。`envconfig/config_test.go` 的 `TestHostIgnoresOllamaHost` 本来就是要验证 `OLLAMA_HOST` 被忽略，保持不变。
  - 复验：`go test ./manifest/ ./compatmigrate/ ./server/ -count=1` 全部通过，此前 server 包的 12 个失败用例一并消失；跑前后 `~/.susan/models` 都是 241 个文件。`go test -c -tags integration ./integration/` 可以编译。`go vet -tags integration` 报的「在非测试 goroutine 里调用 Fatalf」是已有问题，与本次改名无关。

---

## 六、批次 5 追加：CLI 补齐（5.6～5.12）

**状态（2026-10-15）：通过。** E2E-5.6～5.11 全部通过（E2E-5.7 只跑了 free 分支，Pro 分支由单测覆盖）。E2E-5.9 首次运行发现缺陷 F3（未登录时没有登录链接），已修复并复验。随后追加 5.15：被动登录链接区分 CLI / Desktop，CLI 提示文案与验证码已验。

自动化结果：

| 用例 | 测试 | 结果 |
|---|---|---|
| AT-5.6 | `go test ./cmd/...`（8 个包） | 通过 |
| AT-5.7-a | `TestResolveRunModel_SubscriptionModelEmptyPlanAllowsSelection`、`TestLaunchAccountStateEmptyPlanIsUnknown` | 通过 |
| AT-5.7-b | `TestCloudAvailabilityBadgesEmptyPlan`、`TestEnsureCloudModelAccessEmptyPlan` | 通过 |
| AT-5.8-a/b/c | `TestModelRecommendationsCustomCloudHost`，以及原有的 `ModelRecommendations` 用例（URL 断言写成 `https://ollama.com/...`） | 通过 |
| AT-5.11-a/b/c | `TestCloudUpgradeURL`（9 个子用例）、`TestWithCloudUpgradeHintKeepsPlatformMessage` | 通过 |
| 回归 | `go test ./app/ui/ ./auth/platform/ ./envconfig/`；`go test ./server/ -run "ModelShowCache\|ModelRecommendations"`；`go vet ./cmd/ ./server/` | 通过；跑前后 `~/.susan/models/manifests` 文件数不变 |

实现要点：

- 5.9 `server/model_show_cache.go`：启动预热时，若 `/api/tags` 返回 404（`api.StatusError`），只记调试日志。
- 5.10 `cmd/cmd.go` 的 `ensureCloudStub`：错误为 `file does not exist`（Registry 上没有占位清单）时只记调试日志，其它失败仍打 WARN。判断依据是错误类型，而不是 `SUSAN_CLOUD_HOST`，因为这条日志由 CLI 进程打印，而 CLI 的 shell 通常没有设置这个变量。O1 随之消除。
- 5.11 `cmd/cloud_upgrade.go`：`withCloudUpgradeHint` 保留平台原文（`errors.As` 仍能取到 `api.StatusError`），再追加 `You've reached your usage limit, please upgrade to continue:` 和链接。402 时用 3 秒超时调 `Whoami`：Pro 用户指向 `launch.DefaultBillingURL`（`https://susan.com/settings/billing`），其余（含查不到套餐）指向 `https://susan.com/upgrade`。挂在三处：`susan run` 一次性生成、交互模式预加载、交互模式对话出错。

手工验收（2026-10-15）。CLI 用新编译的 `susan.exe`；平台运行在 :8000。E2E-5.6、5.7、5.8、5.10 跑的时候，daemon 还是旧版本（PID 21400，10-14 17:52 启动，带 `SUSAN_CLOUD_HOST`）；这几项只依赖 CLI，结论不受影响。E2E-5.11 跑之前已用新版本重启 daemon。

| 编号 | 结果 | 记录 |
|---|---|---|
| E2E-5.6 | 通过 | `susan run qwen3:0.6b:cloud "Reply with one word: hello"` → 回复 `hello`，没有 cloud stub 的 WARN。直接调 `POST /api/pull {"model":"qwen3:0.6b:cloud"}` 返回 `pull model manifest: file does not exist`，说明 CLI 走的是只记调试日志的分支（旧版本在这里会打 WARN，见 O1）。 |
| E2E-5.7 | 通过（free 分支） | 临时把 `official/qwen3` 单价设为 100、余额清零（已获同意）→ CLI 输出 `Error: 402 Payment Required: Insufficient credit balance / 余额不足，请充值`，随后是 `You've reached your usage limit, please upgrade to continue:` 和 `https://susan.com/upgrade`。跑完已恢复为单价为空、余额 1000。Pro 分支指向 `/settings/billing` 需要改套餐，这次没跑，由 `TestCloudUpgradeURL` 覆盖。 |
| E2E-5.10 | 通过 | 通过 daemon 连发 65 次 `/api/show`（带 `system` 字段绕过 daemon 的 show 缓存）：60 次 200、5 次 429。随后 `susan run` 输出 `Error: Rate limit exceeded / 请求过于频繁`，没有升级链接。这次是在 CLI 查询模型信息（show）那一步就被限流，没走到推理。 |
| E2E-5.8 | 通过 | 临时 `susan cp qwen3:0.6b susancloud/demo:latest` → `susan run susancloud/demo:cloud` 输出 `Error: 403 Forbidden: Pro plan required for susancloud models / susancloud 模型需 Pro 档`，随后是升级提示和 `https://susan.com/upgrade`。别名已删除。 |
| E2E-5.9 | 通过（先发现缺陷 F3，修复后复验） | 在 14344 端口起新版 daemon，用户目录和模型目录都指向临时空目录 → `You need to be signed in to Susan to run Cloud models.`，以及 `If your browser did not open, navigate to: http://localhost:3000/device?user_code=WQW5-ZYP9`。临时 daemon 已停，临时目录已删。 |
| E2E-5.11 | 通过 | 停掉旧 daemon，用新编译的 `susan.exe` 带 `SUSAN_CLOUD_HOST=http://localhost:8000`、`SUSAN_DEBUG=1` 在后台重启（日志在 `%TEMP%\susan-serve.log`）。日志：推荐请求发往 `http://localhost:8000/api/experimental/model-recommendations`，404 后 `skipping model recommendations refresh`，`consecutive_failures=0`；show 缓存的预热 `skipping ... because the cloud host does not list models`（5.9）；快照文件名是 `model-recommendations-localhost_8000.json`。`GET /api/experimental/model-recommendations` 只返回 `gemma4`、`qwen3.5`，云端模型为 0。重启后 `susan run qwen3:0.6b:cloud` 正常回复。 |

**F3（已修复）未登录时 CLI 没有登录链接**

- 现象：首次跑 E2E-5.9 时，CLI 只打印 `You need to be signed in to Susan to run Cloud models.`，没有链接。
- 原因：响应头是 `Server: uvicorn`，说明 401 的内容 `{"error":"unauthorized"}` 是平台返回、daemon 的转发代理原样转出去的。上游 ollama.com 的 401 自带 `signin_url`，平台的没有。daemon 自己能发起设备授权，直接调 `POST /api/signin/device` 就能拿到 `verification_uri_complete`，只是代理这条路径没有用上。
- 修复：`server/cloud_proxy.go` 新增 `writeUpstreamCloudUnauthorized`。上游 401 已带 `signin_url` 时原样转发（ollama.com 行为不变），否则改用现有的 `writeCloudUnauthorized`，由 daemon 发起设备授权，返回 `{"error":"unauthorized","signin_url":...}`。
- 单测：`TestCloudPassthroughUpstreamUnauthorizedAddsSigninURL` 覆盖「平台 401 没带链接 → 补上」和「上游自带链接 → 保留」两种情况；`go test ./server/` 整包通过。
- 附带影响：每次未登录请求云端模型，都会在平台上建一个待确认的设备授权码（15 分钟后过期）。这与 daemon 现有的 `/api/generate` 远程模型路径（`routes.go` 的 `signinURL()`）一致，而且同一时间只保留一个未过期的授权码，会被复用。

**环境观察（已清理）**：原先损坏的 `empty-src` 与一批测试夹具已从 `~/.susan/models` 清掉；保留 `qwen3:0.6b` 与 `susan/test:latest`。孤立清单备份在 `%TEMP%\susan-models-fixture-backup`。daemon 重启后正常 prune，不再报 corrupt manifests。

### 5.15 被动登录链接区分客户端（已完成）

- **问题**：所有 401 的 `signin_url` 都由 daemon 的 `signinURL()` 生成，客户端名写死为「Susan Desktop」。CLI（尤其无 Desktop 的 Linux 终端）确认后会被网页引导「返回 Susan Desktop」并尝试打开 `susan://`。
- **做法**：Desktop 发往 daemon 的请求统一带 `X-Susan-Client: desktop`（反向代理 + Desktop 自己的 HTTP 客户端）；daemon 按请求头 / User-Agent 选择「Susan Desktop」/「Susan CLI」/「Susan」。进行中的流程仍跨客户端复用（daemon 共用）。CLI 提示改为「Open this link to sign in, then run the command again」，并单独打印验证码。
- **自动化**：`TestSigninClientName`、`TestDesktopRequestsIdentifyClient`、`TestPrintSigninInstructions`；相关包测试通过。
- **手工**：未登录 daemon 上，CLI `susan run …:cloud` → 平台记录 `client_name=Susan CLI`；带 `X-Susan-Client: desktop` 的 `/api/me` → `Susan Desktop`。
- **与三条登录路径**：对齐对比表路径 (2) 的 `client_name` 要求。路径 (3)「终端直接登录」按 Ollama 做法记为后续待办：无人值守用 API Key 直连平台，本机 `:cloud` 仍走 Device Flow。

这几项有一个共同的观察：平台的原生 `/api/show` 也计入限流（免费档每分钟 60 次）。CLI 每跑一次会先请求 show，所以短时间内反复运行时，429 会先出现在 show 这一步。不影响结论。

---

## 七、结论

**第二轮结论（2026-10-14）：批次 5 通过。** 五条自动化与五条手工验收全部通过；R2 关闭，R4、R5、O2 已修复；R3、O1 不阻断。`product/productPlan.md` 批次 5 已标注验收结果。以下为首轮结论，保留备查。

**首轮：修复已完成并复验；但按 productPlan「没有部分通过」的口径，本批尚不能判为完全通过。**

原始判定（修复前）为不通过，原因见 F1。本轮处理：

- 自动化：AT-5.2-a/b、AT-5.4-a/b/c 全部通过；AT-5.2-a 已在真实平台复验（`/api/me` 返回 `plan=free`），不再是 mock 假通过；AT-5.4-c 的保留字用例改用真实错误串，覆盖缺口闭合。
- 手工：E2E-5.3、E2E-5.4、E2E-5.5 通过；E2E-5.1 部分验证。
- 回归覆盖：F2 的上游透传测试已回植并通过（新增 4 + 6 个子用例）。

**仍未验证（环境不具备，见 R2，不属代码缺陷）：**

- E2E-5.1 的「再出回复」；
- E2E-5.2 的 402 → 升级条。

代码侧已无阻断项，可验证项全部通过；在具备云端模型与定价模型的环境补跑上述两条后，即可判定本批通过。其余非阻断项见 R3～R5。
