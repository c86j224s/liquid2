OPENAPI_JAVA_HOME ?= /opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home
OPENAPI_JAVA_BIN ?= /opt/homebrew/opt/openjdk@21/bin
DEMO_API_LABEL ?= com.c86j224s.liquid2.demo-api
DEMO_API_BIN ?= /tmp/liquid2-api
DEMO_API_BASE_URL ?= http://localhost:8080
DEMO_API_DB_PATH ?= /tmp/liquid2-demo.db

.PHONY: check backend-test frontend-analyze frontend-test shell-test size-audit generate openapi openapi-check openapi-client persistence-smoke demo demo-api demo-internal demo-client demo-stop fmt dev-browser-start dev-browser-stop dev-browser-restart dev-browser-status dev-browser-logs release-browser-start release-browser-stop release-browser-restart release-browser-status release-browser-logs

check: shell-test size-audit backend-test openapi-check frontend-analyze frontend-test

backend-test:
	go test ./...

frontend-analyze:
	cd client && flutter analyze

frontend-test:
	cd client && flutter test

shell-test:
	../scripts/tests/wsl-process-test.sh

size-audit:
	./scripts/check-file-sizes.sh

generate:
	$${SQLC:-$${HOME}/go/bin/sqlc} generate

openapi:
	mkdir -p api/openapi
	go run ./cmd/openapi > api/openapi/openapi-3.0.json

openapi-check:
	tmp=$$(mktemp); go run ./cmd/openapi > $$tmp; cmp -s $$tmp api/openapi/openapi-3.0.json || (echo "api/openapi/openapi-3.0.json is out of date; run make openapi" && rm -f $$tmp && exit 1); rm -f $$tmp

openapi-client:
	rm -rf client/api
	JAVA_HOME="$${JAVA_HOME:-$(OPENAPI_JAVA_HOME)}" PATH="$(OPENAPI_JAVA_BIN):$$PATH" npx @openapitools/openapi-generator-cli --openapitools /tmp/liquid2-openapitools.json generate -i api/openapi/openapi-3.0.json -g dart-dio -o client/api --skip-validate-spec --additional-properties=pubName=liquid2_api,pubVersion=0.1.0
	cd client/api && dart pub get && dart run build_runner build
	rm -rf client/api/.dart_tool client/api/pubspec.lock
	find client/api -type f -exec perl -0pi -e 's/[ \t]+$$//mg; s/\n+\z/\n/' {} +

persistence-smoke:
	./scripts/persistence-smoke.sh

demo: demo-api demo-client

demo-api:
	go build -o $(DEMO_API_BIN) ./cmd/api
	launchctl remove $(DEMO_API_LABEL) 2>/dev/null || true
	launchctl submit -l $(DEMO_API_LABEL) -- /usr/bin/env LIQUID2_SEED_DEMO=1 LIQUID2_LOG_FORMAT=text LIQUID2_DB_PATH=$(DEMO_API_DB_PATH) LIQUID2_JOBS_ENABLED=1 $(DEMO_API_BIN)
	sleep 1
	curl -fsS $(DEMO_API_BASE_URL)/api/v1/documents >/dev/null

demo-internal:
	./scripts/demo-internal.sh

demo-client:
	cd client && flutter build macos --debug --dart-define=LIQUID2_API_BASE_URL=$(DEMO_API_BASE_URL)
	pkill -f '/liquid2_client.app/Contents/MacOS/liquid2_client' 2>/dev/null || true
	sleep 1
	open -n client/build/macos/Build/Products/Debug/liquid2_client.app

demo-stop:
	launchctl remove $(DEMO_API_LABEL) 2>/dev/null || true

dev-browser-start:
	./scripts/dev-browser.sh start

dev-browser-stop:
	./scripts/dev-browser.sh stop

dev-browser-restart:
	./scripts/dev-browser.sh restart

dev-browser-status:
	./scripts/dev-browser.sh status

dev-browser-logs:
	./scripts/dev-browser.sh logs

release-browser-start:
	./scripts/release-browser.sh start

release-browser-stop:
	./scripts/release-browser.sh stop

release-browser-restart:
	./scripts/release-browser.sh restart

release-browser-status:
	./scripts/release-browser.sh status

release-browser-logs:
	./scripts/release-browser.sh logs

fmt:
	gofmt -w $$(find cmd internal -name '*.go')
