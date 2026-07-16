.PHONY: check test shell-test fmt dev-browser-start dev-browser-stop dev-browser-restart dev-browser-status dev-browser-logs release-browser-start release-browser-stop release-browser-restart release-browser-status release-browser-logs

check: test shell-test

test:
	go test ./...

shell-test:
	../scripts/tests/wsl-process-test.sh

fmt:
	gofmt -w $$(find cmd internal -name '*.go')

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
