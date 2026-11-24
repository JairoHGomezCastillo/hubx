#!/usr/bin/env bash
# Independent review of a story PR: the mechanical checks a human reviewer runs before approving.
# Runs in a throwaway worktree and an isolated Docker project, so it never touches your working copy or local stack.
#
# Usage: scripts/review-story.sh <pr-number> [--fast]
#   --fast  skip the slow checks (integration suite, repeat run, MFE build, gosec/govulncheck)
# Needs: gh (logged in), git, go, node/npm, docker, gosec, govulncheck. Exit 1 if any check FAILs.
# Judgment calls (invented business rules, qa-report claims, real authorship) are left to the reviewer: see
# .claude/skills/review-story/SKILL.md.
set -uo pipefail

pr="${1:?usage: review-story.sh <pr-number> [--fast]}"
fast=false; [ "${2:-}" = "--fast" ] && fast=true
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

results=()   # "STATUS|check|detail"
add() { results+=("$1|$2|$3"); }
fail_count() { printf '%s\n' "${results[@]}" | grep -c '^FAIL|' || true; }

# ---------- PR facts ----------
info=$(gh pr view "$pr" --json state,baseRefName,headRefName,headRefOid,mergeCommit,title) || { echo "PR #$pr not found"; exit 2; }
state=$(jq -r .state <<<"$info"); base=$(jq -r .baseRefName <<<"$info"); head=$(jq -r .headRefName <<<"$info")
head_sha=$(jq -r .headRefOid <<<"$info"); title=$(jq -r .title <<<"$info")
git fetch -q origin "$base" 2>/dev/null; git fetch -q origin "$head_sha" 2>/dev/null || git fetch -q origin
if [ "$state" = "MERGED" ]; then
  base_sha=$(git rev-parse "$(jq -r .mergeCommit.oid <<<"$info")^1")
else
  base_sha=$(git merge-base "origin/$base" "$head_sha")
