import { readFile } from "node:fs/promises";
import path from "node:path";

import type { Page } from "@playwright/test";

import { appendTo, bodyOf, expect, hold, test } from "./fixtures";

const savedAt = /^Saved at /;

/** openedId is the id a new post moved to once its first save gave it one. */
async function openedId(page: Page): Promise<string> {
  await expect(page).toHaveURL(/\/admin\/content\/post\/[0-9A-Z]{26}$/);
  return new URL(page.url()).pathname.split("/").at(-1) ?? "";
}

test("a new post is saved to a file of its own", async ({ page, site }) => {
  await page.goto("/admin/content/post/new");
  await page.getByRole("textbox", { name: "Title" }).fill("A walk in the hills");
  await bodyOf(page).click();
  await page.keyboard.type("The path climbed through the pines.");
  await page.keyboard.press("ControlOrMeta+s");

  const id = await openedId(page);
  await expect(page.getByText(savedAt)).toBeVisible();
  const text = await site.textOf(id);
  expect(text).toContain("title: A walk in the hills");
  expect(text).toContain("The path climbed through the pines.");
});

test("text typed while a save is in flight stays unsaved until the next save", async ({ page, site }) => {
  const post = site.posts["slow-save"];
  const save = await hold(page, "PUT", `**/api/v1/contents/${post.id}`);
  await page.goto(`/admin/content/post/${post.id}`);
  await appendTo(page, "Written before the save.", " Sent with the first save.");
  await page.keyboard.press("ControlOrMeta+s");
  await save.reached;
  await page.keyboard.type(" Typed while it was saving.");
  await expect(page.getByText("Saving", { exact: true })).toBeVisible();
  save.release();

  const whole = "Written before the save. Sent with the first save. Typed while it was saving.";
  await expect(page.getByText("Unsaved changes, kept in this browser")).toBeVisible();
  await expect(bodyOf(page)).toHaveText(whole);
  const first = await site.read(post.file);
  expect(first).toContain("Written before the save. Sent with the first save.");
  expect(first).not.toContain("Typed while it was saving.");

  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByText(savedAt)).toBeVisible();
  expect(await site.read(post.file)).toContain(whole);

  await page.reload();
  await expect(bodyOf(page)).toHaveText(whole);
  await expect(page.getByText("Saved", { exact: true })).toBeVisible();
  await expect(page.getByText(/^Brought back what you wrote/)).toBeHidden();
});

test("a new post keeps what was typed while its first save was in flight", async ({ page, site }) => {
  // Whichever save comes first, the key's or the draft's own, is the one held.
  const create = await hold(page, "POST", "**/api/v1/contents");
  await page.goto("/admin/content/post/new");
  await page.getByRole("textbox", { name: "Title" }).fill("Written in one go");
  await bodyOf(page).click();
  await page.keyboard.type("First line.");
  await page.keyboard.press("ControlOrMeta+s");
  await create.reached;
  await page.keyboard.type(" Second line.");
  create.release();

  const id = await openedId(page);
  await expect(bodyOf(page)).toHaveText("First line. Second line.");
  // A draft saves the rest by itself once the typing stops.
  await expect.poll(() => site.textOf(id)).toContain("First line. Second line.");
  await expect(page.getByText(savedAt)).toBeVisible();

  await page.reload();
  await expect(bodyOf(page)).toHaveText("First line. Second line.");
});

test("a post changed on disk meanwhile merges with the edit made here", async ({ page, site }) => {
  const post = site.posts["three-paragraphs"];
  await page.goto(`/admin/content/post/${post.id}`);
  await expect(bodyOf(page)).toContainText("Charlie paragraph.");

  await site.edit(post, (text) => text.replace("Alpha paragraph.", "Alpha paragraph, changed on disk."));
  await appendTo(page, "Charlie paragraph.", " Changed in the studio.");
  await page.getByRole("button", { name: "Save", exact: true }).click();

  const conflict = page.getByRole("dialog", { name: "This changed while you were editing" });
  await expect(conflict.getByText("Different paragraphs")).toBeVisible();
  await conflict.getByRole("button", { name: "Merge both" }).click();
  await expect(conflict).toBeHidden();

  await expect(bodyOf(page)).toContainText("Alpha paragraph, changed on disk.");
  await expect(bodyOf(page)).toContainText("Charlie paragraph. Changed in the studio.");
  await expect(page.getByText("Unsaved changes, kept in this browser")).toBeVisible();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByText(savedAt)).toBeVisible();

  expect(await site.read(post.file)).toContain(
    "Alpha paragraph, changed on disk.\n\nBravo paragraph.\n\nCharlie paragraph. Changed in the studio.\n",
  );
});

test.describe("with the index behind the disk", () => {
  // Without a watcher the index lags the disk until the next write through the
  // API, as it does for a moment after every change with one.
  test.use({ watch: false });

  test("a conflict compares against the file on disk", async ({ page, site }) => {
    const post = site.posts["three-paragraphs"];
    await page.goto(`/admin/content/post/${post.id}`);
    await appendTo(page, "Charlie paragraph.", " Changed in the studio.");
    await site.write(post.file, (await site.read(post.file)).replace("Alpha paragraph.", "Alpha paragraph, changed on disk."));
    await page.getByRole("button", { name: "Save", exact: true }).click();

    const conflict = page.getByRole("dialog", { name: "This changed while you were editing" });
    await expect(conflict.getByText("Different paragraphs")).toBeVisible();
    await conflict.getByRole("button", { name: "Merge both" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect(page.getByText(savedAt)).toBeVisible();
    expect(await site.read(post.file)).toContain("Alpha paragraph, changed on disk.");
  });
});

// The smallest PNG there is: one grey pixel.
const png = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNoAAAAggCBd81ytgAAAABJRU5ErkJggg==",
  "base64",
);

test("a file uploaded to a post is kept beside it and published with it", async ({ page, site }) => {
  const post = site.posts.coast;
  await page.goto(`/admin/content/post/${post.id}`);
  await page.getByRole("button", { name: "Files" }).click();
  const files = page.getByRole("dialog", { name: "Files" });
  await expect(files.getByText("No files yet.")).toBeVisible();

  const chooser = page.waitForEvent("filechooser");
  await files.getByRole("button", { name: "Upload" }).click();
  await (await chooser).setFiles({ name: "harbour.png", mimeType: "image/png", buffer: png });

  await expect(files.getByText("harbour.png")).toBeVisible();
  await expect(files.getByText("1 file", { exact: true })).toBeVisible();
  const stored = await readFile(path.join(site.root, path.dirname(post.file), "harbour.png"));
  expect(stored.equals(png)).toBe(true);
  expect((await page.request.get("/posts/coast/harbour.png")).status()).toBe(200);
});
