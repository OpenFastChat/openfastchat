// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import { ProvidersSection } from "@openfastchat/ui";

/**
 * Self-host admin page. Self-host is single-user, so the visitor is the
 * admin by definition — there is no token gate, the shared provider
 * panel is shown directly. All hosted-only admin surface (billing,
 * limits, abuse) stays in the hosted wrapper.
 */
export function AdminPage() {
  return (
    <div
      style={{
        maxWidth: "680px",
        margin: "0 auto",
        padding: "2rem 1.5rem",
        fontFamily: "var(--font-sans)",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: "1.5rem",
        }}
      >
        <h1
          style={{
            color: "var(--text)",
            fontSize: "1.25rem",
            fontWeight: 600,
            margin: 0,
          }}
        >
          Admin
        </h1>
        <a
          href="/"
          style={{
            color: "var(--accent-text)",
            fontSize: "0.85rem",
            textDecoration: "none",
          }}
        >
          Back to chat
        </a>
      </div>
      <ProvidersSection />
    </div>
  );
}
