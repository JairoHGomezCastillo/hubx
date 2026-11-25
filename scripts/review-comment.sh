#!/usr/bin/env bash
# Publishes a story review on its PR: one summary comment, updated in place on every re-review (hidden marker), plus
# optional inline comments on lines of the diff.
# Usage: review-comment.sh <pr> <summary.md> [inline.tsv]
#   inline.tsv: one finding per line, TAB-separated: <path>\t<line>\t<comment>. Lines outside the PR diff cannot carry
#   an inline comment (GitHub rejects them); they are appended to the summary instead.
set -euo pipefail
pr="${1:?pr}" summary="${2:?summary file}" inline="${3:-}"
repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
marker="<!-- review-story -->"
body=$(mktemp); trap 'rm -f "$body"' EXIT
{ echo "$marker"; cat "$summary"; } > "$body"

# Inline comments, as one review (event COMMENT: an author cannot approve or request changes on their own PR).
if [ -n "$inline" ] && [ -s "$inline" ]; then
  sha=$(gh pr view "$pr" --json headRefOid --jq .headRefOid)
  outside=()
  while IFS=$'\t' read -r path line text; do
    [ -z "$path" ] && continue
    if gh api -X POST "repos/$repo/pulls/$pr/comments" -f body="$text" -f commit_id="$sha" -f path="$path" \
         -F line="$line" -f side=RIGHT >/dev/null 2>&1; then
      echo "inline: $path:$line"
    else
      outside+=("- \`$path:$line\`: $text")
    fi
  done < "$inline"
  if [ "${#outside[@]}" -gt 0 ]; then
    { echo; echo "### Hallazgos fuera del diff"; printf '%s\n' "${outside[@]}"; } >> "$body"
  fi
fi

# Summary: edit the previous review-story comment of this user if there is one, else create it.
me=$(gh api user --jq .login)
existing=$(gh api "repos/$repo/issues/$pr/comments?per_page=100" \
  --jq "[.[] | select(.user.login==\"$me\" and (.body | startswith(\"$marker\")))] | last | .id // empty")
if [ -n "$existing" ]; then
  gh api -X PATCH "repos/$repo/issues/comments/$existing" -F body=@"$body" --jq .html_url
else
  gh api -X POST "repos/$repo/issues/$pr/comments" -F body=@"$body" --jq .html_url
fi
