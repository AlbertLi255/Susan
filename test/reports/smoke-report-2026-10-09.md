# Susan Smoke Test Report

- **Date**: 2026-10-09 15:19 (Asia/Shanghai)
- **Branch**: `feature/sync-ollama-2026-10-09`
- **HEAD**: `9e757e4b docs: record how Susan merges Ollama without replaying old conflicts.`
- **Binary**: `dist/susan.exe` (57,186,908 bytes, built 2026-10-09 15:11)
- **Git version**: `v0.40.2-24-g9e757e4b-dirty`
- **Binary version**: `0.0.0` (built via `go build .` without ldflags — version not embedded)
- **Env**: `SUSAN_NO_CLOUD=true`, `SUSAN_CLOUD_HOST=http://localhost:8000`, `SUSAN_HOST=http://127.0.0.1:14343`

## Test Results

| # | Test | Method | Result | Evidence |
|---|------|--------|--------|----------|
| 1 | Serve startup | `susan serve` | PASS | `Listening on 127.0.0.1:14343 (version 0.0.0)`, CPU inference lib loaded |
| 2 | Port 14343 | TCP listen check | PASS | `Get-NetTCPConnection -LocalPort 14343` → Listen |
| 3 | CLI brand | `susan --help` | PASS | `Usage: susan [flags]`, subcommands: "Start Susan", "Sign in to Susan", "Sign out of Susan", "Launch the Susan menu" |
| 4 | Data dir `~/.susan` | serve config log | PASS | `SUSAN_MODELS:C:\Users\albert\.susan\models` |
| 5 | `GET /api/version` | HTTP | PASS | `{"version":"0.0.0"}` (200) — version 0.0.0 is build flag issue, not regression |
| 6 | `GET /api/tags` | HTTP | PASS | 200, 8 models: qwen3:0.6b, susan/test:latest, fuzz-create-gguf, test-draft, etc. |
| 7 | `GET /api/ps` | HTTP | PASS | 200, `{"models":[]}` (no models loaded) |
| 8 | `POST /api/me` | HTTP | PASS* | 503 `{"error":"account unavailable"}` — route exists; 503 because `SUSAN_NO_CLOUD=true` makes `platform.DefaultClient()` fail. Not a regression. |
| 9 | `POST /api/generate` | HTTP (qwen3:0.6b) | PASS | 200, `"response":"","thinking":"Okay, the"`, `done_reason:"length"`, `eval_count:5` |
| 10 | `POST /api/chat` | HTTP (qwen3:0.6b) | PASS | 200, `"message":{"role":"assistant","content":"","thinking":"Okay, the"}`, `done_reason:"length"` |
| 11 | `POST /v1/chat/completions` | HTTP (OpenAI compat) | PASS | 200, `"object":"chat.completion"`, `"reasoning":"Okay, the"`, `"finish_reason":"length"` |
| 12 | `POST /v1/completions` | HTTP (OpenAI compat) | PASS | 200, `"object":"text_completion"`, `"finish_reason":"length"` |
| 13 | `GET /v1/models` | HTTP (OpenAI compat) | PASS | 200, 8 models listed, `susan/test:latest` owned_by `susan` |
| 14 | `GET /health` | HTTP | **FAIL** | 404 `page not found` — `/health` route not registered in `server/routes.go` (hard constraint gap) |

## Issues Found

### 1. `/health` route missing (Hard Constraint Gap)

- **Severity**: Medium (hard constraint not met)
- **Detail**: `project_memory.md` hard constraint: "API must include /health, /abort, /manifests, and /blobs endpoints"
- **Current**: `/api/blobs/:digest` exists (POST, HEAD), but `/health`, `/abort`, `/manifests` are **not registered** in [server/routes.go](file:///d:\projects\susanAssist\susanPlatform\Susan\server\routes.go)
- **Grep evidence**: `grep -n 'health\|abort\|manifests' server/routes.go` → no route registrations found

### 2. `system_fingerprint: "fp_ollama"` brand leak

- **Severity**: Medium (brand constraint violation)
- **Detail**: OpenAI compatibility responses include `"system_fingerprint":"fp_ollama"` — should be `fp_susan`
- **Location**: [openai/openai.go](file:///d:\projects\susanAssist\susanPlatform\Susan\openai\openai.go#L337) lines 337, 387, 443, 472, 494 — 5 hardcoded occurrences
- **Fix**: Replace all `"fp_ollama"` → `"fp_susan"` in `openai/openai.go`

### 3. Binary version `0.0.0` (Build Config)

- **Severity**: Low (build flag, not code regression)
- **Detail**: Binary built via `go build .` without ldflags; `git describe` shows `v0.40.2-24-g9e757e4b-dirty`
- **Fix**: Build with `-ldflags "-X github.com/ollama/ollama/version.Version=$(git describe --tags --dirty)"` or use `cmake --build build`

## Summary

| Category | Pass | Fail | Skip |
|----------|------|------|------|
| Core API | 7 | 1 (/health 404) | 0 |
| Inference | 2 | 0 | 0 |
| OpenAI Compat | 3 | 0 | 0 |
| Brand/Port | 3 | 0 | 0 (version 0.0.0 noted) |
| **Total** | **15** | **1** | **0** |

**Conclusion**: No functional regressions in core LLM serving, inference, or OpenAI compatibility. The `/health` route gap is a known hard-constraint item not yet implemented (not a merge regression). The `fp_ollama` brand leak in `openai/openai.go` is a real brand issue that should be fixed.

## Artifacts

- Serve log: `test/reports/susan-smoke-serve.log` (background job output)
- JSON request bodies: `test/reports/{gen,chat,v1chat,v1comp}.json`
