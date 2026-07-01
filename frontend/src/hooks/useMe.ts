// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import { useQuery } from "@tanstack/react-query";

/** The self-host identity, as returned by GET /api/me. */
export interface Me {
  is_guest: boolean;
  is_anonymous: boolean;
  default_flow_id: string | null;
  is_template?: boolean;
  email?: string;
}

export function useMe() {
  return useQuery<Me | null>({
    queryKey: ["me"],
    queryFn: async () => {
      const res = await fetch("/api/me");
      if (!res.ok) return null;
      return (await res.json()) as Me;
    },
    retry: false,
    staleTime: Infinity,
  });
}
