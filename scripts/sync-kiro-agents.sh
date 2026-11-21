#!/usr/bin/env bash
# Copies the body (role instructions) of each opencode agent into .kiro/agent-prompts/<name>.md so Kiro agents
# (.kiro/agents/<name>.json) use the same instructions. Source of truth: .opencode/agents/<name>.md.
# Usage: scripts/sync-kiro-agents.sh          regenerate the prompts
#        scripts/sync-kiro-agents.sh --check  exit 1 if any prompt is out of date (for CI)
set -euo pipefail
cd "$(dirname "$0")/.."

out=.kiro/agent-prompts
mode=${1:-sync}
mkdir -p "$out"
stale=0

for src in .opencode/agents/*.md; do
  name=$(basename "$src" .md)
  dst="$out/$name.md"
  tmp=$(mktemp)
  {
    printf '<!-- Generated from %s by scripts/sync-kiro-agents.sh. Do not edit: change the source and run make agents-sync. -->\n\n' "$src"
    # Drop the YAML front matter (opencode config); keep the instructions.
    awk 'NR==1 && $0=="---" {fm=1; next} fm && $0=="---" {fm=0; next} !fm' "$src"
  } > "$tmp"
  if [ "$mode" = "--check" ]; then
    if ! cmp -s "$tmp" "$dst"; then echo "out of date: $dst (run make agents-sync)"; stale=1; fi
    rm -f "$tmp"
  else
    mv "$tmp" "$dst"
    echo "synced $dst"
  fi
done

exit "$stale"
