#!/bin/sh
# The gates every change passes: the generator and its tests, the SDKs being
# exactly what the generator writes, and each language's own tests and type
# checks. CHECK_DRIFT=1 (CI) also fails when a regenerated SDK differs from
# what is committed. MYPY and PHPSTAN name those tools when they are not on
# the PATH (PHPSTAN may be a phpstan.phar: it is run with php when it ends in
# .phar). Swift needs Apple's toolchain (CryptoKit, Network): it runs only on
# macOS, and SKIP_SWIFT=1 leaves it to the macOS CI job.
#
# Every step prints "== <language>" first, and a language whose toolchain is not
# installed is named and skipped only when SKIP_MISSING=1; without it the gate
# stops there, because a language nobody checked is not checked.
#
# The last step, "== skills", checks skills/: each SKILL.md has valid frontmatter
# and links that resolve, and every Go, TypeScript, Python and PHP example in them
# compiles against the SDKs of this repository (so it runs after the steps above
# installed typescript/node_modules). Examples are whole programs; a fragment is
# fenced as "<lang> nocheck" and counted in the report.
set -eu
cd "$(dirname "$0")"
export GOWORK=off

need() { # need <tool> <language>: stops, or skips with SKIP_MISSING=1
	if command -v "$1" >/dev/null 2>&1; then
		return 0
	fi
	if [ "${SKIP_MISSING:-}" = 1 ]; then
		echo "-- $2 skipped: $1 is not installed"
		return 1
	fi
	echo "$2: $1 is not installed (SKIP_MISSING=1 skips what is missing)" >&2
	exit 1
}

echo "== generator"
unformatted=$("$(go env GOROOT)/bin/gofmt" -l generator fakeapi cmd go)
if [ -n "$unformatted" ]; then
	echo "not gofmt'd:"
	echo "$unformatted"
	exit 1
fi
go vet ./...
go test -count=1 -race ./...
go run ./cmd/sdkgen
if [ "${CHECK_DRIFT:-}" = 1 ] && [ -n "$(git status --porcelain -- openapi.json go typescript python php ruby java csharp elixir dart rust swift Package.swift)" ]; then
	echo "the SDKs differ from what the generator writes: run go run ./cmd/sdkgen and commit"
	git status --porcelain -- go typescript python php ruby java csharp elixir dart rust swift Package.swift
	exit 1
fi

# The fake API, built from the contract: every language's tests call it (the
# contract tests, MEENG_FAKE_URL), and after them every operation must have been
# called, so a method an SDK lacks, or a test that was skipped, fails the gate.
fake_dir=$(mktemp -d "${TMPDIR:-/tmp}/meeng-fake.XXXXXX")
go build -o "$fake_dir/fakeapi" ./cmd/fakeapi
"$fake_dir/fakeapi" -spec openapi.json -url-file "$fake_dir/url" >"$fake_dir/log" 2>&1 &
fake_pid=$!
trap 'kill "$fake_pid" 2>/dev/null; rm -rf "$fake_dir"' EXIT
tries=0
while [ ! -s "$fake_dir/url" ]; do
	tries=$((tries + 1))
	if [ "$tries" -gt 100 ]; then
		echo "the fake API did not start:" >&2
		cat "$fake_dir/log" >&2
		exit 1
	fi
	sleep 0.1
done
MEENG_FAKE_URL=$(cat "$fake_dir/url")
export MEENG_FAKE_URL

contract() { # contract <language>: every operation was called by the language's tests; then forget
	"$fake_dir/fakeapi" -coverage "$MEENG_FAKE_URL" || { echo "$1: its contract test left an operation uncalled" >&2; exit 1; }
	"$fake_dir/fakeapi" -reset "$MEENG_FAKE_URL"
}

examples() { # examples <language> <command> <name>...: runs the command once for each example, against the fake API; then forgets
	language=$1
	command=$2
	shift 2
	for name in "$@"; do
		if ! MEENG_API_KEY=mg_example MEENG_BASE_URL=$MEENG_FAKE_URL sh -c "$command" example "$name" >/dev/null; then
			echo "$language: the example $name failed" >&2
			exit 1
		fi
	done
	"$fake_dir/fakeapi" -reset "$MEENG_FAKE_URL"
}

