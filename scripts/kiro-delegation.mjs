// Kiro delegation envelope for hubx-lead.
// In Kiro (IDE and CLI V3) every tool call of a sub-agent is checked against the parent's policy AND the sub-agent's
// own policy; a deny in either wins, and a sub-agent's allow cannot grant what the parent does not allow
// (https://kiro.dev/docs/custom-agents/subagents/). hubx-lead must therefore allow the union of what its sub-agents
// may do, while each sub-agent stays scoped by its own deny rules.
// Source of the lead's own rules: scripts/agents/hubx-lead.kiro.json (used by Claude Code's guard for the lead's own
// calls). Generated file: .kiro/agents/hubx-lead.json = own rules + this envelope (scripts/sync-agents.mjs).
import { readFileSync } from "node:fs";
import path from "node:path";

export const LEAD = "hubx-lead";
export const SUBAGENTS = ["backend-go", "frontend-react", "qa-seguridad", "devops"];
export const LEAD_SOURCE = "scripts/agents/hubx-lead.kiro.json";

const uniq = (xs) => [...new Set(xs)];

// Union of the sub-agents' allow/ask patterns per capability.
export function envelope(root) {
  const env = { fsWriteAllow: [], shellAllow: [], shellAsk: [] };
  for (const name of SUBAGENTS) {
    const rules = JSON.parse(readFileSync(path.join(root, ".kiro/agents", `${name}.json`), "utf8")).permissions.rules;
    for (const r of rules) {
      if (r.capability === "fs_write" && r.effect === "allow") env.fsWriteAllow.push(...r.match);
      if (r.capability === "shell" && r.effect === "allow") env.shellAllow.push(...r.match);
      if (r.capability === "shell" && r.effect === "ask") env.shellAsk.push(...r.match);
    }
  }
  return { fsWriteAllow: uniq(env.fsWriteAllow), shellAllow: uniq(env.shellAllow), shellAsk: uniq(env.shellAsk) };
}

// The generated Kiro config of hubx-lead: own rules, catch-all denies widened by the envelope, envelope rules appended.
export function leadKiroConfig(root) {
  const lead = JSON.parse(readFileSync(path.join(root, LEAD_SOURCE), "utf8"));
  const env = envelope(root);
  const rules = lead.permissions.rules.map((r) => {
    if (r.effect !== "deny") return r;
    if (r.capability === "fs_write" && r.match.includes("**")) {
      return { ...r, exclude: uniq([...(r.exclude || []), ...env.fsWriteAllow]) };
    }
    if (r.capability === "shell" && r.match.includes("*")) {
      return { ...r, exclude: uniq([...(r.exclude || []), ...env.shellAllow, ...env.shellAsk]) };
    }
    return r;
  });
  rules.push({ capability: "fs_write", match: env.fsWriteAllow, effect: "allow" });
  rules.push({ capability: "shell", match: env.shellAllow, effect: "allow" });
  if (env.shellAsk.length) rules.push({ capability: "shell", match: env.shellAsk, effect: "ask" });
  return { ...lead, permissions: { ...lead.permissions, rules } };
}
