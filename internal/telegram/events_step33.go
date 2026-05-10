package telegram

import "github.com/strawberry-code/tuilegram/internal/model"

// PinnedMsgLoadedMsg è il risultato del loadPinnedMessageCmd (Step 33, ADR-021 §A1).
// Invariante PINNED_STALE_DROP: il consumer dropa il msg se ChatID != activeChatID.
type PinnedMsgLoadedMsg struct {
	ChatID model.ChatID
	Msg    *model.Message // nil se nessun pin o errore
	Err    error
}

// OpenLinkMsg chiede al TUI di aprire un URL nel browser di sistema (Step 33, ADR-021 §DB3).
// Invariante LINK_OPEN_HTTP_ONLY: URL deve iniziare con http:// o https://.
// Emesso da conversation quando l'utente preme il chord `gx` su un msg con link.
type OpenLinkMsg struct {
	URL string
}
