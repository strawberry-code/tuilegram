package views

// convSubstate identifica lo stato precedente alla search in chat.
// Usato da SearchInChatState.ReturnTo per ripristinare il contesto su Esc.
type convSubstate int

const (
	convBrowsingMessages convSubstate = iota
	convMultiSelect
)

// IndexedMessage è un record nell'indice locale per la search in chat.
// textLC è pre-lowercased a build-time per evitare ToLower ad ogni keystroke.
// Pos è l'indice nella slice messages[] originale (usato per scroll viewport).
type IndexedMessage struct {
	MsgID  int
	TextLC string
	Pos    int
}

// TextSpan è un'occorrenza del match nel testo originale (byte offsets).
type TextSpan struct {
	Start int // byte offset inclusivo
	End   int // byte offset esclusivo
}

// SearchMatch rappresenta un messaggio che contiene almeno un'occorrenza della query.
type SearchMatch struct {
	MsgID int
	Spans []TextSpan
}

// SearchInChatState è lo stato completo della barra di ricerca inline.
// Tenuto dentro ConversationModel (per-chat, reset a ogni OpenChat).
// Invarianti formali verificati in search_in_chat.tla.
type SearchInChatState struct {
	Active     bool             // true se la barra è aperta
	Query      string           // valore corrente del textinput
	Index      []IndexedMessage // snapshot dei msg searchabili (non-service, non-empty)
	Matches    []SearchMatch    // hit della query corrente (ordine cronologico)
	CurrentIdx int              // indice in Matches del match evidenziato (current)
	ReturnTo   convSubstate     // stato a cui tornare su Esc
}

// currentMsgID restituisce il MsgID del match corrente, 0 se assente.
// Usato dal renderer per distinguere current match dagli altri.
func (s SearchInChatState) currentMsgID() int {
	if len(s.Matches) == 0 {
		return 0
	}
	return s.Matches[s.CurrentIdx].MsgID
}

// spansFor restituisce gli span del match per il messaggio msgID, nil se non match-a.
func (s SearchInChatState) spansFor(msgID int) []TextSpan {
	for _, m := range s.Matches {
		if m.MsgID == msgID {
			return m.Spans
		}
	}
	return nil
}
