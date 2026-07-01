// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import checker from "vite-plugin-checker";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    checker({
      typescript: { buildMode: true },
    }),
  ],
  envDir: "..",
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
    fs: {
      // Allow the linked @openfastchat/ui to read its ?raw assets
      // (e.g. compact/prompt.md) from the sibling core repo in dev.
      allow: ["../.."],
    },
  },
});
