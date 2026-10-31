#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
target=${1:-.}

if [ ! -f "$root/PATCH-BASE" ]; then
  echo "PATCH-BASE not found in $root" >&2
  exit 1
fi

base=$(awk '{print $1}' "$root/PATCH-BASE")
if ! git -C "$target" rev-parse --verify "${base}^{commit}" >/dev/null 2>&1; then
  echo "Target repository does not contain $base; fetch the upstream tag first" >&2
  exit 1
fi

mapfile -t series < "$root/series"
patches=()
for patch in "${series[@]}"; do
  if [ ! -f "$root/patches/$patch" ]; then
    echo "Missing patch: $root/patches/$patch" >&2
    exit 1
  fi
  patches+=("$root/patches/$patch")
done

if [ ${#patches[@]} -eq 0 ]; then
  echo "No patches listed in $root/series" >&2
  exit 1
fi

git -C "$target" \
  -c user.name="${GIT_AUTHOR_NAME:-strip-sub-bots-patch}" \
  -c user.email="${GIT_AUTHOR_EMAIL:-strip-sub-bots-patch@localhost}" \
  am -3 "${patches[@]}"
