#!/usr/bin/env bash
# Resets one Neon branch to the latest state of its parent ("Reset from parent"), so every staging deployment starts
# from a known database: the parent's schema, users and seed data, without what earlier test runs left behind.
# The connection string does not change. A branch without a parent (the root/production branch) is refused.
# Usage: neon-reset.sh <project-id> <branch-name> [timeout-seconds]. Needs NEON_API_KEY.
# API: GET /projects/{p}/branches; POST /projects/{p}/branches/{id}/restore {"source_branch_id": <parent id>};
#      GET /projects/{p}/branches/{id} until branch.current_state is "ready".
set -euo pipefail
project="${1:?project id}" name="${2:?branch name}" timeout="${3:-300}"
api="https://console.neon.tech/api/v2/projects/$project"
auth=(-H "Authorization: Bearer ${NEON_API_KEY:?NEON_API_KEY is required}" -H "Accept: application/json")

branch=$(curl -fsS "${auth[@]}" "$api/branches" | jq -c --arg n "$name" '.branches[] | select(.name == $n)')
[ -n "$branch" ] || { echo "::error::Neon branch '$name' not found in project $project"; exit 1; }
id=$(jq -r .id <<<"$branch"); parent=$(jq -r '.parent_id // empty' <<<"$branch")
if [ -z "$parent" ]; then
  echo "::error::Neon branch '$name' has no parent (root/production branch): refusing to reset it"; exit 1
fi
echo "resetting Neon branch $name ($id) from its parent $parent"
curl -fsS -X POST "${auth[@]}" -H "Content-Type: application/json" \
  -d "{\"source_branch_id\":\"$parent\"}" "$api/branches/$id/restore" >/dev/null

for ((t = 0; t < timeout; t += 5)); do
  state=$(curl -fsS "${auth[@]}" "$api/branches/$id" | jq -r '.branch.current_state')
  if [ "$state" = "ready" ] && [ "$t" -gt 0 ]; then echo "Neon branch $name reset and ready (${t}s)"; exit 0; fi
  sleep 5
done
echo "::error::Neon branch $name not ready after ${timeout}s (last state: $state)"
exit 1
