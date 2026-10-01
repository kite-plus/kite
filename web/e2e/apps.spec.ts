import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { createServer, type Server } from "node:http";
import { mkdtemp, readFile, realpath, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";

import { expect, test } from "./fixtures";
import { deadIndex } from "./site";

const exec = promisify(execFile);

interface Release {
  version: string;
  published: string;
  notes: string;
  api: string;
  archive: { urls: string[]; sha256: string; size: number };
  loads: string[];
}

/**
 * FakeIndex serves an index of themes and plugins and their archives, the
 * way kite-plus/apps and jsDelivr serve them. Each archive is packed by the
 * kite binary under test, from a package kite started.
 */
class FakeIndex {
  private readonly archives = new Map<string, Buffer>();
  private readonly apps: { kind: string; id: string; versions: Release[] }[] = [];
  url = "";

  private constructor(
    private readonly server: Server,
    readonly dir: string,
  ) {}

  static async start(): Promise<FakeIndex> {
    const dir = await realpath(await mkdtemp(path.join(tmpdir(), "kite-e2e-index-")));
    const server = createServer();
    const index = new FakeIndex(server, dir);
    server.on("request", (request, response) => {
      if (request.url === "/index.json") {
        response.setHeader("Content-Type", "application/json");
        response.end(JSON.stringify(index.document()));
        return;
      }
      const archive = index.archives.get(request.url ?? "");
      response.statusCode = archive ? 200 : 404;
      response.end(archive);
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("the index has no port");
    index.url = `http://127.0.0.1:${address.port}`;
    return index;
  }

  async stop() {
    await new Promise((resolve) => this.server.close(resolve));
    await rm(this.dir, { recursive: true, force: true });
  }

  private document() {
    return {
      format: 1,
      generated: new Date().toISOString(),
      apps: this.apps.map((app) => ({
        kind: app.kind,
        id: app.id,
        official: false,
        repo: `someone/${app.id}`,
        title: { en: app.id[0].toUpperCase() + app.id.slice(1) },
        license: "MIT",
        versions: app.versions,
      })),
    };
  }

  /**
   * publish packs the package kite started in dir/id at a version, with its
   * manifest changed by edit, and lists it.
   */
  async publish(kind: "theme" | "plugin", id: string, version: string, edit: (manifest: string) => string = (m) => m) {
    const bin = process.env.KITE_BIN!;
    const folder = path.join(this.dir, id);
    const manifest = path.join(folder, kind === "theme" ? "theme.yaml" : "plugin.yaml");
    if (!this.apps.some((app) => app.id === id)) {
      await exec(bin, [kind, "new", id], { cwd: this.dir });
    }
    const text = (await readFile(manifest, "utf8")).replace(/^version: .*$/m, `version: ${version}`);
    await writeFile(manifest, edit(text));
    const out = path.join(this.dir, `${id}-${version}.zip`);
    await exec(bin, [kind, "pack", folder, "--output", out], { cwd: this.dir });
    const data = await readFile(out);
    const route = `/${kind}-${id}-${version}/${id}-${version}.zip`;
    this.archives.set(route, data);

    let app = this.apps.find((one) => one.id === id);
    if (!app) {
      app = { kind, id, versions: [] };
      this.apps.push(app);
    }
    app.versions.unshift({
      version,
      published: new Date().toISOString(),
      notes: "",
      api: kind === "theme" ? "kite/v1" : "kite/plugin/v1",
      archive: { urls: [this.url + route], sha256: createHash("sha256").update(data).digest("hex"), size: data.length },
      loads: [],
    });
  }
}

test.describe("the app center", () => {
  let index: FakeIndex;
  test.beforeEach(async ({ site }) => {
    index = await FakeIndex.start();
    await site.editConfig((text) => text.replace(deadIndex, `${index.url}/index.json`));
  });
  test.afterEach(async () => {
    await index.stop();
  });

  test("installs a theme from the index and updates it", async ({ page, site }) => {
    await index.publish("theme", "paper", "0.1.0");
    await page.goto("/admin/apps");

    const card = page.locator("article").filter({ hasText: "Paper" });
    await expect(card).toContainText("Community");
    await card.getByRole("button", { name: "Install" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("Install Paper 0.1.0?");
    await expect(dialog).toContainText("Loads nothing from other sites.");
    await dialog.getByRole("button", { name: "Install" }).click();
    await expect(page.getByText("Installed Paper")).toBeVisible();
    await expect(card).toContainText("0.1.0 installed");
    expect(await site.read("kite.lock")).toMatch(/paper:\n {4}version: 0\.1\.0/);

    await index.publish("theme", "paper", "0.2.0");
    await page.getByRole("button", { name: "Check again" }).click();
    await card.getByRole("button", { name: "Update to 0.2.0" }).click();
    await expect(dialog).toContainText("From 0.1.0 to 0.2.0.");
    await dialog.getByRole("button", { name: "Update to 0.2.0" }).click();
    await expect(page.getByText("Paper is at 0.2.0 now")).toBeVisible();
    await expect(card).toContainText("0.2.0 installed");
    expect(await site.read("themes/paper/theme.yaml")).toContain("version: 0.2.0");
    expect(await site.read("kite.lock")).toMatch(/paper:\n {4}version: 0\.2\.0/);
  });

  test("asks before a plugin's update loads from another site", async ({ page, site }) => {
    await index.publish("plugin", "greet", "0.1.0");
    await page.goto("/admin/apps?kind=plugin");
    const card = page.locator("article").filter({ hasText: "Greet" });
    await card.getByRole("button", { name: "Install" }).click();
    await page.getByRole("alertdialog").getByRole("button", { name: "Install" }).click();
    await expect(card).toContainText("0.1.0 installed");

    await index.publish("plugin", "greet", "0.2.0", (manifest) =>
      manifest.replace(
        "inject:\n",
        'inject:\n  - at: head\n    html: <script src="https://cdn.example.com/greet.js"></script>\n',
      ),
    );
    await page.getByRole("button", { name: "Check again" }).click();
    await card.getByRole("button", { name: "Update to 0.2.0" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("loads from cdn.example.com");
    const update = dialog.getByRole("button", { name: "Update to 0.2.0" });
    await expect(update).toBeDisabled();
    await dialog.getByRole("checkbox").click();
    await update.click();
    await expect(page.getByText("Greet is at 0.2.0 now")).toBeVisible();
    const lock = await site.read("kite.lock");
    expect(lock).toMatch(/greet:\n {4}version: 0\.2\.0/);
    expect(lock).toContain("- cdn.example.com");
  });
});
