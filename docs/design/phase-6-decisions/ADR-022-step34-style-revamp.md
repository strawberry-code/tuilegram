# ADR-022: Step 34 — Style revamp Crush-inspired (palette, borders, modals, status bar, animations)

**Stato**: accettato
**Data**: 2026-05-10
**Riferimenti**: ADR-019 (theming engine), ADR-004 (theming system originale), `phase-2-behavioral/step34-style-revamp.md`, `phase-3-interactions/step34-style-revamp-flow.md`

## Contesto

Post-Step 33, tuilegram è funzionalmente completo (33/33 step DONE) ma stilisticamente disomogeneo:

- Chat list usa bordi colorati per ogni riga (rosa, azzurro, viola, error rosso) → effetto rumoroso, satura
- Selezione chat = swap bordo a `error` → dissonante con resto della UI
- Modali (palette, search, help, keybindings) hanno bordo viola con titolo statico interno → stile "dialog button-y"
- Status bar bottom è plain text con separator `|` ASCII → dated
- Empty state minimal "Tuilegram / Select a chat" → poco informativo
- Sender colors palette satura (rosa/azzurro/giallo) → clash con resto
- Zero animazioni: notifiche statiche nella status bar, no feedback visuale per azioni

L'utente fornisce due cartelle riferimento (`~/Desktop/tuilegram-oggi/` e `~/Desktop/charmland-crush/`) e chiede revamp profondo per replicare estetica + animazioni di **Charmland Crush** (Charm.land terminal AI client). Crush definisce uno standard estetico nel TUI ecosystem charm: dark purple-magenta minimal, bordi sottili uniformi, modali con titolo inline nel border-top, status bar middot-separated, banner notify slide-in.

L'engine di theming (ADR-019) supporta già hot-reload via TOML embedded + atomic swap. Il revamp **non richiede** nuovo engine — richiede:

1. Nuovo theme `crush.toml` (default) + preservazione `legacy.toml` (rollback)
2. Refactor componenti che bypassano i token theme (chat-list bordi per riga, modal titolo statico)
3. Nuovo componente `notify` per banner animato
4. Primitive `RenderModal(title, body)` riusabile (rispetta memory `feedback_modal_charm`)
5. Status bar middot rendering

## Decisione

Apriamo **Step 34** in pipeline come step di revamp stilistico, atomic, design-first. Step 34 NON aggiunge feature funzionali — refactora il layer presentazione per allinearsi a Crush.

### Scope dichiarato

**Inside scope**:

- `internal/theme/crush.toml` (nuovo default embedded)
- `internal/theme/legacy.toml` (vecchio default, ora alternativo)
- `internal/theme/default.go` switch per scegliere quale embeddare come default (build-time const + runtime override via config `theme = "..."`)
- `internal/ui/views/chatlist.go`: rimozione bordi per riga, selezione = bg fill via `Active().Incoming` darkened
- `internal/ui/components/modal.go` (nuovo o refactor): primitive `RenderModal(title, body, hints)` con titolo nel border-top
- `internal/ui/components/notify.go` (nuovo): banner notify slide-in/out con stato animato
- `internal/ui/views/main_view.go`: status bar middot, render banner sopra status bar, brand empty state
- `internal/ui/styles/borders.go` (nuovo, ≤120 LOC): token border-default/focused/modal helpers

**Outside scope**:

- Light theme (rimane post-step 34 separato)
- Custom Unicode/figlet ASCII brand (testo plain stilizzato lipgloss)
- Mouse-driven animation (banner click-to-dismiss OK, drag/swipe NO)
- Audio feedback
- Notification sounds / system bell
- Modifiche al routing message lifecycle / telegram client / domain types
- Multiple banner stacked (NOTIFY_NO_QUEUE invariante)

### Invarianti dichiarate

