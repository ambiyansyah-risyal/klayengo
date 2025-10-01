# Git Hooks Configuration

This project uses Git hooks to enforce code quality and commit message standards.

## Setting Up Git Hooks

Git hooks are stored in the `githooks/` directory and need to be installed manually:

```bash
./install-hooks.sh
```

This script copies the hooks from `githooks/` to your local `.git/hooks/` directory and makes them executable.

## Pre-commit Hook

The pre-commit hook runs `act` to execute GitHub Actions locally before allowing a commit. This ensures that:
- All tests pass
- Code quality checks pass
- Linting passes
- The code is in a deployable state

## Commit-msg Hook

The commit-msg hook validates that commit messages follow these rules:
1. **Prefix required**: Must follow conventional commits format `<type>(optional scope): description`
   - Examples: `feat: add new feature`, `fix: resolve issue`, `docs: update readme`
   - Valid types: `build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test|wip`

2. **Single line**: Commit messages should be concise and fit on a single line (excluding comments)

3. **No AI keywords**: Commit messages must not contain AI-related terms like:
   - `codex`, `copilot`, `claude`, `qwen`, `ai`, `artificial intelligence`, `machine learning`, `ml`, `chatgpt`, `gpt`, `openai`, `anthropic`

## Valid Commit Message Examples

```
feat: add user authentication
fix: resolve memory leak in cache implementation  
docs: update API documentation
refactor: improve error handling in client
test: add unit tests for rate limiter
chore: update dependencies
```

## Invalid Commit Message Examples

```
Add new feature  # (missing prefix)
fix: resolve issue
with more details  # (multi-line)
feat: add new feature with AI assistance  # (contains AI keyword)
```

## Manual Testing

If you want to test the hooks manually:
- For pre-commit: Run `act` in the project root
- For commit-msg: Create a test commit message file and run the validation script