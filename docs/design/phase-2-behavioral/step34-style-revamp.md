# Step 34 — Style Revamp (Crush-inspired)

**Stato**: design (pre-implementation)
**Tipo**: behavioral spec + mood board + token system
**ADR**: ADR-022 (style revamp), riprende ADR-019 (theming engine)

## 1. Obiettivo

Restyling totale di tuilegram per allinearlo all'estetica di **Charmland Crush** (riferimento: 4 screenshot in `~/Desktop/charmland-crush/`). L'app oggi è funzionalmente completa ma stilisticamente disomogenea: troppe saturazioni (rosa/azzurro/verde sulle chat list), bordi multi-color non coerenti, modali stile button-y, status bar plain, zero animazioni.

Crush definisce il target: **dark purple-magenta minimal**, bordi sottili uniformi, modali con titolo inline nel border, status bar middot-separated, animazioni discrete (banner slide-in).

## 2. Mood board — side by side

### tuilegram oggi

| Elemento | Stato |
|---|---|
| Chat list rows | Bordo full per riga, colori per status (rosa/azzurro/viola/verde/error) |
| Selezione chat | Bordo `error` rosso |
| Header | Tab bar grigia macOS-style |
| Modal palette/help/search | Bordo viola con header interno staccato |
| Status bar bottom | Testo plain, separatore `\|` ASCII, layout statico |
| Empty state | "Tuilegram / Select a chat" centrato |
| Sender colors gruppi | 8 colori saturi (rosa/azzurro/giallo/verde/...) |
| Notifiche | Solo testo statico nello status bar |
| Bordi modali | Stessa palette dei border componenti |

### Crush target

| Elemento | Stato |
|---|---|
| Sidebar | Bordo singolo viola scuro, righe senza border, selezione = highlight bg viola |
| Header | Nessun chrome — log brand `╳ Crush` in alto, contesto a destra (sidebar) |
| Modal palette | Bordo viola + titolo nel border-top: `╭─ Commands ─...System Λ User ─╮` |
| Status bar | Hint compatti, separatore middot `·`, foreground dim |
| Brand area | Logo grande magenta `Charm™ Crush` + meta info session |
| Sender colors | Monocromatica violet (toni di accent), no rainbow |
| Notifiche | Banner full-width slide-in dal basso (verde success, rosso error), autohide |
| Bordi | Tutti `border-default` (`#3A3450`), 1px style `lipgloss.NormalBorder()` |

## 3. Design tokens (token spec)

### 3.1 Palette

```toml
# crush.toml — nuovo default theme

[colors]
# Base surfaces
primary           = "#C661FE"  # magenta vivido — logo, header active, accent primary
incoming          = "#7B61FE"  # violet — incoming messages, secondary accent
success           = "#8FCB8F"  # muted green — banner okay
warning           = "#E0B872"  # warm amber — bot, reconnecting
error             = "#D86B6B"  # muted red — banner errors, failed status
private           = "#9580E0"  # soft purple — private chat marker
text              = "#D4D4E0"  # off-white — primary text
text_dim          = "#6B6982"  # cool grey-purple — timestamps, hints
surface           = "#221D2E"  # panel bg — slight elevation over base
border            = "#3A3450"  # dim purple-grey — ALL structural borders
search_secondary  = "#5B5582"  # muted violet — non-current search match
search_inline_bg  = "#2A2538"  # search bar fill
button_fg         = "#1A1625"  # very dark — high contrast on accent bg
button_bg         = "#C661FE"  # accent magenta — primary CTA
button_disabled_fg= "#6B6982"  # dim text
reaction          = "#7B61FE"  # violet
reaction_chosen   = "#C661FE"  # magenta
system_message    = "#6B6982"  # dim
link              = "#A88FFE"  # light violet underlined
pinned            = "#C661FE"  # magenta accent
forward_label     = "#7B61FE"  # violet

# Sender palette: monocromatica desaturata, 8 toni di violet
sender_palette = [
  "#C661FE", "#A88FFE", "#9580E0", "#7B61FE",
  "#8B7FB8", "#A899C8", "#7B6BA0", "#5B5582",
]

[gradient]
# Brand gradient per logo / banner
start = "#C661FE"
end   = "#7B61FE"

[meta]
name        = "crush"
author      = "tuilegram"
version     = "1.0"
description = "Crush-inspired dark purple-magenta theme (Step 34 default)"
```

