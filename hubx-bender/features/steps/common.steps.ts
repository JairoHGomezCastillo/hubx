import { Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import type { HubxWorld } from '../support/world.ts';

// Reusable steps. Story steps go in features/steps/HUxx_<name>.steps.ts.

When('I request {string} from the gateway', async function (this: HubxWorld, path: string) {
  this.response = await this.api.get(path);
});

Then('the response status is {int}', function (this: HubxWorld, status: number) {
  expect(this.response?.status()).toBe(status);
});

Then('the response body is {}', async function (this: HubxWorld, json: string) {
  expect(await this.response?.json()).toEqual(JSON.parse(json));
});

When('I open the web app', async function (this: HubxWorld) {
  await this.ui.goto('/');
});

Then('I see the heading {string}', async function (this: HubxWorld, text: string) {
  await expect(this.ui.getByRole('heading', { name: text })).toBeVisible();
});

// Runs fetch() inside the page: proves the browser -> MFE (nginx) -> gateway -> API path, independent of the UI.
Then(
  'the page can call {string} and gets {}',
  async function (this: HubxWorld, path: string, json: string) {
    const body = await this.ui.evaluate(async (p) => (await fetch(p)).json(), path);
    expect(body).toEqual(JSON.parse(json));
  },
);
