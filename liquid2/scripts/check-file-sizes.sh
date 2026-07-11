#!/bin/sh
set -eu

max_lines="${MAX_LINES:-200}"
tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

find . \
  \( -path './.git' -o \
     -path './client/.dart_tool' -o \
     -path './client/build' -o \
     -path './client/api' -o \
     -path './build' -o \
     -path './api/openapi' -o \
     -path './internal/storage/sqlite/sqlc' \) -prune -o \
  \( -name '*.go' -o -name '*.dart' \) -type f -print |
while IFS= read -r file; do
  case "$file" in
    *.pb.go|*.sql.go|*.g.dart|*.freezed.dart|*.gen.dart|*.mocks.dart|*/generated_plugin_registrant.dart)
      continue
      ;;
  esac

  lines="$(wc -l < "$file" | tr -d ' ')"
  if [ "$lines" -gt "$max_lines" ]; then
    printf '%s:%s lines exceeds %s\n' "$file" "$lines" "$max_lines" >> "$tmp_file"
  fi
done

if [ -s "$tmp_file" ]; then
  cat "$tmp_file"
  exit 1
fi

printf 'source file size audit passed: max %s lines\n' "$max_lines"
