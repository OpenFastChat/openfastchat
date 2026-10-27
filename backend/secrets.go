// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/openfastchat/openfastchat-core/crypto"
)

// loadSecret returns the first non-empty env var of keys, else the secret
// persisted at dataDir/file, generating and saving one on first run.
func loadSecret(dataDir, file string, keys ...string) (string, error) {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v, nil
		}
	}
	path := filepath.Join(dataDir, file)
	stored, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(stored)), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	secret, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return "", err
	}
	slog.Info("generated secret", "env", keys[0], "path", path)
	return secret, nil
}

// defaultDataDir is where generated secrets live when OFC_DATA_DIR is unset.
func defaultDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "openfastchat")
	}
	return "data"
}
