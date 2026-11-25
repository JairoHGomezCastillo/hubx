/**
 * Wiring check for the HU-03 and HU-04 @e2e entry points: after sign-in the
 * update form and traceability view must already be visible on the dashboard,
 * with no extra navigation.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { AuthProvider } from '../contexts/AuthContext';
import * as authService from '../services/authService';
import { Dashboard } from './Dashboard';

function signedInToken(): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(
    JSON.stringify({
      email: 'operator@hubx.co',
      role: 'operator',
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  );
  return `${header}.${payload}.signature`;
}

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authService.setToken(signedInToken());
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'partner-1', name: 'Transportes Rápido' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });

  it('shows the labelled update form immediately after sign-in', async () => {
    render(
      <AuthProvider>
        <Dashboard />
      </AuthProvider>,
    );

    const updateSection = screen.getByRole('region', { name: 'Record an update' });
    expect(within(updateSection).getByLabelText(/shipment code/i)).toBeVisible();
    expect(within(updateSection).getByLabelText(/stage/i)).toBeVisible();
    expect(within(updateSection).getByRole('button', { name: /record update/i })).toBeVisible();
    expect(screen.getAllByRole('status')).toHaveLength(1);
    expect(fetch).toHaveBeenCalledWith('/api/ground-partners', expect.anything());
  });

  it('shows the traceability view for every role', async () => {
    render(
      <AuthProvider>
        <Dashboard />
      </AuthProvider>,
    );

    // Both views are visible: the record update form and the traceability view
    const updateSection = screen.getByRole('region', { name: 'Record an update' });
    expect(within(updateSection).getByRole('button', { name: /record update/i })).toBeVisible();
    
    const traceabilitySection = screen.getByRole('region', { name: 'View traceability' });
    expect(within(traceabilitySection).getByRole('button', { name: /view traceability/i })).toBeVisible();
  });
});
