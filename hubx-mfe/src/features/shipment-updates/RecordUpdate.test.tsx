/**
 * Unit tests for the HU-03 accessible update form.
 * Locators mirror the @e2e scenario: labels and roles only.
 */

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import RecordUpdate from './RecordUpdate';
import type { GroundPartner, RecordedUpdate, RecordUpdateBody, RecordUpdateOutcome } from './RecordUpdate';

const PARTNERS: GroundPartner[] = [{ id: 'partner-1', name: 'Transportes Rápido' }];

const RECORDED: RecordedUpdate = {
  id: 1,
  shipmentCode: 'HBX-SEED-0001',
  sequence: 1,
  stage: 'In flight',
  responsibleParty: {
    id: '00000000-0000-4000-8000-000000000001',
    name: 'Laura Operator',
    role: 'operator',
  },
  assignedPartner: null,
  recordedAt: '2026-11-24T10:15:30Z',
  previousFingerprint: null,
  fingerprint: 'a'.repeat(64),
};

function responder(outcome: RecordUpdateOutcome) {
  return vi.fn(async (_code: string, _body: RecordUpdateBody): Promise<RecordUpdateOutcome> => outcome);
}

function renderForm(
  onRecordUpdate: (code: string, body: RecordUpdateBody) => Promise<RecordUpdateOutcome>,
  onSessionExpired = vi.fn(),
  groundPartners: readonly GroundPartner[] = PARTNERS,
) {
  render(
    <RecordUpdate
      groundPartners={groundPartners}
      onRecordUpdate={onRecordUpdate}
      onSessionExpired={onSessionExpired}
    />,
  );
  return { onSessionExpired };
}

describe('RecordUpdate', () => {
  it('exposes labelled controls and one status region with stage, responsible party and date', async () => {
    const user = userEvent.setup();
    const onRecordUpdate = responder({ kind: 'success', update: RECORDED });
    renderForm(onRecordUpdate);

    expect(screen.getByLabelText(/shipment code/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/stage/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /record update/i })).toBeInTheDocument();
    expect(screen.getAllByRole('status')).toHaveLength(1);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'In flight');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    const status = await screen.findByRole('status');
    expect(status).toHaveTextContent('In flight');
    expect(status).toHaveTextContent('Laura Operator');
    expect(status).toHaveTextContent(/\d{4}-\d{2}-\d{2}/);
    expect(onRecordUpdate).toHaveBeenCalledWith('HBX-SEED-0001', {
      stage: 'In flight',
      assignedPartnerId: null,
    });
  });

  it('sends the selected ground partner id only for the assignment stage', async () => {
    const user = userEvent.setup();
    const onRecordUpdate = responder({ kind: 'success', update: RECORDED });
    renderForm(onRecordUpdate);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'Assigned to partner');
    await user.selectOptions(screen.getByLabelText(/ground partner/i), 'partner-1');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    expect(onRecordUpdate).toHaveBeenCalledWith('HBX-SEED-0001', {
      stage: 'Assigned to partner',
      assignedPartnerId: 'partner-1',
    });
  });

  it('rejects an assignment without a partner before calling the transport', async () => {
    const user = userEvent.setup();
    const onRecordUpdate = responder({ kind: 'success', update: RECORDED });
    renderForm(onRecordUpdate);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'Assigned to partner');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    expect(await screen.findByRole('status')).toHaveTextContent(
      'You must name the ground partner responsible for the delivery.',
    );
    expect(onRecordUpdate).not.toHaveBeenCalled();
  });

  it('distinguishes a network failure from an HTTP error', async () => {
    const user = userEvent.setup();
    renderForm(responder({ kind: 'noResponse' }));

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'In flight');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('No response from the server. Try again.');
  });

  it('reports the contract message for an HTTP rejection', async () => {
    const user = userEvent.setup();
    renderForm(
      responder({
        kind: 'httpError',
        status: 409,
        code: 'OUT_OF_SEQUENCE',
        message: "That update does not follow the shipment's sequence.",
      }),
    );

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'In flight');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    expect(await screen.findByRole('status')).toHaveTextContent(
      "That update does not follow the shipment's sequence.",
    );
  });

  it('signals an expired session when the transport reports 401', async () => {
    const user = userEvent.setup();
    const { onSessionExpired } = renderForm(
      responder({
        kind: 'httpError',
        status: 401,
        code: 'SESSION_EXPIRED',
        message: 'Session expired; sign in again.',
      }),
    );

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'In flight');
    await user.click(screen.getByRole('button', { name: /record update/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('Session expired; sign in again.');
    expect(onSessionExpired).toHaveBeenCalled();
  });

  it('keeps the partner select disabled and explains an empty catalogue', async () => {
    const user = userEvent.setup();
    renderForm(responder({ kind: 'success', update: RECORDED }), vi.fn(), []);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-SEED-0001');
    await user.selectOptions(screen.getByLabelText(/stage/i), 'Assigned to partner');

    expect(screen.getByLabelText(/ground partner/i)).toBeDisabled();
    expect(screen.getByText('No ground partners available.')).toBeInTheDocument();
  });
});
