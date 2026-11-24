#!/usr/bin/env bash
# Prints GitHub Actions outputs for a release or hotfix branch name:
#   release/v0.2.0-sign-in -> version=v0.2.0, title=v0.2.0 sign-in
#   hotfix/v0.1.1          -> version=v0.1.1, title=v0.1.1 hotfix
# Fails if the name has no vX.Y.Z. Usage: release-version.sh <branch>
set -euo pipefail
branch="${1:?usage: release-version.sh <branch>}"
re='^(release|hotfix)/(v[0-9]+\.[0-9]+\.[0-9]+)(-([A-Za-z0-9._-]+))?$'
if [[ ! "$branch" =~ $re ]]; then
  echo "::error::branch '$branch' must be release/vX.Y.Z[-name] or hotfix/vX.Y.Z" >&2
  exit 1
fi
kind="${BASH_REMATCH[1]}" version="${BASH_REMATCH[2]}" name="${BASH_REMATCH[4]}"
if [ -z "$name" ] && [ "$kind" = hotfix ]; then name=hotfix; fi
echo "version=$version"
echo "title=$version${name:+ ${name//-/ }}"
