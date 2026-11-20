#!/bin/sh
# Writes every SDK again from the API's contract.
#
#   ./regen.sh              the contract committed at the HEAD of meeng.dev's develop
#   ./regen.sh <ref>        the contract committed at another ref (a branch, a tag, a hash)
#   ./regen.sh --worktree   the contract file as it stands in meeng.dev's checkout,
#                           uncommitted changes included
#   ./regen.sh --reference  the contract the SDKs started from: the useSend v1
#                           contract in the read-only clone, with the brand swapped
#
# The app's contract is docs/contracts/api.openapi.json in the meeng.dev
# repository, which `aru openapi:check` keeps equal to the app's routes. The app
# does not export it yet, so until it does only --reference works; the committed
# openapi.json is that same contract. MEENG_DEV names the meeng.dev checkout
# (../meeng.dev by default) and USESEND the clone (../reference_usesend).
#
# The contract is copied into openapi.json, then `go run ./cmd/sdkgen` writes the
# eleven SDKs. Run ./check.sh afterwards: nothing is finished until it exits zero.
set -eu
cd "$(dirname "$0")"
export GOWORK=off

meeng_dev="${MEENG_DEV:-../meeng.dev}"
usesend="${USESEND:-../reference_usesend}"
contract=docs/contracts/api.openapi.json
ref="${1:-develop}"

if [ "$ref" = "--reference" ]; then
	source="$usesend/apps/docs/api-reference/openapi.json"
	[ -f "$source" ] || { echo "regen: $source does not exist" >&2; exit 1; }
	tmp=$(mktemp "${TMPDIR:-/tmp}/meeng-contract.XXXXXX")
	trap 'rm -f "$tmp"' EXIT
	# Only the brand changes: the title and the server. Paths, schemas and names stay.
	sed -e 's/"title": "useSend API"/"title": "meeng API"/' -e 's#https://app.usesend.com/api#https://app.meeng.dev/api#' "$source" >"$tmp"
	if grep -qi 'usesend\|unsend' "$tmp"; then
		echo "regen: the useSend contract names its brand somewhere this script does not swap" >&2
		exit 1
	fi
	echo "regen: contract from $source, brand swapped"
	source="$tmp"
elif [ "$ref" = "--worktree" ]; then
	source="$meeng_dev/$contract"
	[ -f "$source" ] || { echo "regen: $source does not exist" >&2; exit 1; }
	echo "regen: contract from $source (working tree)"
else
	tmp=$(mktemp "${TMPDIR:-/tmp}/meeng-contract.XXXXXX")
	trap 'rm -f "$tmp"' EXIT
	git -C "$meeng_dev" show "$ref:$contract" >"$tmp" || { echo "regen: $ref has no $contract in $meeng_dev" >&2; exit 1; }
	source="$tmp"
	echo "regen: contract from $meeng_dev at $ref ($(git -C "$meeng_dev" rev-parse --short "$ref"))"
fi

go run ./cmd/sdkgen -sync "$source"
