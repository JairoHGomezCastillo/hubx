/**
 * Unit tests for CreateShipment page.
 * Tests form rendering, submission, success display, and error handling.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { CreateShipment } from './CreateShipment';
import { AuthProvider } from '../contexts/AuthContext';
import * as shipmentService from '../services/shipmentService';
import * as authService from '../services/authService';

// Mock services
vi.mock('../services/shipmentService');
vi.mock('../services/authService');

describe('CreateShipment', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render form with all labeled inputs for operators', () => {
    // Mock operator user
    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    expect(screen.getByRole('heading', { name: /create shipment/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/origin/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/flight/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/weight \(kg\)/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/customer email/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create shipment/i })).toBeInTheDocument();
  });

  it('should submit form with correct data', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    const mockShipment = {
      code: 'HBX-7F3A9C21',
      origin: 'Bogotá',
      destination: 'Miami',
      flight: 'AV-204',
      weightKg: 120,
      customer: {
        email: 'andina@hubx.co',
        name: 'Distribuciones Andina',
      },
      stage: 'Created',
      createdAt: '2026-11-24T15:04:05Z',
    };

    vi.mocked(shipmentService.createShipment).mockResolvedValue(mockShipment);

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    // Fill form
    await user.type(screen.getByLabelText(/origin/i), 'Bogotá');
    await user.type(screen.getByLabelText(/destination/i), 'Miami');
    await user.type(screen.getByLabelText(/flight/i), 'AV-204');
    await user.type(screen.getByLabelText(/weight \(kg\)/i), '120');
    await user.type(screen.getByLabelText(/customer email/i), 'andina@hubx.co');

    // Submit
    await user.click(screen.getByRole('button', { name: /create shipment/i }));

    // Verify service was called with correct data
    await waitFor(() => {
      expect(shipmentService.createShipment).toHaveBeenCalledWith({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        customerEmail: 'andina@hubx.co',
      });
    });
  });

  it('should show success message with code and stage after creation', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    const mockShipment = {
      code: 'HBX-7F3A9C21',
      origin: 'Bogotá',
      destination: 'Miami',
      flight: 'AV-204',
      weightKg: 120,
      customer: {
        email: 'andina@hubx.co',
        name: 'Distribuciones Andina',
      },
      stage: 'Created',
      createdAt: '2026-11-24T15:04:05Z',
    };

    vi.mocked(shipmentService.createShipment).mockResolvedValue(mockShipment);

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    // Fill and submit form
    await user.type(screen.getByLabelText(/origin/i), 'Bogotá');
    await user.type(screen.getByLabelText(/destination/i), 'Miami');
    await user.type(screen.getByLabelText(/flight/i), 'AV-204');
    await user.type(screen.getByLabelText(/weight \(kg\)/i), '120');
    await user.type(screen.getByLabelText(/customer email/i), 'andina@hubx.co');
    await user.click(screen.getByRole('button', { name: /create shipment/i }));

    // Verify success display
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /shipment created/i })).toBeInTheDocument();
      expect(screen.getByText(/code:\s*HBX-7F3A9C21/i)).toBeInTheDocument();
      expect(screen.getByText(/stage:\s*Created/i)).toBeInTheDocument();
    });
  });

  it('should display server error message in alert region', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    vi.mocked(shipmentService.createShipment).mockRejectedValue(
      new Error('The origin and the destination must be different')
    );

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    // Fill and submit form with invalid data
    await user.type(screen.getByLabelText(/origin/i), 'Bogotá');
    await user.type(screen.getByLabelText(/destination/i), 'Bogotá');
    await user.type(screen.getByLabelText(/flight/i), 'AV-100');
    await user.type(screen.getByLabelText(/weight \(kg\)/i), '100');
    await user.type(screen.getByLabelText(/customer email/i), 'andina@hubx.co');
    await user.click(screen.getByRole('button', { name: /create shipment/i }));

    // Verify error is displayed in alert
    await waitFor(() => {
      const alert = screen.getByRole('alert');
      expect(alert).toHaveTextContent('The origin and the destination must be different');
    });
  });

  it('should show permission message and hide form for non-operators', () => {
    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('ana@hubx.co', 'customer')
    );

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    expect(screen.getByText(/you do not have permission to create shipments/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/origin/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/destination/i)).not.toBeInTheDocument();
  });

  it('should disable form during submission', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    vi.mocked(shipmentService.createShipment).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    await user.type(screen.getByLabelText(/origin/i), 'Bogotá');
    await user.type(screen.getByLabelText(/destination/i), 'Miami');
    await user.type(screen.getByLabelText(/flight/i), 'AV-204');
    await user.type(screen.getByLabelText(/weight \(kg\)/i), '120');
    await user.type(screen.getByLabelText(/customer email/i), 'andina@hubx.co');

    const submitButton = screen.getByRole('button', { name: /create shipment/i }) as HTMLButtonElement;
    await user.click(submitButton);

    // Form should be disabled during submission
    expect(submitButton.disabled).toBe(true);
    expect(screen.getByLabelText(/origin/i)).toBeDisabled();
  });

  it('should handle weight as decimal number', async () => {
    const user = userEvent.setup();

    vi.mocked(authService.getToken).mockReturnValue(
      createMockToken('operator@hubx.co', 'operator')
    );

    const mockShipment = {
      code: 'HBX-12345678',
      origin: 'Bogotá',
      destination: 'Miami',
      flight: 'AV-100',
      weightKg: 0.1,
      customer: {
        email: 'andina@hubx.co',
        name: 'Distribuciones Andina',
      },
      stage: 'Created',
      createdAt: '2026-11-24T15:04:05Z',
    };

    vi.mocked(shipmentService.createShipment).mockResolvedValue(mockShipment);

    render(
      <AuthProvider>
        <CreateShipment />
      </AuthProvider>
    );

    await user.type(screen.getByLabelText(/origin/i), 'Bogotá');
    await user.type(screen.getByLabelText(/destination/i), 'Miami');
    await user.type(screen.getByLabelText(/flight/i), 'AV-100');
    await user.type(screen.getByLabelText(/weight \(kg\)/i), '0.1');
    await user.type(screen.getByLabelText(/customer email/i), 'andina@hubx.co');
    await user.click(screen.getByRole('button', { name: /create shipment/i }));

    await waitFor(() => {
      expect(shipmentService.createShipment).toHaveBeenCalledWith({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 0.1,
        customerEmail: 'andina@hubx.co',
      });
    });
  });
});

// Helper to create mock JWT token
function createMockToken(email: string, role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(
    JSON.stringify({
      email,
      role,
      exp: Math.floor(Date.now() / 1000) + 3600,
    })
  );
  return `${header}.${payload}.signature`;
}
