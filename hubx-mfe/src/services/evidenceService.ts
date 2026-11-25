/**
 * Evidence service for HU-10.
 *
 * Calls GET /api/shipments/{code}/evidence through the gateway and saves the
 * returned PDF in the browser with a blob URL. The PDF is never parsed as JSON
 * and no file is created unless the response is successful.
 */

import * as authService from './authService';
import { fetchWithAuth } from './httpClient';

/** Explicit outcome of a traceability evidence download. */
export type EvidenceDownloadOutcome =
  | { kind: 'downloaded'; filename: string }
  | { kind: 'unauthorized' }
  | { kind: 'forbidden' }
  | { kind: 'not-found' }
  | { kind: 'httpError'; status: number; message: string }
  | { kind: 'noResponse' };

/** Flat JSON error body used by the HU-10 error family (never the PDF body). */
function isMessageBody(value: unknown): value is { message: string } {
  return (
    typeof value === 'object' &&
    value !== null &&
    'message' in value &&
    typeof value.message === 'string' &&
    value.message.length > 0
  );
}

async function errorMessage(response: Response, fallback: string): Promise<string> {
  const body: unknown = await response.json().catch(() => null);
  return isMessageBody(body) ? body.message : fallback;
}

/**
 * Save the PDF bytes with the contract's file name. The temporary anchor is
 * removed and its object URL revoked whether or not the click succeeds.
 */
function savePdf(blob: Blob, filename: string): void {
  const objectUrl = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement('a');
    anchor.href = objectUrl;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}

/**
 * Download the evidence document of a shipment.
 *
 * @param code - Shipment code (encoded into the path, never sent as a query).
 * @returns A discriminated outcome; an error never creates a download.
 */
export async function downloadEvidence(code: string): Promise<EvidenceDownloadOutcome> {
  const path = `/api/shipments/${encodeURIComponent(code)}/evidence`;
  const hadToken = authService.getToken() !== null;

  let response: Response;
  try {
    response = await fetchWithAuth(path, { headers: { Accept: 'application/pdf' } });
  } catch {
    // fetchWithAuth clears the in-memory token, stores the session-expired
    // notice and navigates to sign-in when it sees a 401. No response reaches
    // this point for that status, so the token transition identifies it.
    if (hadToken && authService.getToken() === null) {
      return { kind: 'unauthorized' };
    }
    return { kind: 'noResponse' };
  }

  if (!response.ok) {
    switch (response.status) {
      case 401:
        return { kind: 'unauthorized' };
      case 403:
        return { kind: 'forbidden' };
      case 404:
        return { kind: 'not-found' };
      default:
        return {
          kind: 'httpError',
          status: response.status,
          message: await errorMessage(response, 'The evidence could not be downloaded.'),
        };
    }
  }

  const filename = `evidence-${code}.pdf`;
  try {
    savePdf(await response.blob(), filename);
    return { kind: 'downloaded', filename };
  } catch {
    return { kind: 'noResponse' };
  }
}
