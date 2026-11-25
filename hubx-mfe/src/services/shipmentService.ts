/**
 * Shipment service for creating shipments and listing assigned shipments.
 * All calls go through the gateway at /api.
 */

import { fetchWithAuth, getJson } from './httpClient';

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

export interface AssignedShipment {
  code: string;
  stage: string;
  origin: string;
  destination: string;
  flight: string;
  weightKg: number;
  assignedAt: string;
}

export interface AssignedShipmentsResponse {
  shipments: AssignedShipment[];
  nextOffset: number | null;
  message: string | null;
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

/**
 * List assigned shipments for the current ground partner.
 * 
 * @param offset - Pagination offset (default 0)
 * @returns List of assigned shipments with pagination info
 * @throws Error with server message on failure
 */
export async function listAssignedShipments(offset = 0): Promise<AssignedShipmentsResponse> {
  return getJson<AssignedShipmentsResponse>(`/api/assigned-shipments?offset=${offset}`);
}
