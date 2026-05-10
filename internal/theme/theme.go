// Package theme gestisce il tema visivo dell'applicazione: 21 color keys
// + 2 gradient + metadati. Il tema di default è embedded nel binario.
// Il tema utente viene caricato via LoadTheme e merged sopra il default
// con strategia override-by-key (ADR-019 D3).
package theme

import "github.com/charmbracelet/lipgloss"

// Theme contiene le 18 color keys canoniche + 2 gradient + metadati.
// Tutti i campi sono sempre valorizzati (tipo total per costruzione —
// invariante THEME_TOTAL da theming.tla). Mai parziale post-bootstrap.
type Theme struct {
	// 18 color keys (ADR-019 D7)
	Primary          lipgloss.Color
	Incoming         lipgloss.Color
	Success          lipgloss.Color
	Warning          lipgloss.Color
	Error            lipgloss.Color
	Private          lipgloss.Color
	Text             lipgloss.Color
	TextDim          lipgloss.Color
	Surface          lipgloss.Color
	Border           lipgloss.Color
	SearchSecondary  lipgloss.Color
	SearchInlineBg   lipgloss.Color
	ButtonFg         lipgloss.Color
	ButtonBg         lipgloss.Color
	ButtonDisabledFg lipgloss.Color
	Reaction         lipgloss.Color
	ReactionChosen   lipgloss.Color
	SystemMessage    lipgloss.Color

	// Step 33: 4 nuovi color keys (ADR-021 §DD5).
	Link         lipgloss.Color // link underline + foreground
	Pinned       lipgloss.Color // icona 📌 + bordo pinned bar
	ForwardLabel lipgloss.Color // "From <X>" italic
	// SenderPalette: 8 colori deterministici per sender ID nei gruppi (DE2).
	// Invariante SENDER_COLOR_DETERMINISTIC: palette[abs(id)%8].
	SenderPalette [8]lipgloss.Color

	// 2 gradient keys (per RenderGradient)
	GradientStart string // hex string, convertito da caller con colorful.Hex
	GradientEnd   string

	// Metadati (free text, no validation)
	Name        string
	Author      string
	Version     string
	Description string
}

// KnownColorKeys elenca le 21 chiavi canoniche per validation e drift detection.
// 18 originali (ADR-019 D7) + 3 aggiunte Step 33 (ADR-021 §DD5):
// link, pinned, forward_label.
// Invariante: len(KnownColorKeys) == 21 (testato in TestKnownColorKeysCount).
var KnownColorKeys = []string{
	"primary", "incoming", "success", "warning", "error", "private",
	"text", "text_dim", "surface", "border", "search_secondary",
	"search_inline_bg", "button_fg", "button_bg", "button_disabled_fg",
	"reaction", "reaction_chosen", "system_message",
	"link", "pinned", "forward_label",
}
