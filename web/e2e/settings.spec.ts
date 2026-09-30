import { expect, test } from "./fixtures";

test("settings saved over a kite.yaml changed on disk are refused until reloaded", async ({ page, site }) => {
  await page.goto("/admin/settings");
  const title = page.getByRole("textbox", { name: "Title" });
  const save = page.getByRole("button", { name: "Save", exact: true });
  await expect(title).toHaveValue("E2E Site");
  await title.fill("Renamed in the studio");

  await site.editConfig((text) => text.replace("  title: E2E Site\n", "  title: E2E Site\n  description: Changed on disk\n"));
  await save.click();

  await expect(page.getByText("The configuration changed elsewhere. Reload before editing again.")).toBeVisible();
  await expect(save).toBeDisabled();
  expect(await site.read("kite.yaml")).not.toContain("Renamed in the studio");

  await page.getByRole("button", { name: "Reload settings" }).click();
  await expect(title).toHaveValue("E2E Site");
  await expect(page.getByRole("textbox", { name: "Description" })).toHaveValue("Changed on disk");

  await title.fill("Renamed after reloading");
  await save.click();
  await expect(page.getByText("Settings saved")).toBeVisible();
  const config = await site.read("kite.yaml");
  expect(config).toContain("title: Renamed after reloading");
  expect(config).toContain("description: Changed on disk");
});
