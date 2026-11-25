/**
 * Unit tests for evidenceService: request shape, blob download and the
 * explicit 401/403/404/no-response outcomes. jsdom has no object-URL API, so
 * the two static methods are stubbed and restored after each test.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import * as authService from './authService';
import { downloadEvidence } from './evidenceService';

const CODE = 'HBX-TEST-0001';
const OBJECT_URL = 'blob:hubx-evidence';
const PDF_BYTES = new Uint8Array([0x25, 0x50, 0x44, 0x46, 0x2d]); // %PDF-

const createDescriptor = Object.getOwnPropertyDescriptor(URL, 'createObjectURL');
const revokeDescriptor = Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL');
const clickDescriptor = Object.getOwnPropertyDescriptor(HTMLAnchorElement.prototype, 'click');

let createObjectURL: ReturnType<typeof vi.fn>;
let revokeObjectURL: ReturnType<typeof vi.fn>;
let fetchMock: ReturnType<typeof vi.fn>;
let clickMock: ReturnType<typeof vi.fn>;

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

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function pdfResponse(): Response {
  return new Response(PDF_BYTES, {
    status: 200,
    headers: { 'Content-Type': 'application/pdf' },
  });
}

function firstFetchCall(): [string, { headers: Headers }] {
  return fetchMock.mock.calls[0] as [string, { headers: Headers }];
}

describe('downloadEvidence', () => {
  beforeEach(() => {
    authService.setToken(signedInToken());
    clickMock = vi.fn();

    createObjectURL = vi.fn(() => OBJECT_URL);
    revokeObjectURL = vi.fn();
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: createObjectURL });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL });
    Object.defineProperty(HTMLAnchorElement.prototype, 'click', { configurable: true, value: clickMock });

    fetchMock = vi.fn().mockResolvedValue(pdfResponse());
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    if (createDescriptor) Object.defineProperty(URL, 'createObjectURL', createDescriptor);
    else Reflect.deleteProperty(URL, 'createObjectURL');
    if (revokeDescriptor) Object.defineProperty(URL, 'revokeObjectURL', revokeDescriptor);
    else Reflect.deleteProperty(URL, 'revokeObjectURL');
    if (clickDescriptor) Object.defineProperty(HTMLAnchorElement.prototype, 'click', clickDescriptor);
    else Reflect.deleteProperty(HTMLAnchorElement.prototype, 'click');
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    authService.clearToken();
    sessionStorage.clear();
  });

  it('downloads the PDF through the gateway with the contract headers and filename', async () => {
    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({ kind: 'downloaded', filename: `evidence-${CODE}.pdf` });
    expect(fetchMock).toHaveBeenCalledWith(
      `/api/shipments/${CODE}/evidence`,
      expect.objectContaining({ headers: expect.any(Headers) }),
    );
    const [url, init] = firstFetchCall();
    expect(url).toBe(`/api/shipments/${CODE}/evidence`);
    expect(init.headers.get('Accept')).toBe('application/pdf');
    expect(init.headers.get('Authorization')).toMatch(/^Bearer /);

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    const blob = createObjectURL.mock.calls[0]?.[0] as Blob;
    expect(blob.type).toBe('application/pdf');
    expect(clickMock).toHaveBeenCalledTimes(1);
    const anchor = clickMock.mock.contexts[0] as HTMLAnchorElement | undefined;
    expect(anchor?.getAttribute('href')).toBe(OBJECT_URL);
    expect(anchor?.getAttribute('download')).toBe(`evidence-${CODE}.pdf`);
    expect(anchor?.parentElement).toBeNull();
    expect(revokeObjectURL).toHaveBeenCalledWith(OBJECT_URL);
  });

  it('encodes the shipment code in the request path', async () => {
    await downloadEvidence('HBX-TEST/SPECIAL');
    expect(firstFetchCall()[0]).toBe('/api/shipments/HBX-TEST%2FSPECIAL/evidence');
  });

  it('maps 401 to unauthorized without saving a file', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'UNAUTHORIZED', message: 'Unauthorized' }, 401));

    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({ kind: 'unauthorized' });
    expect(authService.getToken()).toBeNull();
    expect(clickMock).not.toHaveBeenCalled();
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it('maps 403 to forbidden without saving a file', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'FORBIDDEN', message: 'You do not have permission to download evidence' }, 403),
    );

    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({ kind: 'forbidden' });
    expect(clickMock).not.toHaveBeenCalled();
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it('maps 404 to not-found without saving a file', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'SHIPMENT_NOT_FOUND', message: 'No shipment exists with that code' }, 404),
    );

    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({ kind: 'not-found' });
    expect(clickMock).not.toHaveBeenCalled();
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it('maps other HTTP errors to httpError with the server message', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'INTERNAL_SERVER_ERROR', message: 'An unexpected error occurred' }, 500),
    );

    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({
      kind: 'httpError',
      status: 500,
      message: 'An unexpected error occurred',
    });
    expect(clickMock).not.toHaveBeenCalled();
  });

  it('maps a network failure to noResponse', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));

    const outcome = await downloadEvidence(CODE);

    expect(outcome).toEqual({ kind: 'noResponse' });
    expect(clickMock).not.toHaveBeenCalled();
    expect(createObjectURL).not.toHaveBeenCalled();
  });
});
