import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwind from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
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

const target = process.env.KITE_URL ?? "http://127.0.0.1:1717";

// The admin is served from inside the kite binary, so it is built to a
// directory the Go side embeds. In development it runs from Vite instead and
// proxies the API to a running `kite run`, which means the admin talks to the
// same endpoints in both cases and there is no second code path to keep true.
export default defineConfig({
  plugins: [
    // Generates src/routeTree.gen.ts from src/routes, and loads each screen
    // when it is first opened, so the editor's weight is paid only there.
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
    tailwind(),
    keepOutDirInGit(),
  ],
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
      "/api": { target, changeOrigin: true },
      // Everything that is not the admin itself -- the site's pages, an
      // item's images, the theme's assets -- is the server's too, so a
      // preview and a picture look the same here as they do in production.
      "^/(?!admin(/|$))": { target, changeOrigin: true },
    },
  },
});
