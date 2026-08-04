// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

// Package nobilling implements core's handler.Billing seam for the self-host
// wrapper: every prompt is authorized, runs on the operator's own provider
// credit (bring-your-own-key), and is never metered, capped, or deducted.
package nobilling

import (
	"context"
	"net/http"

	"github.com/openfastchat/openfastchat-core/coreconfig"
	"github.com/openfastchat/openfastchat-core/flowengine"
	corehandler "github.com/openfastchat/openfastchat-core/handler"
	"github.com/openfastchat/openfastchat-core/llmclient"

	"github.com/google/uuid"
)

// Billing authorizes every prompt with no budget, cap, or metering.
type Billing struct{}

// New returns the no-op billing seam.
func New() Billing { return Billing{} }

// Preflight always allows.
func (Billing) Preflight(
	_ context.Context, _ http.ResponseWriter, _ uuid.UUID,
) (corehandler.Exec, bool) {
	return exec{}, true
}

type exec struct{}

func (exec) Begin(_, _ uuid.UUID) flowengine.LLMClient           { return unbilledLLM{} }
func (exec) SearchCharged() func(context.Context, string, int)   { return nil }
func (exec) PromptCap() float64                                  { return 0 }
func (exec) Guest() (used, limit int32, ok bool)                 { return 0, 0, false }
func (exec) Settle(context.Context) float64                      { return 0 }

// unbilledLLM runs completions on the active provider without recording usage.
type unbilledLLM struct{}

func (unbilledLLM) Complete(
	ctx context.Context, model string, messages []llmclient.Message, maxTokens int,
) (string, error) {
	text, _, err := coreconfig.ProviderFor(model).CompleteWithUsage(ctx, model, messages, maxTokens)
	return text, err
}

func (unbilledLLM) CompleteRaw(
	ctx context.Context, model string, messages []llmclient.Message,
	tools []llmclient.ToolDef, maxTokens int,
) (llmclient.Message, error) {
	msg, _, err := coreconfig.ProviderFor(model).CompleteRaw(ctx, model, messages, tools, maxTokens)
	return msg, err
}

func (unbilledLLM) Stream(
	ctx context.Context, model string, messages []llmclient.Message, maxTokens int,
) (<-chan string, <-chan error) {
	tokens, usageCh, errs := coreconfig.ProviderFor(model).Stream(ctx, model, messages, maxTokens)
	go func() {
		for range usageCh {
		}
	}()
	return tokens, errs
}

func (unbilledLLM) TotalBilled() float64 { return 0 }
