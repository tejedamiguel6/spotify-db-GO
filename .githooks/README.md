# Git hooks

Security guardrails that run locally on `git commit`.

## Enable (once per clone)

```bash
git config core.hooksPath .githooks
```

## pre-commit

Blocks a commit when staged changes contain:
- secret-like values (API keys, tokens, client secrets, AWS keys, private keys)
- debug dumps (`DEBUG_*.json`)
- the compiled `server` binary
- any file larger than 1 MB

Bypass a false positive with `git commit --no-verify` (review first).