### 3.2 Spacing scale

| Token | Char | Uso |
|---|---|---|
| `space-1` | 1 | padding interno minimo (chat row) |
| `space-2` | 2 | padding modali, sezioni |
| `space-3` | 4 | margin tra blocchi maggiori |

### 3.3 Border tokens

| Token | Style | Color |
|---|---|---|
| `border-default` | `lipgloss.NormalBorder()` | `border` |
| `border-focused` | `lipgloss.NormalBorder()` | `primary` (magenta) |
| `border-modal` | `lipgloss.NormalBorder()` con titolo inline top | `border` |

**Regola**: nessun bordo per chat-list rows. Selezione = bg fill, no border swap.

### 3.4 Typography

- Monospace unico (terminal native)
- `bold` riservato a: brand logo, header titoli sezione
- `underline` riservato a: link nei messaggi
- Nessun reverse video se non per cursor input

## 4. Animazioni — behavioral spec

Vedere `phase-3-interactions/step34-style-revamp-flow.md` per statechart Mermaid e statechart TLA+ delle animazioni concorrenti.

### 4.1 Notify banner

- **Trigger**: `NotifyMsg{Kind, Text}` da qualsiasi command
- **Mount**: slide-in dal bottom (3 frame: -2 row, -1 row, 0 row) a 60ms tick
- **Hold**: visibile per 3000ms (configurable)
- **Dismiss**: slide-out reverse oppure click/Esc
- **Concurrency**: nuova notify durante hold → replace immediato (no queue)

### 4.2 Modal mount

- **Open**: appare istantaneo (nessun fade — terminal limita ridraw senza flicker)
- **Border accent**: bordo `border-default`, titolo inline top con stile `primary`
- **Close**: rimozione istantanea

### 4.3 Focus transition

- **Border swap**: `border-default` ↔ `border-focused` (magenta) on focus change
- **Nessuna transition animata** — instant repaint

## 5. Componenti impattati

| File | Cambio | LOC stimato |
|---|---|---|
| `internal/theme/default.toml` | Sostituito da palette Crush | ~ |
| `internal/theme/crush.toml` (new embed alternativo) | Nuovo file embedded | ~80 |
| `internal/ui/views/chatlist.go` | Rimuovi bordi per-row, selezione = bg fill | ~ |
| `internal/ui/components/notify.go` | NEW — banner notify component | <120 |
| `internal/ui/components/modal.go` | Refactor titolo inline border-top | ~ |
| `internal/ui/views/main_view.go` | Status bar middot, brand empty state | ~ |
| `internal/ui/styles/borders.go` (new) | Token border-default/focused/modal | <120 |

## 6. Invarianti

- **CRUSH_THEME_TOTAL**: tutti i token Crush devono essere presenti in `crush.toml`. Validato a build (riusa `validateRawHex`).
- **NO_PERROW_BORDER**: `chatlist.go` non emette `lipgloss.Border()` per riga. Solo bg fill su selezione.
- **MODAL_TITLE_INLINE**: ogni modal usa primitive comune `RenderModal(title, body)` con titolo nel border-top.
- **NOTIFY_REPLACE**: nuova notify durante hold sostituisce la precedente immediatamente; nessuna queue.
- **STATUS_MIDDOT**: status bar usa `·` come separatore tra hint, mai `|`.

## 7. Out of scope

- Custom Unicode/figlet ASCII brand: usa `lipgloss.NewStyle()` + testo plain "tuilegram" stilizzato.
- Light theme: rimane post-step 34, non blocca.
- Animazioni complesse (typewriter, particles): out of scope.

## 8. Compatibilità

- Tema vecchio salvato come `legacy.toml`, selezionabile via config (`theme = "legacy"`).
- Default switch a `crush.toml` non breaking — utente può rollback.
