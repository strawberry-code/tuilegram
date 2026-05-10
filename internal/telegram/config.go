package telegram

import (
	"os"
	"strconv"
)

// Credenziali embedded — registrate su my.telegram.org per "tuilegram".
// Identificano l'applicazione, non l'utente. Non sono segreti.
// Override possibile via env var per sviluppatori.
const (
	defaultAppID   = 24503073                           // SECRETS-OK: app_id pubblico (my.telegram.org)
	defaultAppHash = "75fdc1ce2b2d79fbe7c50f600c65eb1c" // SECRETS-OK: app_hash pubblico (my.telegram.org)
)

// Config contiene le credenziali Telegram API.
type Config struct {
	AppID   int
	AppHash string
}

// LoadConfig carica le credenziali: env var hanno priorità, altrimenti embedded.
func LoadConfig() Config {
	appID := defaultAppID
	appHash := defaultAppHash

	if idStr := os.Getenv("TELEGRAM_APP_ID"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			appID = id
		}
	}

	if hash := os.Getenv("TELEGRAM_APP_HASH"); hash != "" {
		appHash = hash
	}

	return Config{AppID: appID, AppHash: appHash}
}

// IsValid restituisce true se le credenziali sono configurate.
func (c Config) IsValid() bool {
	return c.AppID > 0 && c.AppHash != ""
}
