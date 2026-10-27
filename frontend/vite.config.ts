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
  build: {
    // Embedded into the Go binary by backend/web.
    outDir: "../backend/web/dist",
    emptyOutDir: true,
  },
  resolve: {
    // @openfastchat/ui keeps its own node_modules (pnpm store, or the core
    // checkout when linked), so without this Vite bundles a second copy of
    // these singletons. Two copies
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
      // A linked @openfastchat/ui (core development) reads its ?raw assets
      // from the sibling core checkout.
      allow: ["../.."],
    },
  },
});
