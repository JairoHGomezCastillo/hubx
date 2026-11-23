#!/usr/bin/env node
// Generates the Kiro and Claude Code copies of the HUBX agents from the single source, .opencode/agents/<name>.md:
//   .kiro/agent-prompts/<name>.md  instructions (body) used by .kiro/agents/<name>.json
//   .claude/agents/<name>.md       Claude Code subagent: model, tools and a guard hook + the same instructions
// Permissions live in .kiro/agents/<name>.json; Claude Code enforces them through scripts/agent-guard.mjs.
//   .kiro/agents/hubx-lead.json    generated: the lead's own rules (scripts/agents/hubx-lead.kiro.json) plus the
//                                  delegation envelope its sub-agents need in Kiro (scripts/kiro-delegation.mjs).
// Usage: node scripts/sync-agents.mjs [--check]   (--check exits 1 if any generated file is out of date)
import { readFileSync, writeFileSync, readdirSync, mkdirSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { LEAD, leadKiroConfig } from "./kiro-delegation.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const check = process.argv.includes("--check");
const src = path.join(root, ".opencode/agents");

// Claude Code model and tools per role. hubx-lead runs on Opus 5.5 in every tool: here (Claude Code), in the opencode
// front matter (`model: opencode/claude-opus-5-5`) and in .kiro/agents/hubx-lead.json (`claude-opus-5.5`).
// The other roles set no model in opencode (default); Kiro sets theirs in .kiro/agents/<name>.json.
const claude = {
  "hubx-lead": { model: "claude-opus-5-5", tools: "Read, Grep, Glob, Edit, Write, Bash, Agent(backend-go, frontend-react, qa-seguridad, devops)" },
  "qa-seguridad": { model: "sonnet" },
  "backend-go": { model: "sonnet" },
  "frontend-react": { model: "sonnet" },
  devops: { model: "haiku" },
};
const subagentTools = "Read, Grep, Glob, Edit, Write, Bash";

let stale = false;
function emit(rel, content) {
  const file = path.join(root, rel);
  if (check) {
    if (!existsSync(file) || readFileSync(file, "utf8") !== content) {
      console.log(`out of date: ${rel} (run make agents-sync)`);
      stale = true;
    }
    return;
  }
  mkdirSync(path.dirname(file), { recursive: true });
  writeFileSync(file, content);
  console.log(`synced ${rel}`);
}

for (const f of readdirSync(src).filter((f) => f.endsWith(".md")).sort()) {
  const name = path.basename(f, ".md");
  const text = readFileSync(path.join(src, f), "utf8").replace(/\r\n/g, "\n");
  const m = text.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/);
  if (!m) throw new Error(`${f}: missing front matter`);
  const [, front, body] = m;
  const description = (front.match(/^description:\s*(.*)$/m) || [])[1];
  if (!description) throw new Error(`${f}: missing description`);
  const note = `<!-- Generated from .opencode/agents/${f} by scripts/sync-agents.mjs. Do not edit: change the source and run make agents-sync. -->\n\n`;

  emit(`.kiro/agent-prompts/${name}.md`, note + body);

  const c = claude[name];
  if (!c) throw new Error(`${f}: add its Claude Code model to scripts/sync-agents.mjs`);
  const header = [
    "---",
    `name: ${name}`,
    `description: ${JSON.stringify(description)}`,
    `model: ${c.model}`,
    `tools: ${c.tools || subagentTools}`,
    "hooks:",
    "  PreToolUse:",
    '    - matcher: "Read|Edit|Write|MultiEdit|NotebookEdit|Bash"',
    "      hooks:",
    "        - type: command",
    `          command: 'node "$CLAUDE_PROJECT_DIR/scripts/agent-guard.mjs" ${name}'`,
    "---",
    "",
  ].join("\n");
  emit(`.claude/agents/${name}.md`, header + note + body);
}

emit(`.kiro/agents/${LEAD}.json`, JSON.stringify(leadKiroConfig(root), null, 2) + "\n");

process.exit(stale ? 1 : 0);
