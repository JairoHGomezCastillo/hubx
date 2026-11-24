import { setWorldConstructor, World, type IWorldOptions } from '@cucumber/cucumber';
import type { APIRequestContext, APIResponse, BrowserContext, Page } from 'playwright';

// Gateway (API scenarios) and MFE (@e2e scenarios). Local defaults; the deploy smoke test points them at the public URLs.
export const GATEWAY_URL = process.env.BENDER_BASE_URL ?? 'http://localhost:8080';
export const UI_URL = process.env.BENDER_UI_URL ?? 'http://localhost:3000';

// State shared by the steps of one scenario. `api` always exists; `page` only in @e2e scenarios.
export class HubxWorld extends World {
  api!: APIRequestContext;
  context?: BrowserContext;
  page?: Page;
  response?: APIResponse;
  token?: string;
  
  // HU-01 scenario state
  role?: string;
  email?: string;

  constructor(options: IWorldOptions) {
    super(options);
  }

  // The current browser page; fails clearly when a browser step is used outside an @e2e scenario.
  get ui(): Page {
    if (!this.page) throw new Error('No browser page: tag the scenario with @e2e');
    return this.page;
  }
}

setWorldConstructor(HubxWorld);
