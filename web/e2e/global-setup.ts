import { execFile } from "node:child_process";
import { access, constants, mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";

const exec = promisify(execFile);
const web = path.resolve(import.meta.dirname, "..");

/**
 * The studio is tested as it ships, inside the binary that serves it. KITE_BIN
 * names a binary built from this checkout, as `make e2e` and CI do; without
 * one the studio and a binary embedding it are built here, so that what runs
 * is never an older bundle than the source.
 */
export default async function globalSetup() {
  if (process.env.KITE_BIN) {
    // Absolute, since each server is started from its own site's directory.
    process.env.KITE_BIN = path.resolve(process.env.KITE_BIN);
    await access(process.env.KITE_BIN, constants.X_OK);
    return;
  }

  console.log("Building the studio and kite for the browser tests; set KITE_BIN to use a binary already built.");
  const out = await mkdtemp(path.join(tmpdir(), "kite-e2e-bin-"));
  const removeOut = () => rm(out, { recursive: true, force: true });
  const bin = path.join(out, process.platform === "win32" ? "kite.exe" : "kite");
  try {
    await exec("pnpm", ["exec", "vite", "build"], { cwd: web, shell: process.platform === "win32" });
    await exec("go", ["build", "-o", bin, "./cmd/kite"], {
      cwd: path.dirname(web),
      env: { ...process.env, CGO_ENABLED: "0" },
    });
  } catch (err) {
    await removeOut();
    throw err;
  }

  // Workers inherit what the global setup puts in the environment.
  process.env.KITE_BIN = bin;
  return removeOut;
}
