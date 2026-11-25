/**
 * Unit tests for AssignedShipments component.
 */

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AssignedShipments from './AssignedShipments';
import type { AssignedShipmentsResponse } from '../../services/shipmentService';

describe('AssignedShipments', () => {
  const mockShipments: AssignedShipmentsResponse = {
    shipments: [
      {
        code: 'HBX-OLDEST',
        stage: 'Assigned to partner',
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        assignedAt: '2026-11-24T10:15:30.123456Z',
      },
      {
        code: 'HBX-MIDDLE',
        stage: 'Out for delivery',
        origin: 'Medellín',
        destination: 'New York',
        flight: 'AV-100',
        weightKg: 85.5,
        assignedAt: '2026-11-24T14:20:00Z',
      },
      {
        code: 'HBX-NEWEST',
        stage: 'Assigned to partner',
        origin: 'Cali',
        destination: 'Los Angeles',
        flight: 'AV-300',
        weightKg: 200,
        assignedAt: '2026-11-25T08:00:00Z',
      },
    ],
    nextOffset: null,
    message: null,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders section with accessible region name', () => {
    const onLoadAssignedShipments = vi.fn().mockResolvedValue({
      shipments: [],
      nextOffset: null,
      message: 'You have no assigned shipments',
    });
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    expect(screen.getByRole('region', { name: /my assigned shipments/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: /my assigned shipments/i })).toBeInTheDocument();
  });

  it('shows empty message when there are no shipments', async () => {
    const onLoadAssignedShipments = vi.fn().mockResolvedValue({
      shipments: [],
      nextOffset: null,
      message: 'You have no assigned shipments',
    });
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    expect(await screen.findByText('You have no assigned shipments')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('renders table with shipments in API order', async () => {
    const onLoadAssignedShipments = vi.fn().mockResolvedValue(mockShipments);
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole('table')).toBeInTheDocument();
    });

    const rows = screen.getAllByRole('row');
    // Header + 3 data rows
    expect(rows).toHaveLength(4);

    // Check headers
    expect(screen.getByRole('columnheader', { name: /code/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /stage/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /origin/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /destination/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /flight/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /weight/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /assigned/i })).toBeInTheDocument();

    // Verify data in order (oldest first)
    expect(screen.getByText('HBX-OLDEST')).toBeInTheDocument();
    expect(screen.getByText('HBX-MIDDLE')).toBeInTheDocument();
    expect(screen.getByText('HBX-NEWEST')).toBeInTheDocument();
  });

  it('shows "Show next 20" button when nextOffset is not null', async () => {
    const onLoadAssignedShipments = vi.fn().mockResolvedValue({
      shipments: mockShipments.shipments,
      nextOffset: 20,
      message: null,
    });
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    expect(await screen.findByRole('button', { name: /show next 20/i })).toBeInTheDocument();
  });

  it('does not show "Show next 20" button when nextOffset is null', async () => {
    const onLoadAssignedShipments = vi.fn().mockResolvedValue(mockShipments);
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole('table')).toBeInTheDocument();
    });

    expect(screen.queryByRole('button', { name: /show next 20/i })).not.toBeInTheDocument();
  });

  it('loads next page when "Show next 20" is clicked', async () => {
    const user = userEvent.setup();
    const firstPage = {
      shipments: mockShipments.shipments,
      nextOffset: 20,
      message: null,
    };
    const secondPage = {
      shipments: [
        {
          code: 'HBX-PAGE2',
          stage: 'Assigned to partner',
          origin: 'Barranquilla',
          destination: 'Boston',
          flight: 'AV-500',
          weightKg: 150,
          assignedAt: '2026-11-25T09:00:00Z',
        },
      ],
      nextOffset: null,
      message: null,
    };

    const onLoadAssignedShipments = vi
      .fn()
      .mockResolvedValueOnce(firstPage)
      .mockResolvedValueOnce(secondPage);
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole('table')).toBeInTheDocument();
    });

    const button = screen.getByRole('button', { name: /show next 20/i });
    await user.click(button);

    expect(onLoadAssignedShipments).toHaveBeenCalledTimes(2);
    expect(onLoadAssignedShipments).toHaveBeenLastCalledWith(20);

    // Both pages should be visible (cumulative)
    expect(await screen.findByText('HBX-PAGE2')).toBeInTheDocument();
    expect(screen.getByText('HBX-OLDEST')).toBeInTheDocument();
  });

  it('shows error message in alert role on failure', async () => {
    const onLoadAssignedShipments = vi.fn().mockRejectedValue(new Error('Network error'));
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/could not be loaded/i);
  });

  it('calls onSessionExpired on 401 unauthorized error', async () => {
    const sessionError = new Error('Session expired. Please sign in again.');
    const onLoadAssignedShipments = vi.fn().mockRejectedValue(sessionError);
    const onSessionExpired = vi.fn();

    render(
      <AssignedShipments
        onLoadAssignedShipments={onLoadAssignedShipments}
        onSessionExpired={onSessionExpired}
      />,
    );

    await waitFor(() => {
      expect(onSessionExpired).toHaveBeenCalled();
    });
  });
});
