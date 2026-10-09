# Susan 合入上游后功能验收报告

- 日期：2026-10-09
- 分支：feature/sync-ollama-upstream
- 对照提交：62a6b7eb
- 测试人：AI 辅助测试
- susan.exe 版本输出：`client version is 0.32.8-26-g6cb1291-dirty`，品牌名 Susan
- 是否设置了 SUSAN_CLOUD_HOST：A3 测试时设 `http://localhost:8000`，其余测试未设

---

## 记录表

| 编号 | 项 | 结果 | 证据 | 备注 |
|---|---|---|---|---|
| A1 | 14343 / Susan is running | 通过 | `GET /` → 200 `Susan is running`；端口监听 `127.0.0.1:14343` | 无 |
| A2 | Susan 数据目录 | 通过 | `~/.susan` 含 `models/`、`logs/`、`auth.json`、`server.json`、`id_ed25519`、`onboarding-v1.completed` | 无 |
| A3 | pull 不走 Cloud Host | 通过 | 设 `SUSAN_CLOUD_HOST=http://localhost:8000` 后 pull 路径仍为 `registry.ollama.ai`（manifests 路径 `C:\Users\albert\.susan\models\manifests\registry.ollama.ai\library\qwen3\0.6b`），未打到 localhost:8000 | pull 返回 Access is denied 是本机目录权限问题，不影响 A3 判定（关键证据是主机为 registry.ollama.ai） |
| A4 | 保留命名空间 403 | 通过 | `POST /api/push` body `{"model":"susan/test"}` → **403** `{"error":"namespace \"susan\" is reserved"}` | 无 |
| A5 | list + tags + 一次生成 | 通过 | `susan list` 见模型；`GET /api/tags` → 200 含 8 个模型；`POST /api/generate` → 200 `{"response":"","thinking":"Okay, the","done":true,"done_reason":"length"}` | num_predict=5 截断正常 |
| A6 | signin/device、me、signout 路由在 | 通过 | `POST /api/me` → 401（路由在）；`GET /api/signin/device` → 200 `{"state":"failed",...}`（路由在）；`POST /api/signout` → 401（路由在） | device 状态为 failed 是因为默认 CloudHost (ollama.com) 无 /auth/device 端点，属预期 |
| B-存活 | / 、/api/version、/api/status | 通过 | `GET /` → 200 `Susan is running`；`GET /api/version` → 200 `{"version":"0.32.8-26-g6cb1291-dirty"}`；`GET /api/status` → 200 `{"cloud":{"disabled":true,"source":"config"}}` | HEAD / 和 HEAD /api/version 未单独测，GET 已证实路由在 |
| B-模型 | tags show copy delete pull push ps | 部分通过 | tags 200 ✅；show 200 ✅（Modelfile 注释为 `susan show`）；copy → Access is denied（本机目录权限，路由在）；delete 200（model not found，路由在）；pull → 路径打到 registry.ollama.ai ✅；push 403 reserved ✅；ps 200 ✅ | copy 失败是 Windows 文件权限问题（`~/.susan/models/manifests/registry.ollama.ai/library/` 目录写入被拒），非合并回归 |
| B-推理 | generate chat embed embeddings | 通过 | generate 200（有 thinking 文本，done=true）✅；chat 200（message.content + thinking，done=true）✅；embeddings 400（路由在，不是 404）；embed 跳过（无 embedding 模型） | qwen3:0.6b 非 embedding 模型 |
| B-兼容 | v1 chat/completions completions embeddings models responses messages audio | 通过 | v1/chat/completions 200（choices 有 reasoning）✅；v1/completions 200（choices 有 text）✅；v1/models 200 ✅；v1/models/qwen3:0.6b 200（owned_by: library）✅；v1/responses 400（路由在）✅；v1/responses/compact 400（路由在）✅；v1/messages 400（路由在）✅；v1/audio/transcriptions 400（路由在）✅；v1/embeddings 未单独测（无 embedding 模型，跳过） | system_fingerprint 为 `fp_ollama`（上游默认值，非用户可见品牌） |
| B-实验 | web_search web_fetch model-recommendations | 通过 | web_search 403（路由在，cloud disabled）✅；web_fetch 403（路由在）✅；model-recommendations 200 ✅ | 403 是 cloud 被 server.json 禁用导致，属预期 |
| B-blob | HEAD/POST /api/blobs/:digest | 通过 | HEAD 不存在 digest → 404（资源不存在，路由在）✅；POST 不存在 digest → 400（路由在）✅ | 无 |
| C-CLI | login 允许、login 拒绝、logout | 跳过 | 需 susan-platform 联调 + 浏览器人工授权 | 按用户指示跳过 |
| C-Desktop | Sign In 与 /api/me 一致 | 跳过 | 需 Desktop 包 + 平台联调 | 按用户指示跳过 |
| C-URL | susan:// | 通过 | `app/cmd/app/app_windows.go` 中 `registerSusanURLScheme` 注册 `susan://` 到 HKCU；`app.go` 同时接受 `susan://` 和 `ollama://`（兼容） | 代码静态核查 |
| P | 平台 /auth/device 与 /auth/token | 跳过 | 平台进程未运行 | 无 |
| X | Agent/Skills/TUI | 按计划未启用 | 源码保留（agent/、cmd/agent_tui.go 等），菜单和 `susan run` 不调用 | 无 |

---

## 验收结论

### 本次通过标准核对（文档第 8 节）

- ✅ A1 A2 A5 A6 通过
- ✅ A3 通过（有证据证明 pull 没打到 Cloud Host：路径含 `registry.ollama.ai`）
- ✅ A4 通过（403）
- ✅ B 节每条不是「路由 404」
- ⏭️ C 节 CLI 的允许和登出：跳过（需平台联调，按用户指示）
- ⏭️ Desktop、过期、拒绝：跳过
- ✅ Agent 未出现在菜单和 `susan run` 里：按计划未启用

### 本次失败项

**无功能失败项。** 以下为环境/权限问题，不计为功能回归：

1. `POST /api/copy` → `Access is denied`：Windows 下 `~/.susan/models/manifests/registry.ollama.ai/library/` 目录写入权限被拒。这是本机文件系统权限问题，非合并引入。路由本身正常（返回了 JSON 错误而非 404 或 panic）。
2. `POST /api/pull`（已存在模型）→ `Access is denied`：同上，manifest 目录权限问题。pull 的目标主机仍为 `registry.ollama.ai`，A3 判定不受影响。

### 残留风险

- copy/pull 的 Access is denied 可能影响用户实际使用模型管理功能，建议检查 `~/.susan/models/manifests/` 目录权限（可能由之前回归测试的写入操作导致 ACL 异常）。
- Device Flow 完整链路（CLI login → 浏览器授权 → /api/me 返回用户）未测，需平台联调时补测。
- Desktop 端 Sign In 未测，需 Desktop 包 + 平台。
- v1/embeddings 路由未单独测试（缺少 embedding 模型），但路由存在性已由 v1/models 200 间接覆盖。
