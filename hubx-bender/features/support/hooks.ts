import { After, AfterAll, Before, setDefaultTimeout } from '@cucumber/cucumber';
import { chromium, request, type Browser } from 'playwright';
import { GATEWAY_URL, UI_URL, type HubxWorld } from './world.ts';

setDefaultTimeout(30_000);

// One browser for the whole run, launched lazily by the first @e2e scenario.
let browser: Browser | undefined;

Before(async function (this: HubxWorld) {
  this.api = await request.newContext({
    baseURL: GATEWAY_URL,
    extraHTTPHeaders: { Accept: 'application/json' },
  });
});

Before({ tags: '@e2e' }, async function (this: HubxWorld) {
  browser ??= await chromium.launch({ headless: process.env.HEADED !== '1' });
  // HU-10's @e2e scenario captures the evidence PDF as a browser download.
  this.context = await browser.newContext({ baseURL: UI_URL, acceptDownloads: true });
  this.page = await this.context.newPage();
});

After(async function (this: HubxWorld, { result }) {
  if (this.page && result?.status === 'FAILED') {
    this.attach(await this.page.screenshot({ fullPage: true }), 'image/png');
  }
  await this.context?.close();
  await this.api?.dispose();
});

AfterAll(async function () {
  await browser?.close();
});
