import { defineConfig } from '@playwright/test';

// All tests go through the gateway, never straight to the API (constitution Principle V).
// BENDER_BASE_URL points at the gateway: local by default, the public URL for the deploy smoke test.
const baseURL = process.env.BENDER_BASE_URL ?? 'http://localhost:8080';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL,
    extraHTTPHeaders: { Accept: 'application/json' },
  },
});
