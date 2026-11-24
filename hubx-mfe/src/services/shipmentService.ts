/**
 * Shipment service for creating shipments.
 * Calls POST /api/shipments through the gateway.
 */

import { fetchWithAuth } from './httpClient';

export interface CreateShipmentInput {
  origin: string;
  destination: string;
  flight: string;
  weightKg: number;
  customerEmail: string;
}

export interface Shipment {
  code: string;
  origin: string;
  destination: string;
  flight: string;
  weightKg: number;
  customer: {
    email: string;
    name: string;
  };
  stage: string;
  createdAt: string;
}

/**
 * Create a new shipment.
 * 
 * @param input - Shipment data (origin, destination, flight, weightKg, customerEmail)
 * @returns Created shipment with code and stage
 * @throws Error with server message on failure
 */
export async function createShipment(input: CreateShipmentInput): Promise<Shipment> {
  const response = await fetchWithAuth('/api/shipments', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || 'Request failed');
  }

  return response.json();
}
