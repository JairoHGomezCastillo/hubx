import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import type { Download } from 'playwright';
import { authHeader, OPERATOR_NAME, prepareShipment, signInAs, type UpdateWorld } from './common.steps.ts';

// HU-10: Download the traceability evidence. All HTTP goes through the gateway
// (world.api, baseURL BENDER_BASE_URL) except the @e2e download, which is driven
// exclusively through world.ui against the MFE.

const FINGERPRINT = /^[0-9a-f]{64}$/;
const HEX64_GLOBAL = /\b[0-9a-f]{64}\b/g;
const UNKNOWN_CODE = 'HBX-99999999';
const NAMES_STATEMENT = [
  'Fingerprints cover the party identifiers, not the printed names.',
  'The covered fields are responsible_party_id and assigned_partner_id.',
  'The names cannot be verified from this document alone.',
];

// Demo data from migration 0001; fixtures use the same owner as prepareShipment.
const CUSTOMER_NAMES: Record<string, string> = {
  'andina@hubx.co': 'Distribuciones Andina',
  'ana@hubx.co': 'Ana Customer',
};

interface EvidenceWorld extends UpdateWorld {
  response?: import('playwright').APIResponse;
  download?: Download;
  evidencePath?: string;
  pdfBytes?: Buffer;
  pdfTextContent?: string;
  recomputedFingerprints?: string[];
}

// ---------------------------------------------------------------------------
// PDF text extraction. The contract fixes uncompressed text streams, so every
// (...) literal string in the content stream is readable document text. Octal
// escapes are decoded back to bytes and then interpreted as UTF-8 (falling back
// to Latin-1), which keeps accented fixture names intact.
// ---------------------------------------------------------------------------

function bytesToText(bytes: number[]): string {
  const buffer = Buffer.from(bytes);
  const utf8 = buffer.toString('utf8');
  return utf8.includes('\uFFFD') ? buffer.toString('latin1') : utf8;
}

function decodePdfLiteral(literal: string): string {
  const bytes: number[] = [];
  const escapes: Record<string, number> = {
    n: 10,
    r: 13,
    t: 9,
    b: 8,
    f: 12,
    '(': 40,
    ')': 41,
    '\\': 92,
  };
  for (let i = 0; i < literal.length; i += 1) {
    const char = literal[i]!;
    if (char !== '\\') {
      bytes.push(char.charCodeAt(0) & 0xff);
      continue;
    }
    const next = literal[++i] ?? '';
    if (/[0-7]/.test(next)) {
      let octal = next;
      while (octal.length < 3 && /[0-7]/.test(literal[i + 1] ?? '')) octal += literal[++i];
      bytes.push(parseInt(octal, 8));
    } else {
      expect(next in escapes, `supported PDF string escape \\${next}`).toBe(true);
      bytes.push(escapes[next]!);
    }
  }
  return bytesToText(bytes);
}

function pdfText(bytes: Buffer): string {
  const raw = bytes.toString('latin1');
  const parts: string[] = [];
  const strings = /\(((?:\\.|[^\\()])*)\)/g;
  for (const match of raw.matchAll(strings)) parts.push(decodePdfLiteral(match[1]!));
  return parts.join('\n');
}

function evidenceText(world: EvidenceWorld): string {
  expect(world.pdfBytes, 'a PDF document was delivered').toBeDefined();
  if (!world.pdfTextContent) world.pdfTextContent = pdfText(world.pdfBytes!);
  return world.pdfTextContent;
}

// The seven canonical key=value lines of contracts/hu03-record-update.md appear
// in the document as one contiguous block per update, in order, with no trailing
// newline. Only the PDF text is used here: no HUBX access is involved.
const CANONICAL_PATTERN =
  /shipment_id=[^\r\n]+\r?\nsequence=[1-9][0-9]*\r?\nstage=[^\r\n]+\r?\nresponsible_party_id=[^\r\n]+\r?\nassigned_partner_id=[^\r\n]+\r?\nrecorded_at=[^\r\n]+\r?\nprevious_fingerprint=[^\r\n]+/g;

