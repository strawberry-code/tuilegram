# Architecture Decision Records

Indice delle decisioni architetturali documentate durante il design.

| ID | Titolo | Stato | Data |
|----|--------|-------|------|
| [ADR-001](ADR-001-concurrency-model.md) | Modello di concorrenza TUI ↔ Telegram | accettato | 2026-04-09 |
| [ADR-002](ADR-002-domain-types.md) | Tipi di dominio separati da gotd/td | accettato | 2026-04-09 |
| [ADR-003](ADR-003-session-security.md) | Gestione sicurezza session file | accettato | 2026-04-09 |
| [ADR-004](ADR-004-theming-system.md) | Sistema di theming con file TOML | accettato | 2026-04-09 |
| [ADR-005](ADR-005-custom-components.md) | Componenti custom vs bubbles standard | accettato | 2026-04-09 |
| [ADR-006](ADR-006-forward-fuzzy-algorithm.md) | Algoritmo fuzzy-search del forward picker | accettato | 2026-04-24 |
| [ADR-007](ADR-007-overlay-in-flight-rpc.md) | Gestione di Esc durante RPC in volo negli overlay | accettato | 2026-04-24 |
| [ADR-008](ADR-008-batch-forward-semantics.md) | Semantica del batch forward (single-target picker reuse) | accettato | 2026-04-24 |
| [ADR-009](ADR-009-batch-delete-confirm.md) | Confirm dialog del batch delete (singolo confirm N-aware) | accettato | 2026-04-24 |
| [ADR-010](ADR-010-typing-ttl-strategy.md) | Strategia TTL per il typing indicator (timestamp + re-arm) | accettato | 2026-04-25 |
| [ADR-011](ADR-011-media-rendering-taxonomy.md) | Tassonomia media + charset waveform braille | accettato | 2026-04-25 |
| [ADR-012](ADR-012-reactions-storage-and-system-detection.md) | Reactions storage shape + system message detection | accettato | 2026-04-25 |
| [ADR-013](ADR-013-search-debounce-and-stale-results.md) | Search overlay — debounce + stale-result drop + Esc-during-RPC | accettato | 2026-04-25 |
| [ADR-014](ADR-014-inline-search-bar-vs-modal.md) | Search in chat — inline bar (no Modal) + sync local search + re-index strategy | accettato | 2026-04-25 |
| [ADR-015](ADR-015-command-palette-whichkey-help.md) | Command palette + which-key + help — Modal primitive, registry statico, mutex overlays, fuzzy subsequence, debounce 300ms | accettato | 2026-04-25 |
| [ADR-016](ADR-016-folder-source-and-filtering.md) | Folder sidebar — server-side `DialogFilter`, sidebar non-overlay, persistence reset, active-chat invariance, compact-mode skip | accettato | 2026-04-25 |
| [ADR-017](ADR-017-chat-info-data-source.md) | Chat info — Modal `placement: right`, cache-first lazy completion, counters stub, F UX-consume, omit-vs-placeholder per ChatType | accettato | 2026-04-25 |
| [ADR-018](ADR-018-responsive-layout-threshold-and-tab.md) | Responsive layout — threshold 100 cols, no-hysteresis, Tab semantics context-aware (focus cycle in Wide / panel switch in Compact), side-effects cross-threshold (sidebar auto-close, compactVisible derivation, overlay invariato) | accettato | 2026-04-25 |
| [ADR-019](ADR-019-theming-and-config-loading.md) | Theming + config loading — TOML, XDG-aware path, default embedded via `embed.FS`, override-by-key merge, fail-soft per file mancante / TOML invalido / chiavi sconosciute, schema 18 color keys + 2 gradient, accessor pattern `styles.ColorPrimary()`, no hot-reload (deferred), `compact_threshold` parametrizzabile | accettato | 2026-05-09 |
| [ADR-020](ADR-020-mouse-support.md) | Mouse support — central hit-test router in `MainModel.handleMouseMsg`, bbox cache invalidata su layout events, wheel-by-cursor (no focus), click+open atomico su chat item, tassonomia overlay dismissable vs modal (click outside chiude i dismissable, no-op su modal), focus shift + action atomic, drag/text-select deferred (escape-hatch terminale), keyboard parity invariant, compact `NO_HIDDEN_CLICK`, composer click semantics (SEND submit / textarea focus); TLA+ skip giustificato (handler sincrono single-channel) | accettato | 2026-05-10 |
| [ADR-021](ADR-021-step33-polish.md) | Step 33 polish (FINAL) — single-ADR multi-feature: pinned bar (snapshot a chat-open, single most-recent, 2-row layout, compact-visible), link rendering+open (`MessageEntities` authoritative, lipgloss underline + OSC 8, `gx` chord canonical, http(s) only), forward display (block prefix `┃` per-line, fallback chain username>name>title>hidden), status bar (dual-slot left-hint + right-error/info, focus-aware `keymapHint()`, `✕` prefix per errori, no auto-clear), sender name color (hash modulo palette[8], group-only, deterministic); 4 nuovi color keys + palette estensione ADR-019; `PINNED_STALE_DROP` riusa pattern ADR-017; `gx` riusa whichkey ADR-015; bbox invalidation trigger esteso (ADR-020 §D2); TLA+ skip giustificato (5 sub-feature sincrone, zero nuova goroutine) | accettato | 2026-05-10 |
| [ADR-022](ADR-022-step34-style-revamp.md) | Step 34 — Style revamp Crush-inspired: nuovo theme `crush.toml` default + `legacy.toml` rollback, primitive `RenderModal(title, body, hints)` con titolo inline border-top, refactor chat-list (no per-row border, selezione bg fill), status bar middot `·`, banner notify slide-in/out (NOTIFY_NO_QUEUE, NOTIFY_FRAME_BOUNDED, replace-on-new), focus border swap instant (FOCUS_SINGLE), invarianti CRUSH_THEME_TOTAL + NO_PERROW_BORDER + MODAL_TITLE_INLINE + STATUS_MIDDOT + MODAL_SINGLE; statechart formali in phase-3, mood board + token spec in phase-2; TLA+ skip giustificato (single message-loop linearization) | accettato | 2026-05-10 |

## Come aggiungere un ADR

1. Copia `ADR-template.md` con nome `ADR-NNN-titolo.md`
2. Compila tutti i campi
3. Aggiungi alla tabella sopra
