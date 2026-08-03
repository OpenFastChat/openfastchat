// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import { ChatShell, type ShellUser } from "@openfastchat/ui";
import { useMe } from "../hooks/useMe";
import { AdminMenu } from "../components/AdminMenu";

/**
 * Self-host chat page: the core ChatShell with none of the hosted
 * money/abuse slots. Bring-your-own-key means every prompt runs on the
 * operator's own provider credit, so there is no balance, billing, or
 * sign-in-to-keep-chatting surface to inject.
 */
export function ChatPage() {
  const { data } = useMe();
  const user: ShellUser = {
    is_guest: data?.is_guest ?? false,
    is_anonymous: data?.is_anonymous ?? false,
    default_flow_id: data?.default_flow_id ?? null,
    is_template: data?.is_template ?? false,
  };

  return (
    <ChatShell
      user={user}
      brand="OpenFastChat"
      statusEmail={data?.email ?? "self-host"}
      headerActions={data?.is_admin ? <AdminMenu /> : undefined}
    />
  );
}
