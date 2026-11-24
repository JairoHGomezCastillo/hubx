/**
 * Unit tests for the HU-05 verify integrity view.
 * Locators mirror the @e2e scenario: labels and roles only.
 */

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import VerifyIntegrity from './VerifyIntegrity';
import type { VerificationResult, VerifyIntegrityOutcome } from './VerifyIntegrity';

const INTACT_RESULT: VerificationResult = {
  shipmentCode: 'HBX-TEST-0001',
  intact: true,
  updatesChecked: 3,
  firstAlteredUpdate: null,
};

const INTACT_ZERO: VerificationResult = {
  shipmentCode: 'HBX-TEST-0002',
  intact: true,
  updatesChecked: 0,
  firstAlteredUpdate: null,
};

const NOT_INTACT: VerificationResult = {
  shipmentCode: 'HBX-TEST-0003',
  intact: false,
  updatesChecked: 3,
  firstAlteredUpdate: 2,
};

function responder(outcome: VerifyIntegrityOutcome) {
  return vi.fn(async (_code: string): Promise<VerifyIntegrityOutcome> => outcome);
}

function renderView(
  onVerify: (code: string) => Promise<VerifyIntegrityOutcome>,
  onSessionExpired = vi.fn(),
) {
  render(
    <VerifyIntegrity
      onVerify={onVerify}
      onSessionExpired={onSessionExpired}
    />,
  );
  return { onSessionExpired };
}

describe('VerifyIntegrity', () => {
  it('exposes a labelled code input, a verify button and one status region', async () => {
    const user = userEvent.setup();
    const onVerify = responder({ kind: 'success', result: INTACT_RESULT });
    renderView(onVerify);

    expect(screen.getByLabelText(/shipment code/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /verify integrity/i })).toBeInTheDocument();
    expect(screen.getAllByRole('status')).toHaveLength(1);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0001');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    expect(onVerify).toHaveBeenCalledWith('HBX-TEST-0001');
  });

  it('shows "The chain is intact" and "3 updates checked" as contiguous text when intact', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'success', result: INTACT_RESULT }));

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0001');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    const status = await screen.findByRole('status');
    expect(status).toHaveTextContent('The chain is intact');
    expect(status).toHaveTextContent('3 updates checked');
  });

  it('shows "There are no updates to check" when updatesChecked is 0', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'success', result: INTACT_ZERO }));

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0002');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    const status = await screen.findByRole('status');
    expect(status).toHaveTextContent('The chain is intact');
    expect(status).toHaveTextContent('There are no updates to check');
  });

  it('shows "The chain is not intact" and "first alteration is in update 2" when not intact', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'success', result: NOT_INTACT }));

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0003');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    const status = await screen.findByRole('status');
    expect(status).toHaveTextContent('The chain is not intact');
    expect(status).toHaveTextContent('The first alteration is in update 2');
  });

  it('shows "No shipment exists with that code" for 404 SHIPMENT_NOT_FOUND', async () => {
    const user = userEvent.setup();
    renderView(
      responder({
        kind: 'httpError',
        status: 404,
        code: 'SHIPMENT_NOT_FOUND',
        message: 'No shipment exists with that code',
      }),
    );

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-UNKNOWN');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('No shipment exists with that code');
  });

  it('distinguishes a network failure from an HTTP error', async () => {
    const user = userEvent.setup();
    renderView(responder({ kind: 'noResponse' }));

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0001');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('No response from the server. Try again.');
  });

  it('signals an expired session when the transport reports 401', async () => {
    const user = userEvent.setup();
    const { onSessionExpired } = renderView(
      responder({
        kind: 'httpError',
        status: 401,
        code: 'UNAUTHORIZED',
        message: 'Session expired. Please sign in again.',
      }),
    );

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0001');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('Session expired. Please sign in again.');
    expect(onSessionExpired).toHaveBeenCalled();
  });

  it('shows a generic message for unexpected HTTP errors', async () => {
    const user = userEvent.setup();
    renderView(
      responder({
        kind: 'httpError',
        status: 500,
        code: 'INTERNAL_SERVER_ERROR',
        message: 'An unexpected error occurred',
      }),
    );

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-0001');
    await user.click(screen.getByRole('button', { name: /verify integrity/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('An unexpected error occurred');
  });
});
