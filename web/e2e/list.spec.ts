import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures";

/** searchOf reads the listing's state back out of the address. */
function searchOf(page: Page): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [key, value] of new URL(page.url()).searchParams) {
    try {
      out[key] = JSON.parse(value);
    } catch {
      out[key] = value;
    }
  }
  return out;
}

/** choose picks an option of one of the listing's filters, and closes it again. */
async function choose(page: Page, filter: string, option: RegExp) {
  await page.getByRole("button", { name: filter }).click();
  await page.getByRole("option", { name: option }).click();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("option", { name: option })).toBeHidden();
}

test("filters, order and page size live in the address and survive a reload", async ({ page }) => {
  await page.goto("/admin/content/post");
  await expect(page.getByText("32 items")).toBeVisible();

  await choose(page, "Status", /^Published/);
  await choose(page, "Tags", /^garden/);
  await page.getByRole("button", { name: "Title", exact: true }).click();
  await page.getByRole("menuitem", { name: "Ascending" }).click();
  await page.getByRole("combobox").click();
  await page.getByRole("option", { name: "10" }).click();

  await expect(page.getByText("24 items")).toBeVisible();
  await expect(page.getByText("Page 1 of 3")).toBeVisible();
  expect(searchOf(page)).toEqual({ status: ["published"], terms: ["tags:garden"], sort: "title", size: 10 });
  await expect(page.getByRole("link", { name: "Garden note 01" })).toBeVisible();

  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Page 2 of 3")).toBeVisible();
  await expect(page.getByRole("link", { name: "Garden note 11" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Garden note 01" })).toBeHidden();

  // The cursor stays out of the address, so a reload starts the same listing over.
  await page.reload();
  await expect(page.getByText("Page 1 of 3")).toBeVisible();
  await expect(page.getByText("24 items")).toBeVisible();
  await expect(page.getByRole("link", { name: "Garden note 01" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Garden draft A" })).toBeHidden();
  await expect(page.getByRole("combobox")).toHaveText("10");
  await expect(page.getByRole("button", { name: "Status" })).toContainText("Published");
});

test("a batch delete reports the item changed on disk and trashes the rest", async ({ page, site }) => {
  const one = site.posts["old-news-one"];
  const two = site.posts["old-news-two"];
  await page.goto("/admin/content/post");
  await page.getByPlaceholder("Search title and body").fill("news");
  await page.getByRole("checkbox", { name: "Old news one" }).check();
  await page.getByRole("checkbox", { name: "Old news two" }).check();
  await page.getByRole("toolbar").getByRole("button", { name: "Delete" }).click();
  const confirm = page.getByRole("alertdialog", { name: "Delete 2 items?" });
  await expect(confirm).toBeVisible();

  await site.edit(two, (text) => text.replace("The day before.", "The day before, corrected."));
  await confirm.getByRole("button", { name: "Delete" }).click();

  const report = page.getByRole("alert").filter({ hasText: "Completed 1, failed 1" });
  await expect(report).toContainText("Old news two: This item changed since it was loaded.");
  await expect(page.getByRole("checkbox", { name: "Old news two" })).toBeChecked();

  await page.getByRole("link", { name: /^Trash/ }).click();
  await expect(page.getByRole("row", { name: /Old news one/ })).toBeVisible();
  await expect(page.getByRole("row", { name: /Old news two/ })).toBeHidden();
  expect(await site.read(one.file)).toMatch(/^deleted_at: /m);
  expect(await site.read(two.file)).not.toMatch(/^deleted_at: /m);
});