function canonicalBlocks(text: string): string[] {
  return [...text.matchAll(CANONICAL_PATTERN)].map((match) => match[0].replace(/\r\n/g, '\n'));
}

function field(block: string, key: string): string {
  const match = block.match(new RegExp(`^${key}=(.*)$`, 'm'));
  expect(match, `${key} line in the canonical input`).toBeTruthy();
  return match![1]!;
}

async function requestEvidence(world: EvidenceWorld, code: string): Promise<void> {
  world.response = await world.api.get(`/api/shipments/${encodeURIComponent(code)}/evidence`, {
    headers: { ...authHeader(world), Accept: 'application/pdf' },
  });
  if (world.response.status() === 200) {
    world.pdfBytes = Buffer.from(await world.response.body());
    world.pdfTextContent = undefined;
  }
}

async function assertNoPdf(world: EvidenceWorld): Promise<void> {
  expect(world.pdfBytes, 'no PDF bytes on an error response').toBeUndefined();
  expect(world.response, 'HTTP response').toBeDefined();
  const contentType = world.response!.headers()['content-type'] ?? '';
  expect(contentType, 'error response is not a PDF').not.toContain('application/pdf');
  const raw = await world.response!.body();
  expect(raw.subarray(0, 5).toString('latin1'), 'error body is not a PDF').not.toBe('%PDF-');
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// "a shipment has three updates recorded" is defined once in
// HU04_view-traceability.steps.ts and reused here verbatim: it prepares a fresh
// shipment in "Assigned to partner" (In flight -> At destination airport ->
// Assigned to partner). "a shipment has no updates recorded" and
// "<actor> has signed in" are shared from common.steps.ts.

Given(
  /^an operator downloaded the evidence of a shipment with unaltered updates$/,
  async function (this: EvidenceWorld) {
    await prepareShipment(this, 'Assigned to partner');
    expect(this.shipmentCode, 'fixture shipment with three updates').toMatch(/^HBX-[0-9A-F]{8}$/);
    await signInAs(this, 'operator');
    await requestEvidence(this, this.shipmentCode!);
    expect(this.response!.status(), 'fixture evidence download through the gateway').toBe(200);
    expect(this.pdfBytes, 'PDF bytes downloaded by the fixture').toBeDefined();
  },
);

Given(/^nobody has signed in$/, async function (this: EvidenceWorld) {
  // No JWT in world: the request step below sends no Authorization header.
  this.token = undefined;
  this.actorRole = undefined;
});

// ---------------------------------------------------------------------------
// Download / refusal requests
// ---------------------------------------------------------------------------

When(/^the operator downloads the shipment's traceability evidence$/, async function (this: EvidenceWorld) {
  const code = this.shipmentCode!;
  if (this.page) {
    // @e2e: the critical happy path goes through the MFE only. The 3-update
    // fixture above used world.api (the HU-04/HU-05 precedent), but this action
    // and every assertion below use world.ui and the downloaded file.
    // Explicit short visibility assertion so a missing operator-only link fails
    // with the locator in the message instead of a generic Cucumber timeout.
    const link = this.ui.getByRole('link', { name: /download evidence/i });
    await expect(link, 'operator-only "Download evidence" link on the Dashboard').toBeVisible({ timeout: 5000 });
    await link.click();
    await expect(this.ui).toHaveURL(/\/download-evidence$/);

    const region = this.ui.getByRole('region', { name: /download evidence/i });
    await region.getByLabel(/shipment code/i).fill(code);
    const downloadPromise = this.ui.waitForEvent('download');
    await region.getByRole('button', { name: /download evidence/i }).click();
    this.download = await downloadPromise;

    this.evidencePath = join(tmpdir(), `hubx-evidence-${code}.pdf`);
    await this.download.saveAs(this.evidencePath);
    this.pdfBytes = await readFile(this.evidencePath);
    this.pdfTextContent = undefined;
    return;
  }

  await requestEvidence(this, code);
});

When(
  /^the operator tries to download the evidence of a code that matches no shipment$/,
  async function (this: EvidenceWorld) {
    await requestEvidence(this, UNKNOWN_CODE);
  },
);

