# Repository Guidelines

## Project Shape

This repository is `meta-file-system`, a Go service for MetaID file upload and indexing. It has two main binaries:

- `cmd/uploader`: upload API, async upload tasks, multipart upload support, and the static upload page on port `7282`.
- `cmd/indexer`: blockchain scanner plus query/download API on port `7281`.

Core directories:

- `controller`: Gin routers, handlers, response helpers, CORS, Swagger wiring, and static page routing.
- `service/upload_service`: upload task processing and cleanup.
- `service/indexer_service`: indexing, migration, query, sync status, and rescan behavior.
- `indexer`: block scanning, MetaID parsing, multi-chain coordination, and ZMQ support.
- `database`: database interface and MySQL/Pebble adapters; uploader DB is MySQL-only.
- `model` and `model/dao`: persisted data models and DAO wrappers.
- `storage`: local, OSS, S3, and MinIO storage implementations.
- `web`: static browser UI plus vendored/minified browser libraries copied from npm packages.
- `docs/indexer` and `docs/uploader`: generated Swagger docs.
- `deploy`: Dockerfiles and docker-compose definitions.

## Commands

- Install Go dependencies: `make deps`
- Run all Go tests: `go test ./...` or `make test`
- Build both binaries: `make build`
- Run uploader locally: `go run cmd/uploader/main.go --env=loc` or `make run-uploader`
- Run indexer locally: `go run cmd/indexer/main.go --env=loc` or `make run-indexer`
- Generate Swagger docs: `make swagger`
- Install the Swagger generator: `make install-swag`
- Refresh browser libraries: `cd web && npm install` or `cd web && npm run copy-libs`
- Docker full stack: `make docker-up`, `make docker-down`, `make docker-logs`

`web/package.json` has no useful test script; it intentionally exits with an error.

## Config And Runtime Notes

Configuration is selected by `--env`, not by a `--config` flag:

- `--env=loc` loads `conf/conf_loc.yaml`
- `--env=mainnet` loads `conf/conf_pro.yaml`
- `--env=testnet` expects `conf/conf_test.yaml`
- `--env=example` loads `conf/conf_example.yaml`

The uploader defaults to `loc`; the indexer defaults to `mainnet`. Be explicit with `--env=loc` for local runs.

Tracked config is only `conf/conf_example.yaml`. Files matching `conf/conf_*` except the example, root `conf_pro.yaml`, `ops_local/`, `data/`, and `bin/` are ignored and may contain local or deployment-specific state. Do not edit or commit those unless the user asks specifically.

Database schema files are `sql/indexer.sql` and `sql/uploader.sql`. Some README snippets mention `scripts/init.sql` or `--config=...`; verify against current code before relying on those examples.

Common runtime endpoints:

- Uploader health: `http://localhost:7282/health`
- Uploader Swagger: `http://localhost:7282/swagger/index.html`
- Indexer health: `http://localhost:7281/health`
- Indexer Swagger: `http://localhost:7281/swagger/index.html`

## Development Rules

- Keep API changes layered through router, handler, service, DAO, and database adapter as appropriate.
- If routes, request/response structs, or Swagger annotations change, regenerate `docs/indexer/*` or `docs/uploader/*` with `make swagger`.
- Preserve existing compatibility routes unless the user explicitly asks to remove them.
- When changing indexing behavior, check both MySQL and Pebble paths where the database interface requires both.
- When changing upload behavior, account for chain-specific params from `uploader.chains`; DOGE uses byte-sized chunks while MVC uses MB-sized chunks.
- Avoid broad rewrites of generated Swagger files unless annotations actually changed.
- Prefer focused Go tests near the touched package, then run `go test ./...` when the change has cross-package impact.

## Commit and Merge Rules

- If you notice unfamiliar or unrelated file changes, continue working and stay focused on your own scoped edits unless the user asks you to inspect them.
- For each completed round that modifies existing code/docs or adds new code/docs, automatically stage and commit only the files you changed and understand.
- For deletion changes, wait until the user explicitly says "commit" before staging and committing those deletions.
- Prefer small, frequent commits. Commit each independent, verifiable unit of work as soon as it is complete.
- For every modification or newly added feature, create one commit.
- For every commit, use the `metabot-post-buzz` skill to post a detailed development-journal entry on-chain describing the change.
- Use commit messages in the format `<type>: <short description>`, where `<type>` is one of `feat`, `fix`, `refactor`, `docs`, or `chore`.
- Before committing, make sure the relevant local tests or verification steps pass for your changes.
- When merging completed work into `main`, use `git merge --no-ff` to preserve the feature merge point.

## Behavioral Guidelines

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.
