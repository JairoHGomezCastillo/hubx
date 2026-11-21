#!/usr/bin/env node
// Claude Code PreToolUse hook: enforces the role permissions of one HUBX agent in Claude Code, reading the same rules
// Kiro uses (.kiro/agents/<agent>.json, permissions.rules). Semantics follow Kiro: deny > ask > allow, a rule applies
// when a target matches `match` and no `exclude`; compound shell commands are checked one sub-command at a time.
// Usage (from .claude/agents/<agent>.md): node "$CLAUDE_PROJECT_DIR/scripts/agent-guard.mjs" <agent>
// Reads the hook JSON on stdin. Deny: exit 2 with the reason on stderr. Ask: JSON decision on stdout. Else: silent.
import { readFileSync } from "node:fs";
import path from "node:path";

const agent = process.argv[2];
const root = process.env.CLAUDE_PROJECT_DIR || process.cwd();

function block(reason) {
  process.stderr.write(`[agent-guard:${agent}] ${reason}\n`);
  process.exit(2);
}

if (!agent) block("missing agent name");
let rules;
try {
  rules = JSON.parse(readFileSync(path.join(root, ".kiro/agents", `${agent}.json`), "utf8")).permissions.rules;
} catch (e) {
  block(`cannot read .kiro/agents/${agent}.json: ${e.message}`);
}

let input;
try {
  input = JSON.parse(readFileSync(0, "utf8"));
} catch {
  block("invalid hook input");
}
const tool = input.tool_name || "";
const args = input.tool_input || {};

// Glob -> RegExp. Paths: `**` crosses directories, `*` stays inside one segment. Shell: `*` is any sequence.
function globToRegExp(glob, kind) {
  let re = "";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (kind === "path" && glob[i + 1] === "*") {
        i++;
        if (glob[i + 1] === "/") { i++; re += "(?:.*/)?"; } else re += ".*";
      } else re += kind === "path" ? "[^/]*" : ".*";
    } else re += c.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
  }
  return new RegExp(`^${re}$`, "s");
}
const matches = (target, globs = [], kind) => globs.some((g) => globToRegExp(g, kind).test(target));

// Strongest effect among the rules of `capability` that apply to `target`.
function decide(capability, target, kind) {
  let effect = null;
  const rank = { allow: 1, ask: 2, deny: 3 };
  for (const r of rules) {
    if (r.capability !== capability) continue;
    if (!matches(target, r.match, kind) || matches(target, r.exclude, kind)) continue;
    if (!effect || rank[r.effect] > rank[effect]) effect = r.effect;
  }
  return effect;
}

function relPath(p) {
  const abs = path.resolve(root, p);
  const rel = path.relative(root, abs).split(path.sep).join("/");
  return rel.startsWith("..") ? abs : rel;
}

function splitShell(cmd) {
  return cmd.split(/&&|\|\||[;|\n]/).map((s) => s.trim()).filter(Boolean);
}

const checks = [];
if (tool === "Bash") {
  for (const sub of splitShell(args.command || "")) checks.push(["shell", sub, "shell"]);
} else if (["Edit", "Write", "MultiEdit", "NotebookEdit"].includes(tool)) {
  checks.push(["fs_write", relPath(args.file_path || args.notebook_path || ""), "path"]);
} else if (tool === "Read") {
  checks.push(["fs_read", relPath(args.file_path || ""), "path"]);
}

let ask = null;
for (const [capability, target, kind] of checks) {
  const effect = decide(capability, target, kind);
  if (effect === "deny") block(`${capability} denied for '${target}' by .kiro/agents/${agent}.json`);
  if (effect === "ask") ask = `${capability} '${target}' needs confirmation (.kiro/agents/${agent}.json)`;
}
if (ask) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: { hookEventName: "PreToolUse", permissionDecision: "ask", permissionDecisionReason: ask },
  }));
}
process.exit(0);
