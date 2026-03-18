#!/bin/sh
# PreToolUse hook for Read tool.
# Reads hook JSON from stdin, runs ctxguard check-file, returns additionalContext.

set -e

# Read stdin (hook input JSON)
INPUT=$(cat -)

# Extract file_path from tool_input using lightweight JSON parsing.
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"file_path"[[:space:]]*:[[:space:]]*"//;s/"$//')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Extract repo root from cwd.
CWD=$(echo "$INPUT" | grep -o '"cwd"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"cwd"[[:space:]]*:[[:space:]]*"//;s/"$//')

# Run ctxguard check-file.
RESULT=$(ctxguard check-file --path "$FILE_PATH" --repo "$CWD" 2>/dev/null) || exit 0

# If there's a warning, inject it as additionalContext.
if [ -n "$RESULT" ]; then
  # Escape for JSON
  ESCAPED=$(echo "$RESULT" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n' | sed 's/\\n$//')
  echo "{\"hookSpecificOutput\":{\"additionalContext\":\"$ESCAPED\"}}"
fi
