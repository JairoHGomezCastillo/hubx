/**
 * Unit tests for the HU-10 evidence view. Locators mirror the @e2e scenario:
 * region, label and button only, scoped to the download evidence region.
 */

import { describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import DownloadEvidence from './DownloadEvidence';
import type { DownloadEvidenceOutcome } from './DownloadEvidence';

const CODE = 'HBX-TEST-0001';

function responder(outcome: DownloadEvidenceOutcome) {
  return vi.fn(async (_code: string): Promise<DownloadEvidenceOutcome> => outcome);
}

function renderView(
  onDownload: (code: string) => Promise<DownloadEvidenceOutcome>,
  onSessionExpired = vi.fn(),
) {
  render(<DownloadEvidence onDownload={onDownload} onSessionExpired={onSessionExpired} />);
  return { onSessionExpired };
}

function region() {
  return screen.getByRole('region', { name: /download evidence/i });
}

async function downloadCode(user: ReturnType<typeof userEvent.setup>, code: string) {
  await user.type(within(region()).getByLabelText(/shipment code/i), code);
  await user.click(within(region()).getByRole('button', { name: /download evidence/i }));
}

describe('DownloadEvidence', () => {
  it('exposes an accessible region with a labelled code input and a download button', async () => {
    const user = userEvent.setup();
    const onDownload = responder({ kind: 'downloaded', filename: `evidence-${CODE}.pdf` });
    renderView(onDownload);

    expect(within(region()).getByLabelText(/shipment code/i)).toBeVisible();
    expect(within(region()).getByRole('button', { name: /download evidence/i })).toBeVisible();
    expect(screen.getAllByRole('status')).toHaveLength(1);

    await downloadCode(user, CODE);
    expect(onDownload).toHaveBeenCalledWith(CODE);
  });

  it('trims the shipment code before calling the transport', async () => {
    const user = userEvent.setup();
    const onDownload = responder({ kind: 'downloaded', filename: `evidence-${CODE}.pdf` });
    renderView(onDownload);

    await downloadCode(user, `  ${CODE}  `);

    expect(onDownload).toHaveBeenCalledWith(CODE);
  });

  it('reports the saved filename on success', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'downloaded', filename: `evidence-${CODE}.pdf` }));

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent(`evidence-${CODE}.pdf`);
  });

  it('reports the contract message for 403', async () => {
    const user = userEvent.setup();
    renderView(
      responder({ kind: 'forbidden', message: 'You do not have permission to download evidence' }),
    );

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent(
      'You do not have permission to download evidence',
    );
  });

  it('reports the contract message for 404', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'not-found', message: 'No shipment exists with that code' }));

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent('No shipment exists with that code');
  });

  it('reports the server message for other HTTP errors', async () => {
    const user = userEvent.setup();
    renderView(
      responder({
        kind: 'httpError',
        status: 500,
        message: 'An unexpected error occurred',
      }),
    );

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent('An unexpected error occurred');
  });

  it('distinguishes no response from an HTTP error', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'noResponse' }));

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent('No response from the server. Try again.');
  });

  it('signals an expired session and never starts a download', async () => {
    const user = userEvent.setup();
    const { onSessionExpired } = renderView(responder({ kind: 'unauthorized' }));

    await downloadCode(user, CODE);

    expect(await screen.findByRole('status')).toHaveTextContent('Session expired. Please sign in again.');
    expect(onSessionExpired).toHaveBeenCalled();
  });

  it('does not call the transport for an empty code', async () => {
    const user = userEvent.setup();
    const onDownload = responder({ kind: 'downloaded', filename: `evidence-${CODE}.pdf` });
    renderView(onDownload);

    await user.click(within(region()).getByRole('button', { name: /download evidence/i }));

    expect(onDownload).not.toHaveBeenCalled();
    expect(await screen.findByRole('status')).toHaveTextContent('Please enter a shipment code.');
  });
});
