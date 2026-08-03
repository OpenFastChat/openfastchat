// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import { pill } from "@openfastchat/ui";

/**
 * Header entry to the provider-settings page. Self-host is single-user
 * and always admin, so this is a plain link with no auth to drop.
 */
export function AdminMenu() {
  return (
    <a href="/admin" style={{ ...pill, textDecoration: "none" }}>
      Admin
    </a>
  );
}
