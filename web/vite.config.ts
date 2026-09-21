import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwind from "@tailwindcss/vite";
import fs from "node:fs";
import path from "node:path";

/**
 * The Go build embeds dist, and go:embed fails to compile when the directory
 * does not exist. The directory is therefore kept in git through a placeholder
 * -- which emptying the output directory removes, so it is put back here
 * rather than in a make target, so that running vite alone is enough.
 */
function keepOutDirInGit(): Plugin {
  return {
    name: "kite:keep-outdir-in-git",
    closeBundle() {
      fs.writeFileSync(path.resolve(import.meta.dirname, "dist/.gitkeep"), "");
    },
  };
}

// The admin is served from inside the kite binary, so it is built to a
// directory the Go side embeds. In development it runs from Vite instead and
// proxies the API to a running `kite run`, which means the admin talks to the
// same endpoints in both cases and there is no second code path to keep true.
export default defineConfig({
  plugins: [react(), tailwind(), keepOutDirInGit()],
  base: "/admin/",
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.KITE_URL ?? "http://127.0.0.1:1717",
        changeOrigin: true,
      },
    },
  },
});
