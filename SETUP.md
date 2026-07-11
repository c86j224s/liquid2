# Agent Setup Guide

이 문서는 새 에이전트 세션이 이 저장소를 준비, 최신화, 빌드, 실행할 때 따르는
절차다. 에이전트는 임의의 `go build`, `launchctl`, `flutter run` 조합을 만들지 말고
저장소 스크립트를 우선 사용한다.

## 적용 범위

이 저장소에는 두 제품이 있다.

- `liquid2/`: Go API, Flutter web client, SQLite 저장소.
- `plasma/`: Go browser/API server, SQLite 저장소, Liquid2 connector.

개발 서버는 6000번대 포트를 쓰고, 로컬 릴리즈 서버는 3000번대 포트를 쓴다.

- Liquid2 development: web `6001`, API `6011`
- Plasma development: web/API `6002`
- Liquid2 release: web `3001`, API `3011`
- Plasma release: web/API `3002`

런타임 설정 파일과 DB는 커밋하지 않는다. 설정 규칙은
[`docs/configuration.md`](docs/configuration.md)를 따른다.

## 준비 원칙

에이전트는 다음 순서로 진행한다.

1. 현재 브랜치와 작업트리 상태를 확인한다.
2. 필요한 도구가 설치되어 있는지 확인하고, 없으면 설치한다.
3. 이미 준비된 환경이면 최신 `main`으로 갱신하고 스크립트 `build`를 다시 실행한다.
4. 스크립트 `status`로 실행 가능 상태를 확인한다.
5. 준비가 끝나면 아래 "완료 보고에 포함할 사용법" 명령 목록을 사용자에게 출력한다.

작업트리에 사용자가 만든 미커밋 변경이 있으면 `git pull`, dependency update,
formatting, generated file 변경을 진행하기 전에 사용자에게 보고한다. 사용자 변경을
덮어쓰지 않는다.

## 필수 도구

macOS 로컬 실행을 기준으로 한다. 다음 도구가 필요하다.

- `git`
- `go` 1.26.5 이상
- `flutter` with Dart SDK 3.12 이상
- `make`
- `curl`
- macOS `launchctl`, `plutil`

확인 명령:

```sh
git --version
go version
flutter --version
make --version
curl --version
launchctl version
plutil -help >/dev/null
```

도구가 없고 Homebrew를 사용할 수 있으면 다음처럼 설치한다.

```sh
brew install go flutter
flutter doctor
flutter config --enable-web
flutter precache --web
```

`flutter doctor`가 Xcode, CocoaPods, Android toolchain 같은 항목을 경고할 수 있다.
이 저장소의 브라우저 서버 준비에는 Flutter web이 핵심이다. Flutter web 실행을 막는
오류가 아니면 해당 경고를 별도로 보고하고 계속 진행할 수 있다.

## 처음 준비

저장소 루트에서 실행한다.

```sh
pwd
git status --short --branch
```

`main`이 아니면 사용자 요청이 명확한 경우에만 전환한다.

```sh
git switch main
git pull --ff-only
```

Go 모듈과 Flutter 의존성을 확인한다.

```sh
(cd liquid2 && go test ./...)
(cd plasma && go test ./...)
(cd liquid2/client && flutter pub get)
```

더 엄격한 전체 확인이 필요하면 다음을 실행한다.

```sh
make -C liquid2 check
make -C plasma check
```

`make -C liquid2 check`는 Flutter analyze/test까지 포함하므로 시간이 더 걸릴 수
있다.

## 이미 준비된 환경 최신화

이 과정을 이미 한 번 수행한 환경에서는 다시 설치부터 시작하지 않는다. 먼저 현재
상태를 확인한다.

```sh
git status --short --branch
./dev-browser.sh status
./release-browser.sh status
```

작업트리가 깨끗하고 `main...origin/main`이면 최신 코드가 이미 반영된 상태다. 그래도
스크립트와 바이너리를 최신 코드로 다시 맞추려면 다음을 실행한다.

```sh
./dev-browser.sh build
./release-browser.sh build
```

원격 `main`이 앞서 있으면 다음 순서로 갱신한다.

```sh
git pull --ff-only
./dev-browser.sh build
./release-browser.sh build
./dev-browser.sh status
./release-browser.sh status
```

`git pull --ff-only`가 실패하면 로컬 변경이나 divergent history가 있다는 뜻이다.
이때는 병합이나 reset을 임의로 하지 말고 사용자에게 상태를 보고한다.

## 빌드와 실행

루트 스크립트를 사용하면 두 제품을 함께 제어한다.

```sh
./dev-browser.sh build
./dev-browser.sh start
./dev-browser.sh status
./dev-browser.sh logs
./dev-browser.sh stop
```

릴리즈 서버는 별도 DB와 3000번대 포트를 사용한다.

```sh
./release-browser.sh build
./release-browser.sh start
./release-browser.sh status
./release-browser.sh logs
./release-browser.sh stop
```

제품 하나만 제어할 수도 있다.

```sh
./dev-browser.sh liquid2 status
./dev-browser.sh plasma status
./release-browser.sh liquid2 status
./release-browser.sh plasma status
```

`install`, `start`, `restart`는 현재 스크립트에서 같은 준비 흐름을 탄다. 일반적으로
처음 띄울 때는 `start`, 이미 떠 있는 서버를 최신 코드로 다시 띄울 때는 `restart`를
쓴다.

```sh
./dev-browser.sh restart
./release-browser.sh restart
```

## 정상 상태 기준

`./dev-browser.sh status`가 다음 의미를 만족하면 개발 서버 준비가 끝난 것이다.

- Liquid2 API가 `HTTP ok`, service `loaded`
- Liquid2 Web이 `HTTP ok`, service `loaded`
- Plasma가 `HTTP ok`, service `loaded`
- Plasma가 development Liquid2 API를 바라봄
- DB 경로가 `~/research-artifacts/liquid2/...` 아래임

`./release-browser.sh status`가 다음 의미를 만족하면 로컬 릴리즈 서버 준비가 끝난
것이다.

- Liquid2 API가 `HTTP ok`, service `loaded`
- Liquid2 Web이 `HTTP ok`, service `loaded`
- Plasma가 `HTTP ok`, service `loaded`
- Plasma가 release Liquid2 API를 바라봄
- DB 경로가 `~/Library/Application Support/...` 아래임

HTTP가 `down`이면 먼저 로그를 본다.

```sh
./dev-browser.sh logs
./release-browser.sh logs
```

## 완료 보고에 포함할 사용법

에이전트가 이 파일을 읽고 준비를 완료했으면 사용자에게 다음 사용법을 출력한다.
상태 값은 실제 `status` 결과를 함께 요약한다.

```text
준비 완료.

개발 서버:
  ./dev-browser.sh start
  ./dev-browser.sh status
  ./dev-browser.sh restart
  ./dev-browser.sh logs
  ./dev-browser.sh stop

릴리즈 서버:
  ./release-browser.sh start
  ./release-browser.sh status
  ./release-browser.sh restart
  ./release-browser.sh logs
  ./release-browser.sh stop

제품별 제어:
  ./dev-browser.sh liquid2 status
  ./dev-browser.sh plasma status
  ./release-browser.sh liquid2 status
  ./release-browser.sh plasma status
```

준비를 완료하지 못했으면 "준비 완료"라고 쓰지 않는다. 실패한 명령, 실패 이유,
다음에 필요한 사용자 조치를 함께 보고한다.
