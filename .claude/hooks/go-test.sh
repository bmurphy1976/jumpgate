#!/bin/bash
# Run make test when Claude finishes, if Go/templ files were modified.

INPUT=$(cat)
STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')

# Prevent infinite loop: skip if re-running after a failed stop hook
if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR" || exit 0

# Only run if .go or .templ files were modified
CHANGED=$(git status --porcelain 2>/dev/null | grep -E '\.(go|templ)$')
if [ -z "$CHANGED" ]; then
  exit 0
fi

OUTPUT=$(make test 2>&1)
if [ $? -ne 0 ]; then
  echo "Tests failed. Fix the issues before finishing." >&2
  echo "$OUTPUT" >&2
  exit 2
fi
