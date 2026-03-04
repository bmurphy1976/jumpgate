#!/bin/bash
# Run go vet after .go file edits for immediate feedback.

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Skip non-.go files
if [[ "$FILE_PATH" != *.go ]]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR" || exit 0

OUTPUT=$(go vet ./... 2>&1)
if [ $? -ne 0 ]; then
  echo "$OUTPUT" >&2
  exit 2
fi
