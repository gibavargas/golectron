# AGENTS

## Codex CLI workflow

For longer-running or reusable Codex work, enable Codex `/goal` locally:

1. Update Codex CLI to `0.128.0` or newer and confirm with `codex --version`.
2. Edit `~/.codex/config.toml`.
3. Under the existing `[features]` table, add:

   ```toml
   [features]
   goals = true
   ```

   If `[features]` already exists, add only `goals = true`; do not create a
   duplicate TOML table.
4. Restart Codex.
5. Type `/goal` in the Codex session.

`~/.codex/config.toml` is local user/machine configuration and must not be
committed to this repository. `/goal` is a workflow aid; project correctness
must still be verified with normal tests such as `go test ./...`.
