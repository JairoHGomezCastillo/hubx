/**
 * Unit tests for ViewTraceability: form, result list order, fields, empty chain, not-found message.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { TraceabilityData, TraceabilityOutcome } from '../../api/client';
import ViewTraceability from './ViewTraceability';

const TRACEABILITY: TraceabilityData = {
  shipmentCode: 'HBX-TEST-CODE',
  stage: 'Assigned to partner',
  updates: [
    {
      sequence: 1,
      stage: 'In flight',
      responsibleParty: { id: 'user-1', name: 'Laura Operator', role: 'operator' },
      assignedPartner: null,
      recordedAt: '2026-11-25T10:15:30.123456789Z',
      previousFingerprint: null,
      fingerprint: 'a'.repeat(64),
    },
    {
      sequence: 2,
      stage: 'At destination airport',
      responsibleParty: { id: 'user-1', name: 'Laura Operator', role: 'operator' },
      assignedPartner: null,
      recordedAt: '2026-11-25T12:00:00.5Z',
      previousFingerprint: 'a'.repeat(64),
      fingerprint: 'b'.repeat(64),
    },
    {
      sequence: 3,
      stage: 'Assigned to partner',
      responsibleParty: { id: 'user-1', name: 'Laura Operator', role: 'operator' },
      assignedPartner: { id: 'partner-1', name: 'Transportes Rápido' },
      recordedAt: '2026-11-25T12:30:00Z',
      previousFingerprint: 'b'.repeat(64),
      fingerprint: 'c'.repeat(64),
    },
  ],
};

describe('ViewTraceability', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows the form with accessible label and button', () => {
    const onViewTraceability = vi.fn();
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    expect(screen.getByLabelText(/shipment code/i)).toBeVisible();
    expect(screen.getByRole('button', { name: /view traceability/i })).toBeVisible();
  });

  it('calls onViewTraceability with the entered code', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'ok', data: TRACEABILITY } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-CODE');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    await waitFor(() => {
      expect(onViewTraceability).toHaveBeenCalledWith('HBX-TEST-CODE');
    });
  });

  it('shows the heading, current stage and ordered list of updates', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'ok', data: TRACEABILITY } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-CODE');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    expect(await screen.findByRole('heading', { name: /traceability of HBX-TEST-CODE/i })).toBeVisible();
    expect(screen.getByText(/current stage: assigned to partner/i)).toBeVisible();

    const list = screen.getByRole('list', { name: /updates/i });
    expect(list).toBeVisible();
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(3);
  });

  it('shows stage, responsible party name and role, and date in each item', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'ok', data: TRACEABILITY } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-CODE');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    await screen.findByRole('heading', { name: /traceability of HBX-TEST-CODE/i });

    const items = screen.getAllByRole('listitem');
    
    // First item: In flight
    expect(items[0]).toHaveTextContent('In flight');
    expect(items[0]).toHaveTextContent('Laura Operator');
    expect(items[0]).toHaveTextContent('operator');
    expect(items[0]).toHaveTextContent(/2026-11-25/);

    // Third item: Assigned to partner (shows assigned partner)
    expect(items[2]).toHaveTextContent('Assigned to partner');
    expect(items[2]).toHaveTextContent('Transportes Rápido');
  });

  it('shows the empty chain message when there are no updates', async () => {
    const user = userEvent.setup();
    const emptyData: TraceabilityData = {
      shipmentCode: 'HBX-NO-UPDATES',
      stage: 'Created',
      updates: [],
    };
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'ok', data: emptyData } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-NO-UPDATES');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    expect(await screen.findByText(/current stage: created/i)).toBeVisible();
    expect(screen.getByText(/this shipment has no updates yet/i)).toBeVisible();
    expect(screen.queryByRole('list', { name: /updates/i })).not.toBeInTheDocument();
  });

  it('shows the not-found alert on 404', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'not-found' } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-99999999');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/no shipment exists with that code/i);
  });

  it('calls onSessionExpired on session-expired outcome', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'session-expired' } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-CODE');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    await waitFor(() => {
      expect(onSessionExpired).toHaveBeenCalled();
    });
  });

  it('shows a generic error alert on error outcome', async () => {
    const user = userEvent.setup();
    const onViewTraceability = vi.fn().mockResolvedValue({ kind: 'error' } as TraceabilityOutcome);
    const onSessionExpired = vi.fn();

    render(<ViewTraceability onViewTraceability={onViewTraceability} onSessionExpired={onSessionExpired} />);

    await user.type(screen.getByLabelText(/shipment code/i), 'HBX-TEST-CODE');
    await user.click(screen.getByRole('button', { name: /view traceability/i }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/the traceability could not be retrieved/i);
  });
});
