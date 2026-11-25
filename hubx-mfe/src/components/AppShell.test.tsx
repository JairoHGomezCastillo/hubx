import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { AuthProvider } from '../contexts/AuthContext';
import * as authService from '../services/authService';
import { AppShell } from './AppShell';

describe('AppShell', () => {
  beforeEach(() => {
    const payload = btoa(JSON.stringify({ email: 'operator@hubx.co', role: 'operator' }));
    authService.setToken(`header.${payload}.signature`);
    window.history.replaceState(null, '', '/shipments/new');
  });

  function renderShell(showBack = false) {
    render(
      <AuthProvider>
        <AppShell showBack={showBack}>content</AppShell>
      </AuthProvider>,
    );
  }

  it('shows the signed-in user and role, without a back button by default', () => {
    renderShell();
    expect(screen.getByText('operator@hubx.co')).toBeVisible();
    expect(screen.getByText(/Role:\s*operator/)).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Back to Dashboard' })).not.toBeInTheDocument();
  });

  it('goes back to the dashboard', async () => {
    renderShell(true);
    await userEvent.click(screen.getByRole('button', { name: 'Back to Dashboard' }));
    expect(window.location.pathname).toBe('/dashboard');
  });

  it('signs out and returns to sign-in', async () => {
    renderShell();
    await userEvent.click(screen.getByRole('button', { name: 'Sign Out' }));
    expect(authService.getToken()).toBeNull();
    expect(window.location.pathname).toBe('/sign-in');
  });
});
