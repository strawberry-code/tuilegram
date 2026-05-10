package styles

import "github.com/charmbracelet/lipgloss"

// Accessor functions per i colori del tema attivo.
// Ogni funzione legge dal puntatore atomico Active() — un singolo load.
// Perf: trascurabile (pointer dereference + field access), chiamate da View().
// Nessun literal hex sparso in internal/ui/ (ADR-019 §Conseguenze, invariante 12).

// ColorPrimary è l'accent generico: focused border, gruppi, palette.
func ColorPrimary() lipgloss.Color { return Active().Primary }

// ColorIncoming è il colore per messaggi incoming, canali, dot unread.
func ColorIncoming() lipgloss.Color { return Active().Incoming }

// ColorSuccess è il colore per dot online, connected status.
func ColorSuccess() lipgloss.Color { return Active().Success }

// ColorWarning è il colore per bot border, status reconnecting.
func ColorWarning() lipgloss.Color { return Active().Warning }

// ColorError è il colore per active chat border, errori, no-match search.
func ColorError() lipgloss.Color { return Active().Error }

// ColorPrivate è il colore per chat private.
func ColorPrivate() lipgloss.Color { return Active().Private }

// ColorText è il foreground di default per testo principale.
func ColorText() lipgloss.Color { return Active().Text }

// ColorTextDim è il foreground dim per timestamp, hint, label.
func ColorTextDim() lipgloss.Color { return Active().TextDim }

// ColorSurface è il background scuro per palette/search/folder selected.
func ColorSurface() lipgloss.Color { return Active().Surface }

// ColorBorder è il colore per bordi strutturali (chatlist, conversation).
func ColorBorder() lipgloss.Color { return Active().Border }

// ColorSearchSecondary è l'highlight per search match non-current.
func ColorSearchSecondary() lipgloss.Color { return Active().SearchSecondary }

// ColorSearchInlineBg è il background della search bar inline.
func ColorSearchInlineBg() lipgloss.Color { return Active().SearchInlineBg }

// ColorButtonFg è il foreground del button attivo.
func ColorButtonFg() lipgloss.Color { return Active().ButtonFg }

// ColorButtonBg è il background del button attivo.
func ColorButtonBg() lipgloss.Color { return Active().ButtonBg }

// ColorButtonDisabledFg è il foreground del button disabled.
func ColorButtonDisabledFg() lipgloss.Color { return Active().ButtonDisabledFg }

// ColorReaction è il colore di default per le reaction.
func ColorReaction() lipgloss.Color { return Active().Reaction }

// ColorReactionChosen è il colore per reaction scelte dall'utente.
func ColorReactionChosen() lipgloss.Color { return Active().ReactionChosen }

// ColorSystemMessage è il colore per system message (join/leave/pin).
func ColorSystemMessage() lipgloss.Color { return Active().SystemMessage }

// ColorLink è il colore per link (underline + foreground) nei messaggi.
func ColorLink() lipgloss.Color { return Active().Link }

// ColorPinned è il colore per l'icona 📌 e il bordo della pinned bar.
func ColorPinned() lipgloss.Color { return Active().Pinned }

// ColorForwardLabel è il colore per la riga "From <X>" dei messaggi inoltrati.
func ColorForwardLabel() lipgloss.Color { return Active().ForwardLabel }

// ColorSenderPalette ritorna la palette di 8 colori per sender name nei gruppi.
// Invariante SENDER_COLOR_DETERMINISTIC: palette[abs(id)%8].
func ColorSenderPalette() [8]lipgloss.Color { return Active().SenderPalette }
