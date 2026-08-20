# AGENTS.md

Go 1.25 (module `github.com/daffadon/digihome`), fx DI, chi router, pgvector/Postgres, Ollama.
No tests, no CI, no README — don't go looking for them.

## Commands
- Run: `make run` (needs `make dev-start` first: Postgres/pgvector, rustfs S3, Ollama via docker compose; `ollama-init` pulls `bge-m3` + `qwen3.5:0.8b`).
- Verify: `go build ./...` then `go vet ./...` (no test suite).
- `make docker-build` / `make build-binary` are broken — they call `script/*.sh` which doesn't exist.
- `make migrate-create name=<name>` → `migrations/0000NN_name.{up,down}.sql` (requires `migrate` CLI + `DB_USER/DB_PASSWORD/DB_NAME` in `.env`); apply with `make migrate-up`.

## Codegen & schema (two sources of truth, keep in sync)
- Edit `chat.sql`/`rag.sql` AND `internal/domain/repository/db_schema/db.schema.sql`, then run `sqlc generate` (sqlc.yaml maps each `.sql` to a committed package: `chat_repository`, `rag_repository`).
- NEVER hand-edit generated `*.sql.go`, `models.go`, `db.go`.
- DB changes also need a migration file in `migrations/`.

## Config
- viper reads `config.local.yaml` (dev) / `config.yaml` (prod) / `config.test.yaml` (ENV=test). `config*.yaml` is gitignored — no committed template, only `.env.example`.
- Env vars override keys with `.`→`_` (e.g. `llm.chat.model` ← `LLM_CHAT_MODEL`).
- LLM asymmetry: embeddings (`bge-m3`, endpoint) are hardcoded constants in `internal/constant/llm.go` (constant misspelled `DEFAULT_EMBED_ENDOPINT`); chat (`qwen3.5:0.8b`, endpoint, num_ctx, reserve_reply_tokens, top_k_history, recent_messages) is viper-driven via `chatConfigFromViper` in `internal/domain/services/chat.go`.

## Wiring
- fx: new constructors go in `fx.Provide`, route registration in `fx.Invoke` in `cmd/main.go` (`Register*Route(r chi.Router, h ...)`).
- Layering: handler → services (interfaces holding sqlc `*Queries`) → repository. Reusable infra clients live in `internal/pkg` as plain funcs (e.g. `pkg.Embed`, `pkg.Chat`).

## Conventions & gotchas
- Handlers must `return` immediately after `pkg.ReturnError`/`pkg.ReturnSuccess` — they write the response; falling through double-writes. (Current `internal/domain/handler/chat.go` `Chat()` violates this.)
- `gofmt -l` flags every file (no trailing newlines repo-wide by convention) — do NOT reformat.
- Messages with NULL `embedding` are invisible to `SearchSimilarMessages`; only rows created after migration 000004 are searchable until backfilled.

## Commits
- Conventional Commits: `type(scope): description` (e.g. `feat(service): add chat semantic context`); lowercase, imperative, no period. Scopes seen: service, repo, pkg, handler, schema, constant, migrations, docker, server.