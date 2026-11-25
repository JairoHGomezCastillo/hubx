/**
 * Integrity service for verifying shipment chain integrity.
 * Calls GET /api/shipments/{code}/integrity through the gateway.
 */

import { fetchWithAuth } from './httpClient';

export interface VerificationResult {
  shipmentCode: string;
  intact: boolean;
  updatesChecked: number;
  firstAlteredUpdate: number | null;
}

/**
 * Verify the integrity of a shipment's chain.
 * 
 * @param code - Shipment code
 * @returns Verification result with intact status and updates checked
 * @throws Error with server message on failure
 */
export async function verifyIntegrity(code: string): Promise<VerificationResult> {
  const encodedCode = encodeURIComponent(code);
  const response = await fetchWithAuth(`/api/shipments/${encodedCode}/integrity`);

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || 'Request failed');
  }

  return response.json();
}
