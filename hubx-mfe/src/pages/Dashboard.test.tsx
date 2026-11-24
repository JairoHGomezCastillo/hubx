/**
 * Wiring check for the HU-03 @e2e entry point: after sign-in the update form
 * must already be visible on the dashboard, with no extra navigation.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
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

    expect(await screen.findByLabelText(/shipment code/i)).toBeVisible();
    expect(screen.getByLabelText(/stage/i)).toBeVisible();
    expect(screen.getByRole('button', { name: /record update/i })).toBeVisible();
    expect(screen.getAllByRole('status')).toHaveLength(1);
    expect(fetch).toHaveBeenCalledWith('/api/ground-partners', expect.anything());
  });
});
