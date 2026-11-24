#!/usr/bin/env node
// Every commit tagged `[agent:<role>]` may only touch files in that role's write zone. This catches an agent doing
// (and signing) another agent's work in any tool, including Kiro, where the lead's permissions cannot block it.
// Zones come from the same rules the tools use: .kiro/agents/<role>.json (sub-agents) and
// scripts/agents/hubx-lead.kiro.json (the lead's own rules). Untagged commits (humans, tooling) are not checked.
// Known historical exceptions, with their reason: scripts/agent-zones-exceptions.txt.
// Usage: node scripts/check-agent-zones.mjs [base-ref]   (default: the only open origin/release/*; checks base..HEAD)
import { execFileSync } from "node:child_process";
import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { LEAD, LEAD_SOURCE, SUBAGENTS } from "./kiro-delegation.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
// Default base: the current release branch (exactly one origin/release/* is open at a time).
function currentRelease() {
  const branches = execFileSync("git", ["branch", "-r", "--list", "origin/release/*", "--format=%(refname:short)"], {
    cwd: root,
    encoding: "utf8",
  }).split("\n").filter(Boolean);
  if (branches.length !== 1) {
    console.error(`expected exactly one origin/release/* branch, found ${branches.length}: pass the base ref explicitly`);
    process.exit(2);
  }
  return branches[0];
}
const base = process.argv[2] || currentRelease();

// Same glob semantics as scripts/agent-guard.mjs for paths: `**` crosses directories, `*` stays in one segment.
function globToRegExp(glob) {
  let re = "";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (glob[i + 1] === "*") {
        i++;
        if (glob[i + 1] === "/") { i++; re += "(?:.*/)?"; } else re += ".*";
      } else re += "[^/]*";
    } else re += c.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
  }
  return new RegExp(`^${re}$`);
}
const matches = (file, globs = []) => globs.some((g) => globToRegExp(g).test(file));

function zone(role) {
  const file = role === LEAD ? LEAD_SOURCE : `.kiro/agents/${role}.json`;
  const rules = JSON.parse(readFileSync(path.join(root, file), "utf8")).permissions.rules.filter(
    (r) => r.capability === "fs_write",
  );
  const allow = rules.filter((r) => r.effect === "allow").flatMap((r) => r.match);
  // Specific denies (not the catch-all "**", which is the complement of the allow list).
  const deny = rules.filter((r) => r.effect === "deny" && !r.match.includes("**")).flatMap((r) => r.match);
  return { allow, deny };
}
const zones = Object.fromEntries([LEAD, ...SUBAGENTS].map((r) => [r, zone(r)]));
const inZone = (role, file) => matches(file, zones[role].allow) && !matches(file, zones[role].deny);

const exceptionsFile = path.join(root, "scripts/agent-zones-exceptions.txt");
const exceptions = existsSync(exceptionsFile)
  ? readFileSync(exceptionsFile, "utf8").split("\n").map((l) => l.trim()).filter((l) => l && !l.startsWith("#"))
      .map((l) => l.split(/\s+/)[0])
  : [];

const log = execFileSync("git", ["log", "--no-merges", "--format=%x1e%H%x1f%s", "--name-only", `${base}..HEAD`], {
  cwd: root,
  encoding: "utf8",
});
let violations = 0;
for (const entry of log.split("\x1e").slice(1)) {
  const [head, ...files] = entry.split("\n").filter(Boolean);
  const [sha, subject] = head.split("\x1f");
  const role = (subject.match(/\[agent:([a-z-]+)\]/) || [])[1];
  if (!role) continue;
  if (exceptions.some((e) => sha.startsWith(e))) continue;
  if (!zones[role]) {
    console.log(`${sha.slice(0, 7)} unknown agent tag [agent:${role}]: ${subject}`);
    violations++;
    continue;
  }
  const outside = files.filter((f) => !inZone(role, f));
  if (outside.length) {
    console.log(`${sha.slice(0, 7)} [agent:${role}] touches files outside its zone: ${outside.join(", ")}`);
    console.log(`        ${subject}`);
    violations++;
  }
}
if (violations) {
  console.log(`\n${violations} commit(s) break the agent zones. Redo the work with the owning agent, or (only for`);
  console.log("history that cannot change) list the commit in scripts/agent-zones-exceptions.txt with its reason.");
  process.exit(1);
}
console.log(`agent zones ok (${base}..HEAD)`);
