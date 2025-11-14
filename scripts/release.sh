#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 vX.Y.Z" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  usage
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: version must look like v1.2.3" >&2
  exit 1
fi

if ! git diff --quiet --stat HEAD; then
  echo "error: working tree has uncommitted changes" >&2
  exit 1
fi

if [[ -n "$(git ls-files --others --exclude-standard)" ]]; then
  echo "error: working tree has untracked files" >&2
  exit 1
fi

PREVIOUS=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
echo "Previous release tag: $PREVIOUS"

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "error: tag $VERSION already exists" >&2
  exit 1
fi

echo "Tagging release $VERSION..."
git tag -a "$VERSION" -m "Conflata $VERSION"
echo "Created tag $VERSION. Push with:"
echo "  git push origin $VERSION"
