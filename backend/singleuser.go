// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"

	corehandler "github.com/1-Bart-1/openfastchat-core/handler"
	"github.com/1-Bart-1/openfastchat-core/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// selfHostUserEmail identifies the single self-host user. Self-host is
// single-user by design: every cookieless session resolves to this one row
// instead of minting a fresh guest, so two browsers share one account and one
// history — there is never a second user to reason about.
const selfHostUserEmail = "you@self-host.local"

// provisionSingleUser returns the one self-host user, creating it (with its
// default thread and flow) on first call. It is the GuestProvisioner handed to
// EnsureUser; unlike core's per-session ProvisionGuestUser it is idempotent, so
// repeated sign-ins converge on the same identity.
func provisionSingleUser(
	ctx context.Context, queries *store.Queries,
) (uuid.UUID, string, error) {
	user, err := queries.GetUserByEmail(ctx, selfHostUserEmail)
	if err == nil {
		return user.ID, user.Email, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", err
	}
	created, err := queries.CreateGuestUser(ctx, store.CreateGuestUserParams{
		Email:       selfHostUserEmail,
		DisplayName: "You",
	})
	if err != nil {
		if existing, gErr := queries.GetUserByEmail(ctx, selfHostUserEmail); gErr == nil {
			return existing.ID, existing.Email, nil
		}
		return uuid.Nil, "", err
	}
	corehandler.ProvisionUserDefaults(ctx, queries, created.ID)
	return created.ID, created.Email, nil
}
