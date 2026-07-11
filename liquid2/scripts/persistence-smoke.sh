#!/bin/sh
set -eu

addr="${LIQUID2_SMOKE_ADDR:-127.0.0.1:18083}"
base_url="http://${addr}"
tmp_dir="$(mktemp -d)"
bin_path="${tmp_dir}/liquid2-api"
db_path="${LIQUID2_SMOKE_DB_PATH:-${tmp_dir}/liquid2.db}"
log_path="${tmp_dir}/liquid2-api.log"
pid=""

cleanup() {
  if [ -n "$pid" ]; then
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

go build -o "$bin_path" ./cmd/api

assert_smoke_addr_free() {
  if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
    echo "Smoke address is already serving ${base_url}/healthz; set LIQUID2_SMOKE_ADDR to a free port." >&2
    return 1
  fi
}

start_api() {
  assert_smoke_addr_free
  LIQUID2_ADDR="$addr" \
    LIQUID2_DB_PATH="$db_path" \
    LIQUID2_LOG_FORMAT="${LIQUID2_LOG_FORMAT:-text}" \
    "$bin_path" >"$log_path" 2>&1 &
  pid="$!"
  for _ in $(seq 1 80); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      echo "API process exited before becoming ready; log follows:" >&2
      cat "$log_path" >&2 || true
      wait "$pid" >/dev/null 2>&1 || true
      pid=""
      return 1
    fi
    if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  echo "API did not become ready; log follows:" >&2
  cat "$log_path" >&2 || true
  return 1
}

stop_api() {
  kill "$pid" >/dev/null 2>&1 || true
  wait "$pid" >/dev/null 2>&1 || true
  pid=""
}

json_id() {
  # Keep smoke dependencies minimal; these sed matchers assume compact API JSON.
  sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -n 1
}

json_document_id() {
  sed -n 's/.*"document":{"id":"\([^"]*\)".*/\1/p' | head -n 1
}

start_api
folder_response="$(curl -fsS -H 'Content-Type: application/json' \
  -d '{"name":"Research","sortOrder":10}' \
  "${base_url}/api/v1/folders")"
folder_id="$(printf '%s' "$folder_response" | json_id)"
tag_response="$(curl -fsS -H 'Content-Type: application/json' \
  -d '{"name":"SQLite"}' \
  "${base_url}/api/v1/tags")"
tag_id="$(printf '%s' "$tag_response" | json_id)"
doc_response="$(curl -fsS -H 'Content-Type: application/json' \
  -d "{\"url\":\"https://example.com/persist\",\"title\":\"Persistent Link\",\"folderId\":\"${folder_id}\",\"tagIds\":[\"${tag_id}\"]}" \
  "${base_url}/api/v1/documents/bookmark")"
doc_id="$(printf '%s' "$doc_response" | json_document_id)"
note_response="$(curl -fsS -H 'Content-Type: application/json' \
  -d '{"body":"Smoke note","format":"text"}' \
  "${base_url}/api/v1/documents/${doc_id}/notes")"
note_id="$(printf '%s' "$note_response" | json_id)"
curl -fsS \
  -F title='Uploaded Smoke' \
  -F file=@-';filename=smoke.txt;type=text/plain' \
  "${base_url}/api/v1/documents/upload" <<'EOF' >/dev/null
Uploaded smoke body
EOF
stop_api

start_api
curl -fsS "${base_url}/api/v1/documents/${doc_id}" | grep -q '"title":"Persistent Link"'
curl -fsS "${base_url}/api/v1/documents/${doc_id}" | grep -q '"slug":"sqlite"'
curl -fsS "${base_url}/api/v1/documents/${doc_id}/notes" | grep -q "\"id\":\"${note_id}\""
curl -fsS "${base_url}/api/v1/folders" | grep -q '"name":"Research"'
curl -fsS "${base_url}/api/v1/tags" | grep -q '"slug":"sqlite"'
curl -fsS "${base_url}/api/v1/documents?kind=uploaded_file" | grep -q '"title":"Uploaded Smoke"'

echo "persistence smoke passed: ${db_path}"
