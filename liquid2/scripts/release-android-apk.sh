#!/bin/sh
set -eu

usage() {
  cat <<'EOF'
usage: release-android-apk.sh --version X.Y.Z --build-number N [--upload] [--repo OWNER/REPO]

Builds and verifies the production-flavor Android APK locally, then optionally
uploads the APK and its SHA-256 file to an existing GitHub Release vX.Y.Z.
The current Android release build uses the debug signing key and is intended
for tester sideloading, not Play Store distribution.
EOF
}

version=""
build_number=""
upload=0
repo="c86j224s/liquid2"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) version="${2:-}"; shift 2 ;;
    --build-number) build_number="${2:-}"; shift 2 ;;
    --repo) repo="${2:-}"; shift 2 ;;
    --upload) upload=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

case "$version" in
  ''|*[!0-9.]*) printf 'invalid version: %s\n' "$version" >&2; exit 2 ;;
esac
case "$version" in
  *.*.*) ;;
  *) printf 'version must use X.Y.Z: %s\n' "$version" >&2; exit 2 ;;
esac
case "$build_number" in
  ''|*[!0-9]*) printf 'build number must be a positive integer\n' >&2; exit 2 ;;
esac
[ "$build_number" -gt 0 ] || { printf 'build number must be positive\n' >&2; exit 2; }

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
liquid_dir="$(CDPATH= cd -- "${script_dir}/.." && pwd)"
client_dir="${liquid_dir}/client"
sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/homebrew/share/android-commandlinetools}}"
build_tools="$(find "${sdk_root}/build-tools" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort -V | tail -1)"
aapt="${build_tools}/aapt"
apksigner="${build_tools}/apksigner"

[ -x "$aapt" ] || { printf 'missing aapt under Android SDK: %s\n' "$sdk_root" >&2; exit 1; }
[ -x "$apksigner" ] || { printf 'missing apksigner under Android SDK: %s\n' "$sdk_root" >&2; exit 1; }
command -v flutter >/dev/null 2>&1 || { printf 'flutter is not on PATH\n' >&2; exit 1; }

java_home="${JAVA_HOME:-}"
if [ -z "$java_home" ]; then
  for candidate in /opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home /opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home; do
    if [ -x "${candidate}/bin/java" ]; then
      java_home="$candidate"
      break
    fi
  done
fi
[ -n "$java_home" ] && [ -x "${java_home}/bin/java" ] || {
  printf 'missing Java runtime; set JAVA_HOME to JDK 17 or newer\n' >&2
  exit 1
}

export ANDROID_HOME="$sdk_root"
export ANDROID_SDK_ROOT="$sdk_root"
export JAVA_HOME="$java_home"
export PATH="${JAVA_HOME}/bin:${PATH}"
(cd "$client_dir" && flutter clean && flutter pub get && flutter build apk --release --flavor prod --build-name "$version" --build-number "$build_number")

built_apk="${client_dir}/build/app/outputs/flutter-apk/app-prod-release.apk"
[ -f "$built_apk" ] || { printf 'missing built APK: %s\n' "$built_apk" >&2; exit 1; }

badging="$($aapt dump badging "$built_apk")"
printf '%s\n' "$badging" | grep -F "package: name='kr.smpl.liquid2' versionCode='${build_number}' versionName='${version}'" >/dev/null || {
  printf 'APK package/version verification failed\n%s\n' "$badging" >&2
  exit 1
}
$apksigner verify --verbose --print-certs "$built_apk" >"${built_apk}.signing.txt"
grep -F 'CN=Android Debug' "${built_apk}.signing.txt" >/dev/null || {
  printf 'APK is not signed with the expected Android debug key\n' >&2
  exit 1
}

artifact_dir="${liquid_dir}/build/releases/v${version}"
mkdir -p "$artifact_dir"
asset="${artifact_dir}/liquid2-android-v${version}.apk"
cp "$built_apk" "$asset"
(cd "$artifact_dir" && shasum -a 256 "$(basename "$asset")" >"$(basename "$asset").sha256")
(cd "$artifact_dir" && shasum -a 256 -c "$(basename "$asset").sha256")

if [ "$upload" -eq 1 ]; then
  command -v gh >/dev/null 2>&1 || { printf 'gh is not on PATH\n' >&2; exit 1; }
  gh release view "v${version}" --repo "$repo" >/dev/null
  gh release upload "v${version}" "$asset" "${asset}.sha256" --repo "$repo" --clobber
  gh release view "v${version}" --repo "$repo" --json assets --jq '.assets[] | [.name, .size, .state] | @tsv'
fi

printf 'APK: %s\nSHA-256: %s\nSigning: Android debug key (tester sideload only)\n' "$asset" "${asset}.sha256"
