#!/bin/bash
# Format .go files after edits.

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Skip non-.go files
if [[ "$FILE_PATH" != *.go ]]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR" || exit 0

gofmt -w "$FILE_PATH"
