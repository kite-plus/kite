import { test as base, type Page, type Route } from "@playwright/test";

import { Site } from "./site";

export { expect } from "@playwright/test";

/** test gives every test a site and a server of its own, and points the browser at it. */
export const test = base.extend<{ site: Site; watch: boolean }>({
  watch: [true, { option: true }],
  site: async ({ watch }, use, testInfo) => {
    const site = await Site.create({ watch });
    try {
      await use(site);
    } finally {
      await site.stop();
      if (testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach("kite.log", { body: site.log, contentType: "text/plain" });
      }
      await site.remove();
    }
  },
  baseURL: async ({ site }, use) => {
    await use(site.url);
  },
});

/** bodyOf is the editor's text in the visual mode, which has no role or label of its own. */
export function bodyOf(page: Page) {
  return page.locator('[contenteditable="true"]');
}

/** appendTo types text at the end of a paragraph of the editor's text. */
export async function appendTo(page: Page, paragraph: string, text: string) {
  await bodyOf(page).getByText(paragraph).click();
  await page.keyboard.press("End");
  await page.keyboard.type(text);
}

/**
 * hold lets the next request of a method to url reach the server but keeps
 * its answer from the page until release is called, as a slow connection
 * would. reached resolves once the server has answered.
 */
export async function hold(page: Page, method: string, url: string) {
  let release = () => {};
  const released = new Promise<void>((resolve) => {
    release = resolve;
  });
  let answered = () => {};
  const reached = new Promise<void>((resolve) => {
    answered = resolve;
  });

  let held = false;
  await page.route(url, async (route: Route) => {
    if (held || route.request().method() !== method) {
      await route.fallback();
      return;
    }
    held = true;
    const response = await route.fetch();
    answered();
    await released;
    await route.fulfill({ response });
  });
  return { reached, release };
}