- `CRUSH_THEME_TOTAL`: tutti i token Crush devono essere presenti in `crush.toml`. Validato a build via `validateRawHex`. Build-time panic se manca o hex invalido.
- `NO_PERROW_BORDER`: `chatlist.go` non emette `lipgloss.Border()` per riga. Solo bg fill su selezione. Enforced via code review (no helper `RenderChatRowBorder` deve esistere).
- `MODAL_TITLE_INLINE`: ogni modal usa `RenderModal(title, body, hints)`. Modali esistenti (palette, search, help, keybindings) refactorate per chiamare la primitive. No render bordo + header staccato.
- `NOTIFY_REPLACE`: nuova `NotifyMsg` durante `Visible` o `Mounting` sostituisce immediatamente. `len(pending) ≤ 1` sempre.
- `NOTIFY_FRAME_BOUNDED`: frame counter è 0..2.
- `STATUS_MIDDOT`: status bar usa `·` come separatore. Mai `|`.
- `MODAL_SINGLE`: max 1 modal aperto.
- `FOCUS_SINGLE`: esattamente un panel focused.

### Rollout

1. Design-first (questo step). Stop review prima di codice. **STATO ATTUALE**.
2. Implementazione incrementale dietro flag `theme = "crush"` (default `legacy` in dev branch durante implementazione, switch a `crush` come ultimo commit di Step 34).
3. Test end-to-end: tutti gli step 1-33 continuano a funzionare con nuovo theme.
4. Closing: aggiornamento status pipeline (TODO→DONE), commit messaggio convenzionale.

## Conseguenze

### Positive

- UI coerente, allineata a standard estetico charm ecosystem (Crush, Bubbles examples).
- Theming engine già in place (ADR-019) → revamp = swap palette + refactor isolati, no nuova architettura.
- Primitive `RenderModal` elimina duplicazione tra palette/search/help/keybindings (riduce LOC totali).
- Animazioni notify migliorano feedback UX senza intaccare modello bubbletea (rientrano nel message loop).
- Rollback semplice: `theme = "legacy"` in config riporta look pre-step 34.

### Negative

- Refactor di 4-5 componenti UI esistenti → rischio regressione. Mitigato da invarianti + test manuale step 1-33 nel test plan.
- Notify component aggiunge timer + frame counter → nuova source di state nel root model. Mitigato da statechart formale + invariante `NOTIFY_NO_QUEUE`.
- Switch default theme può sorprendere utenti esistenti. Mitigato da `legacy.toml` + nota in CHANGELOG.

### Tech debt evitato

- Senza primitive `RenderModal`: ogni modal continua a inline il proprio header → drift estetico inevitabile su step futuri.
- Senza notify component: notifiche restano confinate alla status bar → no animazioni possibili senza tornare qui.

## Alternative considerate

### A. Solo swap palette, no refactor componenti

Rifiutata: i bordi per-riga e i modali statici sono il problema visivo principale. Cambiare solo i colori non risolve.

### B. Theme engine completamente nuovo (override per-componente)

Rifiutata: ADR-019 engine è già totale e supporta hot-reload. Aggiungere layer override per componente è over-engineering per uno step di revamp.

### C. Animazioni complesse (typewriter, particles, fade-all)

Rifiutata: terminal redraw senza flicker è limitato. Fade-all richiede full-screen re-render sub-100ms → CPU/perf hit. Slide-in banner è il massimo realistico.

### D. Spezzare in 2 step (palette + animazioni)

Rifiutata: pipeline atomic-step rule. Inoltre lo split crea uno stato intermedio incoerente (palette nuova, modali vecchi).

## Note implementative (per agente esecutore)

- Riusare engine theme esistente (`internal/theme/*`). Nessun cambio struct `Theme`.
- `crush.toml` è embedded via `//go:embed` come `default.toml`. Switch default = cambio nome file embedded oppure variabile `defaultThemeName`.
- `RenderModal(title, body, hints string) string` deve stare in `internal/ui/components/modal.go`, ≤120 LOC. Test unit con golden file.
- `NotifyComponent` implementa `tea.Model`. Embedded nel root model. Riceve `NotifyMsg` via `app.Update`.
- `tea.Tick` per frame animation. NON usare goroutine custom.
- 120 LOC per file enforced via `make loc-check`.
