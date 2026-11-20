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
unformatted=$("$(go env GOROOT)/bin/gofmt" -l generator cmd go)
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

echo "== go"
(cd go && go vet ./... && go test -count=1 -race ./...)

echo "== typescript"
if need npm typescript; then
	(cd typescript && npm ci --ignore-scripts --no-audit --no-fund --silent && npx tsc -p tsconfig.json --noEmit && npm run typecheck:test --silent && node --test 'test/*.test.ts')
fi

echo "== python"
if need python3 python; then
	(cd python && PYTHONPATH=src python3 -W error -m unittest discover -s tests)
	if command -v "${MYPY:-mypy}" >/dev/null 2>&1; then
		(cd python && "${MYPY:-mypy}" --strict --python-version 3.11 src/meeng)
	else
		echo "-- python: mypy is not installed, so the types were not checked (MYPY names it)"
		[ "${SKIP_MISSING:-}" = 1 ] || exit 1
	fi
fi

echo "== php"
if need php php; then
	(cd php && php tests/run.php)
	phpstan="${PHPSTAN:-phpstan}"
	case "$phpstan" in
	*.phar) (cd php && php "$phpstan" analyse --no-progress --level 8 src) ;;
	*)
		if command -v "$phpstan" >/dev/null 2>&1; then
			(cd php && "$phpstan" analyse --no-progress --level 8 src)
		else
			echo "-- php: phpstan is not installed, so the types were not checked (PHPSTAN names it)"
			[ "${SKIP_MISSING:-}" = 1 ] || exit 1
		fi
		;;
	esac
fi

echo "== ruby"
if need ruby ruby; then
	(cd ruby && ruby -w -Ilib -Itest test/meeng_test.rb)
fi

echo "== java"
if need mvn java; then
	(cd java && mvn -q -B verify)
fi

echo "== csharp"
if need dotnet csharp; then
	(cd csharp && dotnet test --nologo)
fi

echo "== elixir"
if need mix elixir; then
	(cd elixir && mix compile --warnings-as-errors && mix test --warnings-as-errors)
fi

echo "== dart"
if need dart dart; then
	(cd dart && dart pub get && dart format --language-version=3.8 --output=none --set-exit-if-changed . && dart analyze --fatal-infos && dart test)
fi

echo "== rust"
if need cargo rust; then
	(cd rust && cargo fmt --check && cargo clippy --all-targets -- -D warnings && cargo test)
fi

if [ "${SKIP_SWIFT:-}" != 1 ]; then
	echo "== swift"
	if need swift swift; then
		swift=swift
		if command -v xcrun >/dev/null; then
			swift="xcrun swift"
		fi
		(cd swift && $swift test) && $swift build
	fi
fi