echo "== go"
(cd go && go vet ./... && go test -count=1 -race ./...)
contract go
examples go 'cd go && go run ./examples/"$1"' send_email send_batch schedule_email add_domain verify_webhook

echo "== typescript"
if need npm typescript; then
	(cd typescript && npm ci --ignore-scripts --no-audit --no-fund --silent && npx tsc -p tsconfig.json --noEmit && npm run typecheck:test --silent && node --test 'test/*.test.ts')
	contract typescript
	examples typescript 'cd typescript && node examples/"$1".ts' send-email send-batch schedule-email add-domain verify-webhook
fi

echo "== python"
if need python3 python; then
	(cd python && PYTHONPATH=src python3 -W error -m unittest discover -s tests)
	contract python
	examples python 'cd python && PYTHONPATH=src python3 -W error examples/"$1".py' send_email send_batch schedule_email add_domain verify_webhook
	if command -v "${MYPY:-mypy}" >/dev/null 2>&1; then
		(cd python && "${MYPY:-mypy}" --strict --python-version 3.11 src/meeng examples)
	else
		echo "-- python: mypy is not installed, so the types were not checked (MYPY names it)"
		[ "${SKIP_MISSING:-}" = 1 ] || exit 1
	fi
fi

echo "== php"
if need php php; then
	(cd php && php tests/run.php)
	contract php
	examples php 'cd php && php examples/"$1".php' send_email send_batch schedule_email add_domain verify_webhook
	phpstan="${PHPSTAN:-phpstan}"
	case "$phpstan" in
	*.phar) (cd php && php "$phpstan" analyse --no-progress --memory-limit=1G --level 8 src examples) ;;
	*)
		if command -v "$phpstan" >/dev/null 2>&1; then
			(cd php && "$phpstan" analyse --no-progress --memory-limit=1G --level 8 src examples)
		else
			echo "-- php: phpstan is not installed, so the types were not checked (PHPSTAN names it)"
			[ "${SKIP_MISSING:-}" = 1 ] || exit 1
		fi
		;;
	esac
fi

echo "== ruby"
if need ruby ruby; then
	(cd ruby && ruby -w -Ilib -Itest -e 'Dir["test/*_test.rb"].sort.each { |file| require "./#{file}" }')
	contract ruby
	examples ruby 'cd ruby && ruby -w examples/"$1".rb' send_email send_batch schedule_email add_domain verify_webhook
fi

echo "== java"
if need mvn java; then
	(cd java && mvn -q -B verify)
	contract java
	examples java 'cd java && java -cp target/classes examples/"$1".java' SendEmail SendBatch ScheduleEmail AddDomain VerifyWebhook
fi

echo "== csharp"
if need dotnet csharp; then
	(cd csharp && dotnet test --nologo)
	contract csharp
	examples csharp 'cd csharp && dotnet run --project examples -- "$1"' send-email send-batch schedule-email add-domain verify-webhook
fi

echo "== elixir"
if need mix elixir; then
	(cd elixir && mix compile --warnings-as-errors && mix test --warnings-as-errors)
	contract elixir
	examples elixir 'cd elixir && mix run examples/"$1".exs' send_email send_batch schedule_email add_domain verify_webhook
fi

echo "== dart"
if need dart dart; then
	(cd dart && dart pub get && dart format --language-version=3.8 --output=none --set-exit-if-changed . && dart analyze --fatal-infos && dart test)
	contract dart
	examples dart 'cd dart && dart run example/"$1".dart' send_email send_batch schedule_email add_domain verify_webhook
fi

echo "== rust"
if need cargo rust; then
	(cd rust && cargo fmt --check && cargo clippy --all-targets -- -D warnings && cargo test)
	contract rust
	examples rust 'cd rust && cargo run -q --example "$1"' send_email send_batch schedule_email add_domain verify_webhook
fi

if [ "${SKIP_SWIFT:-}" != 1 ]; then
	echo "== swift"
	if need swift swift; then
		swift=swift
		if command -v xcrun >/dev/null; then
			swift="xcrun swift"
		fi
		(cd swift && $swift test) && $swift build
		contract swift
		(cd swift && $swift build --product MeengExamples)
		examples swift "cd swift && $swift run -q MeengExamples \"\$1\"" send-email send-batch schedule-email add-domain verify-webhook
	fi
fi

echo "== skills"
go run ./cmd/skillscheck
