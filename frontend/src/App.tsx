// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { showNotice, errorStatus, errorMessage } from "@openfastchat/ui";
import { ChatPage } from "./pages/ChatPage";

// Surface only genuine server faults; client-side 4xx are handled inline
// by the components that make the request.
function handleServerError(error: unknown) {
  const status = errorStatus(error);
  if (!status || status < 500) return;
  showNotice({
    kind: "server_error",
    message: errorMessage(error, "Something went wrong on the server."),
    statusCode: status,
    canReport: false,
  });
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: handleServerError }),
  mutationCache: new MutationCache({ onError: handleServerError }),
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ChatPage />} />
          <Route path="/flow/:id" element={<ChatPage />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
