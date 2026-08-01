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
  resolve: {
    // @openfastchat/ui is a linked package with its own node_modules, so
    // without this Vite bundles a second copy of these singletons. Two copies
    // of react-query means two Contexts: the app's QueryClientProvider can't
    // reach the useQuery calls inside ChatShell ("No QueryClient set").
    dedupe: [
      "react",
      "react-dom",
      "react-router-dom",
      "@tanstack/react-query",
    ],
  },
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
