#!/bin/sh
# Writes every SDK again from the API's contract.
#
#   ./regen.sh              the contract committed at origin/develop of meeng.dev
#                           (fetched first)
#   ./regen.sh <ref>        the contract committed at another ref (a branch, a tag, a hash)
#   ./regen.sh --worktree   the contract file as it stands in meeng.dev's checkout,
#                           uncommitted changes included
#
# The app's contract is docs/contracts/api.openapi.json in the meeng.dev
# repository, which `aru openapi:check` keeps equal to the app's routes: the
# route is the source and the SDKs follow it. MEENG_DEV names the meeng.dev
# checkout (../meeng.dev by default).
#
# The contract is copied into openapi.json (generator.SanitizeContract leaves out
# the one sentence of the description that names another product, and refuses
# a contract that names one anywhere else), then `go run ./cmd/sdkgen` writes the
# eleven SDKs. Run ./check.sh afterwards: nothing is finished until it exits zero.
set -eu
cd "$(dirname "$0")"
export GOWORK=off

meeng_dev="${MEENG_DEV:-../meeng.dev}"
contract=docs/contracts/api.openapi.json
ref="${1:-origin/develop}"

if [ "$ref" = "--worktree" ]; then
	source="$meeng_dev/$contract"
	[ -f "$source" ] || { echo "regen: $source does not exist" >&2; exit 1; }
	echo "regen: contract from $source (working tree)"
else
	if [ "$ref" = "origin/develop" ]; then
		git -C "$meeng_dev" fetch --quiet origin || { echo "regen: could not fetch origin in $meeng_dev" >&2; exit 1; }
	fi
	tmp=$(mktemp "${TMPDIR:-/tmp}/meeng-contract.XXXXXX")
	trap 'rm -f "$tmp"' EXIT
	git -C "$meeng_dev" show "$ref:$contract" >"$tmp" || { echo "regen: $ref has no $contract in $meeng_dev" >&2; exit 1; }
	source="$tmp"
	echo "regen: contract from $meeng_dev at $ref ($(git -C "$meeng_dev" rev-parse --short "$ref"))"
fi

go run ./cmd/sdkgen -sync "$source"
