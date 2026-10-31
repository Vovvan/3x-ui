#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
repo=${1:?usage: export.sh <fork-repo> [base] [output-dir]}
base=${2:-v3.11.0}
out=${3:-"$root/patches"}

base_commit=$(git -C "$repo" rev-parse "${base}^{commit}")
rm -rf "$out"
mkdir -p "$out"

git -C "$repo" format-patch --no-signature -o "$out" "$base"..HEAD -- . ':(exclude).github'
(cd "$out" && ls -1 *.patch | sort) > "$root/series"
printf '%s %s\n' "$base" "$base_commit" > "$root/PATCH-BASE"
