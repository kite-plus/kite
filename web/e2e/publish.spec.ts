import { appendTo, expect, test } from "./fixtures";

test("a publish a git hook refuses says why, and goes out without the hooks", async ({ page, site }) => {
  const post = site.posts.ready;
  await site.hook("pre-commit", "#!/bin/sh\necho 'e2e: no commits today' >&2\nexit 1\n");
  const before = await site.remoteHead();

  await page.goto(`/admin/content/post/${post.id}`);
  await appendTo(page, "First words.", " Second words.");
  await page.getByRole("button", { name: "Update" }).click();

  const refusal = page.getByRole("dialog").filter({ hasText: "A git hook in this repository refused the commit." });
  await expect(refusal).toContainText("pre-commit hook refused: e2e: no commits today");
  expect(await site.git("status", "--porcelain")).toContain(post.file);

  await refusal.getByRole("button", { name: "Publish without the hooks" }).click();
  await expect(page.getByRole("region", { name: /^Notifications/ }).getByText("Published", { exact: true })).toBeVisible();

  const head = await site.git("rev-parse", "HEAD");
  expect(head).not.toBe(before);
  expect(await site.remoteHead()).toBe(head);
  expect(await site.git("log", "-1", "--format=%s")).toBe("publish: Ready to publish");
  expect(await site.git("show", `HEAD:${post.file}`)).toContain("First words. Second words.");
  expect(await site.git("status", "--porcelain")).toBe("");
});