fi
nnn=$(grep -oE '^[0-9]{3}' <<<"$head" || true); hu=""
[ -n "$nnn" ] && hu="HU$(printf '%02d' "$((10#$nnn))")"

wt=$(mktemp -d)/review
git worktree add -q --detach "$wt" "$head_sha"
project="hubx-review-$pr"
cleanup() {
  (cd "$wt" 2>/dev/null && docker compose -p "$project" down -v --remove-orphans >/dev/null 2>&1)
  git -C "$repo_root" worktree remove --force "$wt" >/dev/null 2>&1
}
trap cleanup EXIT
cd "$wt"
# Local secrets and Node dependencies come from the main working tree (the first worktree), or are created here.
main_tree=$(git -C "$repo_root" worktree list --porcelain | awk '/^worktree /{print $2; exit}')
if [ -f "$main_tree/.env" ]; then cp "$main_tree/.env" .env; else bash scripts/env.sh >/dev/null; fi
for d in hubx-bender hubx-mfe; do
  if [ -d "$main_tree/$d/node_modules" ]; then cp -R "$main_tree/$d/node_modules" "$d/"
  else npm --prefix "$d" ci --no-audit --no-fund >/dev/null 2>&1; fi
done

echo "Reviewing PR #$pr \"$title\" ($head -> $base, ${head_sha:0:7}, story ${hu:-none})" >&2

# ---------- 1. Target and CI ----------
if [ "$base" = "main" ]; then
  bash scripts/release-version.sh "$head" >/dev/null 2>&1 \
    && add PASS "Target branch" "$head -> main (release/hotfix)" \
    || add FAIL "Target branch" "$head -> main: only release/* or hotfix/* may enter main"
else
  add PASS "Target branch" "$head -> $base (integration branch)"
fi
ci=$(gh pr checks "$pr" 2>/dev/null | awk -F'\t' '{print $1": "$2}' | sort -u)
if grep -qE ': (fail|cancel)' <<<"$ci"; then add FAIL "CI checks" "$(grep -E ': (fail|cancel)' <<<"$ci" | tr '\n' ' ')"
elif grep -qE ': pending' <<<"$ci"; then add WARN "CI checks" "still running: $(grep ': pending' <<<"$ci" | tr '\n' ' ')"
else add PASS "CI checks" "$(wc -l <<<"$ci" | tr -d ' ') checks green"; fi

# ---------- 2. Agent zones and commits ----------
zones=$(node scripts/check-agent-zones.mjs "$base_sha" 2>&1) && add PASS "Agent zones" "every [agent:<role>] commit inside its zone" \
  || add FAIL "Agent zones" "$(grep -v '^$' <<<"$zones" | head -3 | tr '\n' ' ')"
untagged=$(git log --no-merges --format='%h %s' "$base_sha..$head_sha" | grep -v '\[agent:' || true)
[ -z "$untagged" ] && add PASS "Commit tags" "all commits tagged [agent:<role>]" \
  || add WARN "Commit tags" "untagged (not by an agent?): $(head -3 <<<"$untagged" | tr '\n' ';')"

# ---------- 3. Specification ----------
if [ -n "$nnn" ]; then
  spec=$(ls specs/${nnn}-*/spec.md 2>/dev/null | head -1)
  if [ -z "$spec" ]; then add FAIL "Spec" "no specs/${nnn}-*/spec.md"
  else
    tech=$(grep -n -i -E '\b(jwt|bcrypt|sha-?256|postgres|sql|json|http|[45][0-9]{2} (ok|unauthorized|forbidden|not found))\b' "$spec" | grep -v -i 'plan.md' | head -3)
    [ -z "$tech" ] && add PASS "Spec without technology" "$spec" || add WARN "Spec without technology" "$(tr '\n' ' ' <<<"$tech")"
    grep -q -E '^\*?\*?Source\*?\*?: (backlog HU-|user description)' "$spec" && add PASS "Spec source line" "present" \
      || add WARN "Spec source line" "missing 'Source: backlog HU-xx'"
    dir=$(dirname "$spec")
    for f in plan.md tasks.md qa-report.md pr.md; do [ -f "$dir/$f" ] || add FAIL "SDD artifacts" "missing $dir/$f"; done
    grep -q -i 'verdict' "$dir/qa-report.md" 2>/dev/null && add PASS "QA report" "$dir/qa-report.md has a verdict (claims: reviewer)"
    if [ -f "$dir/deploy.md" ]; then
      grep -qE '[0-9]+ scenarios \([0-9]+ passed\)' "$dir/deploy.md" && add PASS "Deploy validation" "$dir/deploy.md has a green staging smoke" \
        || add WARN "Deploy validation" "$dir/deploy.md has no green smoke result"
    else
      add WARN "Deploy validation" "no $dir/deploy.md (devops validates the story deployed to staging before the PR)"
    fi
  fi
fi

# ---------- 4. Acceptance tests (Gherkin) ----------
if [ -n "$hu" ]; then
  feature=$(ls hubx-bender/features/${hu}_*.feature 2>/dev/null | head -1)
  if [ -z "$feature" ]; then add FAIL "Feature file" "no hubx-bender/features/${hu}_*.feature"
  else
    norm() { sed 's/[[:space:]]*$//; s/  */ /g' | grep -v '^$'; }
    awk -v h="## HU-${hu#HU}" 'index($0,h)==1{f=1;next} /^## HU-/{f=0} f' backlog/backlog.md \
      | awk '/```gherkin/{g=1;next} /```/{g=0} g' | norm > /tmp/rs-bk.$$
    grep -v -E '^\s*(@|#)' "$feature" | norm > /tmp/rs-ft.$$
    missing=$(diff <(sort -u /tmp/rs-bk.$$) <(sort -u /tmp/rs-ft.$$) | grep '^<' | head -3)
    # Titles only, "Scenario Outline:" treated as "Scenario:" (an @e2e copy may instantiate an outline's example).
    titles() { grep -E '^ ?Scenario' "$1" | sed -E 's/^ ?Scenario( Outline)?: */Scenario: /' | sort -u; }
    invented=$(comm -13 <(titles /tmp/rs-bk.$$) <(titles /tmp/rs-ft.$$))
    [ -z "$missing" ] && add PASS "Backlog Gherkin verbatim" "every backlog line is in $feature" \
      || add FAIL "Backlog Gherkin verbatim" "missing/changed: $(tr '\n' ' ' <<<"$missing")"
    [ -z "$invented" ] && add PASS "No invented scenarios" "every scenario title exists in the backlog" \
      || add FAIL "No invented scenarios" "$(tr '\n' ';' <<<"$invented")"
    grep -q '@e2e' "$feature" && add PASS "@e2e scenario" "present" || add FAIL "@e2e scenario" "none (constitution V)"
    seeds=$(grep -n -E 'HBX-SEED-[0-9]+' "$feature" hubx-bender/features/steps/${hu}*.ts 2>/dev/null | head -3)
    [ -z "$seeds" ] && add PASS "Independent of seed data" "no HBX-SEED-* in the story" \
      || add FAIL "Independent of seed data" "$(tr '\n' ' ' <<<"$seeds") (breaks re-runs on staging)"
    rm -f /tmp/rs-bk.$$ /tmp/rs-ft.$$
  fi
fi

# ---------- 5. Go structure (constitution IV) ----------
loose=$(find hubx-api hubx-gw -name '*_test.go' -not -path '*/test/*' 2>/dev/null | head -3)
[ -z "$loose" ] && add PASS "Tests in test/" "no *_test.go next to code" || add FAIL "Tests in test/" "$(tr '\n' ' ' <<<"$loose")"
http_in_domain=$(grep -l '"net/http"' $(find hubx-api/internal -maxdepth 2 -name '*.go' -not -path '*/handlers/*' -not -path '*/test/*' -not -path '*/mocks/*' -not -path '*/health/*') 2>/dev/null | head -3)
[ -z "$http_in_domain" ] && add PASS "Domain without net/http" "handlers stay in internal/handlers" \
  || add WARN "Domain without net/http" "$(tr '\n' ' ' <<<"$http_in_domain")"
hand_mocks=$(for f in $(find hubx-api hubx-gw -path '*/mocks/*.go' 2>/dev/null); do grep -q 'Code generated by MockGen' "$f" || echo "$f"; done | head -3)
[ -z "$hand_mocks" ] && add PASS "Generated mocks" "every mocks/*.go comes from mockgen" || add FAIL "Generated mocks" "hand-written: $hand_mocks"
new_migr=$(git diff --name-only --diff-filter=A "$base_sha" "$head_sha" -- hubx-api/migrations | tr '\n' ' ')
[ -z "$new_migr" ] && add PASS "Migrations" "none added" || add WARN "Migrations" "added: $new_migr (check the number is reserved in memory.md)"
for s in hubx-api hubx-gw; do
  go -C "$s" mod tidy -diff >/dev/null 2>&1 && add PASS "$s go.mod tidy" "clean" || add FAIL "$s go.mod tidy" "go mod tidy changes it"
done
go -C hubx-api generate ./... >/dev/null 2>&1 && go -C hubx-gw generate ./... >/dev/null 2>&1
[ -z "$(git status --porcelain -- hubx-api hubx-gw)" ] && add PASS "Mocks up to date" "go generate changes nothing" \
  || add FAIL "Mocks up to date" "go generate changes: $(git status --porcelain -- hubx-api hubx-gw | head -3 | tr '\n' ' ')"

# ---------- 6. Tests and security ----------
for s in hubx-api hubx-gw; do
  go -C "$s" test ./... >/dev/null 2>&1 && add PASS "$s unit tests" "go test ./..." || add FAIL "$s unit tests" "go test ./... fails"
done
if ! $fast; then
  for t in typecheck lint build test; do
    npm --prefix hubx-mfe run "$t" >/dev/null 2>&1 && add PASS "MFE $t" "ok" || add FAIL "MFE $t" "npm run $t fails"
  done
  for s in hubx-api hubx-gw; do
    (cd "$s" && gosec -quiet ./... >/dev/null 2>&1) && add PASS "$s gosec" "no findings" || add FAIL "$s gosec" "findings"
    out=$(govulncheck -C "$s" ./... 2>&1)
    grep -qE 'No vulnerabilities|affected by 0 vulnerabilities' <<<"$out" && add PASS "$s govulncheck" "no reachable vulnerabilities" \
      || add FAIL "$s govulncheck" "$(grep -E 'Vulnerability #' <<<"$out" | head -2 | tr '\n' ' ')"
  done
  # Integration on an isolated stack, twice on the same database: staging keeps its data between deployments.
  busy=$(for p in 8080 3000 5432; do lsof -nP -iTCP:$p -sTCP:LISTEN >/dev/null 2>&1 && echo $p; done | tr '\n' ' ')
  if [ -n "$busy" ]; then
    add WARN "Acceptance suite" "not run: ports busy ($busy), another stack is up (make up or a lead's test-integration); re-run when free"
  else
    docker compose -p "$project" up -d --build --quiet-pull >/dev/null 2>&1 \
      && bash scripts/wait-health.sh http://localhost:8080 120 >/dev/null 2>&1 && bash scripts/wait-ui.sh http://localhost:3000 120 >/dev/null 2>&1
    if [ $? -ne 0 ]; then add FAIL "Acceptance suite" "stack did not start"
    else
      npm --prefix hubx-bender run browsers >/dev/null 2>&1
      set -a; [ -f .env ] && . ./.env; set +a
      for run in 1 2; do
        res=$(cd hubx-bender && npx cucumber-js 2>&1 | grep -E 'scenarios \(' | tail -1)
        if [ -z "$res" ]; then
          add FAIL "Acceptance run $run" "no result (cucumber did not report scenarios)"
        elif grep -qE 'failed|undefined|pending|ambiguous' <<<"$res"; then
          add FAIL "Acceptance run $run" "$res$([ $run = 2 ] && echo ' (re-run on the same data: seed or state dependency?)')"
        else
          add PASS "Acceptance run $run" "$res"
        fi
      done
    fi
  fi
fi

# ---------- Report ----------
echo
echo "## Review of PR #$pr: $title"
echo
echo "| Result | Check | Detail |"
echo "|---|---|---|"
printf '%s\n' "${results[@]}" | while IFS='|' read -r st ck dt; do
  icon=$([ "$st" = PASS ] && echo "✅" || { [ "$st" = WARN ] && echo "⚠️" || echo "❌"; })
  echo "| $icon $st | $ck | ${dt//|//} |"
done
n=$(fail_count)
echo
echo "**Mechanical verdict:** $([ "$n" = 0 ] && echo "no failing checks" || echo "$n failing check(s)")$($fast && echo ' (fast mode: integration, MFE build and security scans skipped)')."
echo "Judgment checks (business rules decided by the human, qa-report claims, real authorship) are the reviewer's."
[ "$n" = 0 ]
