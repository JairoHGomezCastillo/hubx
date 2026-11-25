/**
 * Unit tests for StageCounts component.
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import StageCounts from './StageCounts';
import type { StageCountOutcome } from '../../services/stageCountService';

describe('StageCounts', () => {
  it('should render heading and button', () => {
    const mockFetch = vi.fn().mockResolvedValue({ kind: 'success', data: { stages: [], total: 0 } });
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    expect(screen.getByRole('heading', { name: /shipment count by stage/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /load stage counts/i })).toBeInTheDocument();
  });

  it('should show loading state when fetching', async () => {
    const mockFetch = vi.fn().mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    expect(screen.getByText(/loading stage counts/i)).toBeInTheDocument();
  });

  it('should render six stages in journey order with counts', async () => {
    const mockData = {
      stages: [
        { stage: 'Created', count: 2 },
        { stage: 'In flight', count: 3 },
        { stage: 'At destination airport', count: 1 },
        { stage: 'Assigned to partner', count: 0 },
        { stage: 'Out for delivery', count: 2 },
        { stage: 'Delivered', count: 5 },
      ],
      total: 13,
    };
    const mockFetch = vi.fn().mockResolvedValue({ kind: 'success', data: mockData });
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByRole('region', { name: /stage counts/i })).toBeInTheDocument();
    });

    const region = screen.getByRole('region', { name: /stage counts/i });
    const rows = region.querySelectorAll('tbody tr');
    expect(rows).toHaveLength(6);

    // Check journey order
    expect(rows[0]).toHaveAccessibleName(/created.*2/i);
    expect(rows[1]).toHaveAccessibleName(/in flight.*3/i);
    expect(rows[2]).toHaveAccessibleName(/at destination airport.*1/i);
    expect(rows[3]).toHaveAccessibleName(/assigned to partner.*0/i);
    expect(rows[4]).toHaveAccessibleName(/out for delivery.*2/i);
    expect(rows[5]).toHaveAccessibleName(/delivered.*5/i);

    // Check total
    expect(screen.getByText(/total: 13 shipments/i)).toBeInTheDocument();
  });

  it('should show zeros for empty stages', async () => {
    const mockData = {
      stages: [
        { stage: 'Created', count: 0 },
        { stage: 'In flight', count: 0 },
        { stage: 'At destination airport', count: 0 },
        { stage: 'Assigned to partner', count: 0 },
        { stage: 'Out for delivery', count: 0 },
        { stage: 'Delivered', count: 0 },
      ],
      total: 0,
    };
    const mockFetch = vi.fn().mockResolvedValue({ kind: 'success', data: mockData });
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByRole('region', { name: /stage counts/i })).toBeInTheDocument();
    });

    const rows = screen.getByRole('region', { name: /stage counts/i }).querySelectorAll('tbody tr');
    expect(rows).toHaveLength(6);
    rows.forEach((row) => {
      expect(row).toHaveAccessibleName(/.*0/);
    });

    expect(screen.getByText(/total: 0 shipments/i)).toBeInTheDocument();
  });

  it('should show forbidden message when permission denied', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      kind: 'forbidden',
      message: 'You do not have permission to see that information',
    } as StageCountOutcome);
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/you do not have permission to see that information/i)).toBeInTheDocument();
    });

    expect(screen.queryByRole('region', { name: /stage counts/i })).not.toBeInTheDocument();
  });

  it('should call onSessionExpired when session expires', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ kind: 'session-expired' } as StageCountOutcome);
    const mockSessionExpired = vi.fn();
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={mockSessionExpired} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(mockSessionExpired).toHaveBeenCalledTimes(1);
    });
  });

  it('should show error message on failure', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ kind: 'error' } as StageCountOutcome);
    render(<StageCounts onGetStageCounts={mockFetch} onSessionExpired={vi.fn()} />);

    const button = screen.getByRole('button', { name: /load stage counts/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/the stage counts could not be loaded/i)).toBeInTheDocument();
    });
  });
});
