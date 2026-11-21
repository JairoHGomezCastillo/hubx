import { test, expect } from '@playwright/test';

// Regression smoke: the gateway is up and proxies /api to hubx-api.
test.describe('health', () => {
  test('SMOKE_gateway_health', async ({ request }) => {
    const res = await request.get('/health');
    expect(res.status()).toBe(200);
    expect(await res.json()).toEqual({ status: 'ok', service: 'hubx-gw' });
  });

  test('SMOKE_api_health_through_gateway', async ({ request }) => {
    const res = await request.get('/api/health');
    expect(res.status()).toBe(200);
    expect(await res.json()).toEqual({ status: 'ok', service: 'hubx-api' });
  });
});
