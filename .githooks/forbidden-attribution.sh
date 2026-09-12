#!/bin/sh
# Shared matcher for AI-assistant attribution that must never reach the
# public history or a pull request.
#
# Usage: forbidden-attribution.sh <file>
# Exits 0 when the file is clean, 1 when it matches, printing the offenders.

set -eu

target="${1:?usage: forbidden-attribution.sh <file>}"

# -i so casing does not matter, -E for alternation. Each pattern is anchored
# to how the attribution actually appears rather than to the vendor name
# alone, so prose that legitimately discusses these tools still passes.
pattern='claude-session:|claude-code-session:|co-authored-by:[[:space:]]*claude|generated with[[:space:]]*\[?claude|https://claude\.ai/code/session|noreply@anthropic\.com|🤖[[:space:]]*generated with'

if matches=$(grep -inE "$pattern" "$target"); then
	echo "Refusing: assistant attribution found." >&2
	echo >&2
	echo "$matches" >&2
	echo >&2
	echo "This repository is public. Remove these lines and try again." >&2
	exit 1
fi

exit 0
