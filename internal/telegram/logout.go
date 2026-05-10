package telegram

import (
	"context"
	"os"
)

// logout.go — Step 34: account sign-out via Auth.LogOut + session file removal.
// Session file path mirrors client.go: "session.json" relativo al cwd.
const sessionPath = "session.json"

// Logout invoca AuthLogOut RPC server-side e rimuove il session file locale.
// Errori RPC non bloccano la rimozione del file (best-effort).
func (b *Bridge) Logout(ctx context.Context) error {
	if b.api != nil {
		_, _ = b.api.AuthLogOut(ctx)
	}
	return os.Remove(sessionPath)
}
