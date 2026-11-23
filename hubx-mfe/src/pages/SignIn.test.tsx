/**
 * Unit tests for SignIn page.
 * Tests form submission, error display, and navigation.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { SignIn } from './SignIn';
import { AuthProvider } from '../contexts/AuthContext';
import * as authService from '../services/authService';

// Mock authService
vi.mock('../services/authService');

describe('SignIn', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.clearAllMocks();
    
    // Reset location.href using vi.stubGlobal
    vi.stubGlobal('location', { href: '/sign-in' });
    
    vi.mocked(authService.getToken).mockReturnValue(null);
  });

  it('should render sign-in form', () => {
    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  it('should display demo user information', () => {
    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    expect(screen.getByText(/demo users:/i)).toBeInTheDocument();
    expect(screen.getByText(/operator@hubx.co/i)).toBeInTheDocument();
    expect(screen.getByText(/rapido@hubx.co/i)).toBeInTheDocument();
    expect(screen.getByText(/ana@hubx.co/i)).toBeInTheDocument();
  });

  it('should submit form with trimmed email', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.signIn).mockResolvedValue({
      token: 'test.jwt.token',
      user: {
        email: 'operator@hubx.co',
        role: 'operator',
      },
    });
    vi.mocked(authService.setToken).mockImplementation(() => {});

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(emailInput, '  operator@hubx.co  ');
    await user.type(passwordInput, 'Hubx-Demo-2026');
    await user.click(submitButton);

    await waitFor(() => {
      expect(authService.signIn).toHaveBeenCalledWith(
        'operator@hubx.co',
        'Hubx-Demo-2026'
      );
    });
  });

  it('should display error on 401 Unauthorized', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.signIn).mockRejectedValue(
      new Error('Invalid credentials')
    );

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(emailInput, 'wrong@hubx.co');
    await user.type(passwordInput, 'wrong-password');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid credentials');
    });
  });

  it('should display error on 400 Bad Request', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.signIn).mockRejectedValue(
      new Error('Email and password are required')
    );

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    // Fill inputs to bypass HTML5 validation
    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(emailInput, 'test@hubx.co');
    await user.type(passwordInput, 'password');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Email and password are required'
      );
    });
  });

  it('should display session expired message from sessionStorage', () => {
    sessionStorage.setItem(
      'session_expired_message',
      'Session expired. Please sign in again.'
    );

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Session expired. Please sign in again.'
    );
    expect(sessionStorage.getItem('session_expired_message')).toBeNull();
  });

  it('should redirect to dashboard when already authenticated', async () => {
    const validToken = btoa(
      JSON.stringify({ alg: 'HS256', typ: 'JWT' })
    ) + '.' + btoa(
      JSON.stringify({
        email: 'operator@hubx.co',
        role: 'operator',
        exp: Math.floor(Date.now() / 1000) + 3600,
      })
    ) + '.signature';

    vi.mocked(authService.getToken).mockReturnValue(validToken);

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(globalThis.location.href).toBe('/dashboard');
    });
  });

  it('should disable form during submission', async () => {
    const user = userEvent.setup();

    // Make signIn take some time to resolve
    vi.mocked(authService.signIn).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    const emailInput = screen.getByLabelText(/email/i) as HTMLInputElement;
    const passwordInput = screen.getByLabelText(/password/i) as HTMLInputElement;
    const submitButton = screen.getByRole('button', { name: /sign in/i }) as HTMLButtonElement;

    await user.type(emailInput, 'test@hubx.co');
    await user.type(passwordInput, 'password');
    await user.click(submitButton);

    // Form should be disabled during submission
    expect(emailInput.disabled).toBe(true);
    expect(passwordInput.disabled).toBe(true);
    expect(submitButton.disabled).toBe(true);
    expect(submitButton).toHaveTextContent('Signing in...');
  });

  it('should handle ground partner sign-in', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.signIn).mockResolvedValue({
      token: 'partner.jwt.token',
      user: {
        email: 'rapido@hubx.co',
        role: 'ground partner',
      },
    });
    vi.mocked(authService.setToken).mockImplementation(() => {});

    render(
      <AuthProvider>
        <SignIn />
      </AuthProvider>
    );

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(emailInput, 'rapido@hubx.co');
    await user.type(passwordInput, 'Hubx-Demo-2026');
    await user.click(submitButton);

    await waitFor(() => {
      expect(authService.signIn).toHaveBeenCalled();
    });
  });
});
