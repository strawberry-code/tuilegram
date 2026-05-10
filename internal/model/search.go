package model

import "time"

// SearchHit è un singolo risultato di una ricerca globale Telegram.
// ChatID e MessageID identificano univocamente il messaggio nel sistema.
// Snippet è il testo del messaggio, troncato a ~120 rune per la preview.
// SenderName può essere vuoto per messaggi in canali (sender anonimo).
type SearchHit struct {
	ChatID     ChatID
	ChatTitle  string
	MessageID  int
	SenderName string
	Snippet    string
	Date       time.Time
}

// SearchResult raccoglie gli hit di una singola invocazione SearchGlobal.
// QueryID è il token opaque propagato dal chiamante per gestire il
// drop di risultati stale (ADR-013: latestQueryID policy).
// Hits è nil (non empty slice) se la query era vuota o non ci sono risultati.
type SearchResult struct {
	QueryID uint64
	Hits    []SearchHit
}