When(/^the (ground partner|customer) tries to download the evidence of a shipment$/, async function (
  this: EvidenceWorld,
  role: string,
) {
  // Each denial scenario creates its own fresh shipment through the gateway
  // (HU-03 precedent), never a seed code. The denial assertion does not depend
  // on the shipment state, so an empty Created shipment is enough.
  await prepareShipment(this, 'Created');
  // prepareShipment signs in as the operator internally; restore the scenario
  // actor (the "<role> has signed in" Given) before the request.
  await signInAs(this, role as 'ground partner' | 'customer');
  await requestEvidence(this, this.shipmentCode!);
});

When(/^a person tries to download the evidence of a shipment$/, async function (this: EvidenceWorld) {
  // Fresh shipment through the gateway, then drop the fixture JWT so the person
  // really has no session when the request is made.
  await prepareShipment(this, 'Created');
  this.token = undefined;
  expect(this.token, 'no signed-in session').toBeUndefined();
  this.response = await this.api.get(`/api/shipments/${encodeURIComponent(this.shipmentCode!)}/evidence`, {
    headers: { Accept: 'application/pdf' },
  });
});

// ---------------------------------------------------------------------------
// Assertions
// ---------------------------------------------------------------------------

Then(/^they receive a document with the shipment's data$/, async function (this: EvidenceWorld) {
  if (this.page) {
    expect(this.download, 'browser download event').toBeDefined();
    expect(this.download!.suggestedFilename(), 'downloaded file name').toMatch(
      /^evidence-HBX-[0-9A-F]{8}\.pdf$/,
    );
  } else {
    expect(this.response?.status(), 'evidence response status').toBe(200);
    const contentType = this.response!.headers()['content-type'] ?? '';
    expect(contentType, 'evidence response content type').toContain('application/pdf');
  }

  expect(this.pdfBytes, 'PDF bytes').toBeDefined();
  expect(this.pdfBytes!.subarray(0, 5).toString('latin1'), 'PDF magic bytes').toBe('%PDF-');

  const text = evidenceText(this);
  expect(text, 'document heading').toContain('HUBX traceability evidence');
  expect(text, 'shipment code').toContain(this.shipmentCode!);
  expect(text, 'origin').toContain('Bogotá');
  expect(text, 'destination').toContain('Miami');
  expect(text, 'flight').toContain('AV-204');
  expect(text, 'weight').toMatch(/\b100(?:\.0)?\b/);
  const owner = process.env.BENDER_OTHER_CUSTOMER_EMAIL ?? 'andina@hubx.co';
  expect(text, 'customer').toContain(CUSTOMER_NAMES[owner] ?? 'Distribuciones Andina');
  expect(text, 'current stage').toContain(this.currentStage!);
  expect(text, 'internal-consistency scope statement').toMatch(/consistent with itself/i);
  expect(text, 'no-guarantee scope statement').toMatch(/did not rebuild the whole history/i);
  // Human decision 2026-11-25 (PR #43): three sentences, each contiguous (one literal string of its own;
  // pdfText joins literals with a newline, so a sentence found whole cannot span two strings).
  for (const sentence of NAMES_STATEMENT) {
    expect(text, `names statement: "${sentence}"`).toContain(sentence);
  }
});

Then(
  /^with each update, its stage, its responsible party, its date and its fingerprint$/,
  async function (this: EvidenceWorld) {
    const text = evidenceText(this);
    expect(text, 'Updates: section').toContain('Updates:');

    // Chain order: the three fixture stages must appear oldest first.
    let from = 0;
    for (const stage of ['In flight', 'At destination airport', 'Assigned to partner']) {
      const index = text.indexOf(stage, from);
      expect(index, `update "${stage}" in chain order`).toBeGreaterThanOrEqual(0);
      from = index + stage.length;
    }

    expect(text, 'responsible party on the updates').toContain(OPERATOR_NAME);
    const dates = text.match(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/g) ?? [];
    expect(dates.length, 'a date for every update').toBeGreaterThanOrEqual(3);
    const fingerprints = text.match(HEX64_GLOBAL) ?? [];
    expect(fingerprints.length, 'a fingerprint for every update').toBeGreaterThanOrEqual(3);
    expect(fingerprints.every((value) => FINGERPRINT.test(value)), 'lowercase 64-hex fingerprints').toBe(true);
  },
);

