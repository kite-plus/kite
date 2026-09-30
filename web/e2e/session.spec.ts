import { randomBytes } from "node:crypto";

import { appendTo, bodyOf, expect, test } from "./fixtures";

test("a request refused for want of a session goes to sign in, and back with the work", async ({ page, site }) => {
  const post = site.posts.ready;
  await page.goto(`/admin/content/post/${post.id}`);
  await appendTo(page, "First words.", " Kept through signing in.");
  // The count follows the typing; once it is in, the save below is the next request.
  await expect(page.getByRole("contentinfo")).toContainText("6 words");

  // The studio was open; from now on the server wants a password this page
  // was never given.
  const user = `e2e-${randomBytes(4).toString("hex")}`;
  const password = randomBytes(18).toString("base64url");
  const set = await site.api("/account/credentials", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user, password }),
  });
  expect(set.status).toBe(200);

  // A key rather than the button: a background refresh may be refused first,
  // and the page is then already on its way to signing in.
  await page.keyboard.press("ControlOrMeta+s");
  await expect(page).toHaveURL(/\/admin\/sign-in\?/);
  expect(new URL(page.url()).searchParams.get("redirect")).toBe(`/content/post/${post.id}`);

  await page.getByLabel("User name").fill(user);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page).toHaveURL(new RegExp(`/admin/content/post/${post.id}$`));
  await expect(page.getByText(/^Brought back what you wrote/)).toBeVisible();
  await expect(bodyOf(page)).toHaveText("First words. Kept through signing in.");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByText(/^Saved at /)).toBeVisible();
  expect(await site.read(post.file)).toContain("First words. Kept through signing in.");
});