Then(/^with the date on which the document was generated$/, async function (this: EvidenceWorld) {
  const text = evidenceText(this);
  expect(text, 'generation-date line').toContain('Generated at:');
  const match = text.match(/Generated at:\s*(\d{4}-\d{2}-\d{2}T[^\s]+)/);
  expect(match, 'generation timestamp in the document').toBeTruthy();
  expect(Number.isNaN(Date.parse(match![1]!)), 'parseable generation timestamp').toBe(false);
  expect(text, 'verification instructions').toContain('How to verify the fingerprints:');
});

// ---------------------------------------------------------------------------
// Third-party verification: only the PDF text is read; no HTTP request is made.
// ---------------------------------------------------------------------------

When(
  /^a third party without access to HUBX follows the document's instructions to check the fingerprints$/,
  async function (this: EvidenceWorld) {
    // Drop the fixture JWT: the third party has no HUBX session. This step makes
    // zero HTTP calls and recomputes exclusively from the downloaded document.
    this.token = undefined;
    const text = evidenceText(this);
    const blocks = canonicalBlocks(text);
    expect(blocks.length, 'one canonical input block per update').toBe(3);

    const computed: string[] = [];
    for (let i = 0; i < blocks.length; i += 1) {
      const block = blocks[i]!;
      const hash = createHash('sha256').update(block, 'utf8').digest('hex');
      expect(hash, 'SHA-256 of the canonical input').toMatch(FINGERPRINT);
      expect(text, `fingerprint ${hash} printed in the document`).toMatch(
        new RegExp(`(?<!previous_)fingerprint[:=\\s]+${hash}`, 'i'),
      );
      if (i === 0) {
        expect(field(block, 'previous_fingerprint'), 'first update has no predecessor').toBe('-');
      } else {
        expect(field(block, 'previous_fingerprint'), `update ${i + 1} links to update ${i}`).toBe(computed[i - 1]!);
      }
      computed.push(hash);
    }
    this.recomputedFingerprints = computed;
  },
);

Then(/^the fingerprints they obtain match the ones in the document$/, async function (this: EvidenceWorld) {
  const recomputed = this.recomputedFingerprints;
  expect(recomputed?.length, 'three recomputed fingerprints').toBe(3);
  const text = evidenceText(this);
  const printed = [...text.matchAll(/(?<!previous_)fingerprint[:=\s]+([0-9a-f]{64})/gi)].map((match) =>
    match[1]!.toLowerCase(),
  );
  expect(printed, 'fingerprints printed in the document, in chain order').toEqual(recomputed);
});

Then(/^it states that the shipment has no updates yet$/, async function (this: EvidenceWorld) {
  expect(this.response?.status(), 'evidence response status').toBe(200);
  const text = evidenceText(this);
  expect(text, 'explicit empty-chain statement').toContain('The shipment has no updates yet.');
  expect(canonicalBlocks(text).length, 'no update chain in the document').toBe(0);
});

Then(/^they are told they do not have permission to download it$/, async function (this: EvidenceWorld) {
  expect(this.response?.status(), '403 for ground partner or customer').toBe(403);
  const body = (await this.response!.json()) as { error?: string; message?: string };
  expect(body.error, 'error code').toBe('FORBIDDEN');
  expect(body.message, 'error message').toBe('You do not have permission to download evidence');
  await assertNoPdf(this);
});

Then(/^they are asked to sign in$/, async function (this: EvidenceWorld) {
  expect(this.response?.status(), '401 without a session').toBe(401);
  const body = (await this.response!.json()) as { error?: string; message?: string };
  expect(body.error, 'error code').toBe('UNAUTHORIZED');
  // The gateway validates the JWT first on this route: the message comes from
  // contracts/auth-sign-in.md, not from a HU-10-specific body.
  expect(body.message, 'error message').toBe('Authorization header required');
  await assertNoPdf(this);
});

Then(/^they receive no document$/, async function (this: EvidenceWorld) {
  await assertNoPdf(this);
});
