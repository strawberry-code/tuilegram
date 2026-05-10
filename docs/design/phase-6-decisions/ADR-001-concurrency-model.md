# ADR-001: Modello di concorrenza TUI ↔ Telegram

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

tuilegram ha due loop concorrenti: il TUI loop (bubbletea) e il client Telegram (gotd/td). Servono un meccanismo di comunicazione thread-safe e un modello di ownership dei dati.

## Decisione

**Comunicazione unidirezionale via `tea.Program.Send()`**: il client Telegram invia eventi al TUI loop tramite `p.Send(msg)`. Il TUI loop invia richieste al client Telegram tramite `tea.Cmd` (funzioni asincrone che possono chiamare l'API e restituire un `tea.Msg`).

**Ownership esclusiva del Model**: il `tea.Model` è accessibile SOLO dal TUI loop. Nessuna goroutine esterna legge o scrive il Model.

**No mutex, no shared state**: tutta la comunicazione avviene via messaggi. Nessun lock, nessuna variabile condivisa.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| Shared state + mutex | Accesso diretto, meno boilerplate | Race conditions, deadlock risk, debugging difficile |
| Channel bidirezionale | Più esplicito | Complessità, rischio di channel blocking |
| **Message passing via p.Send()** | **Thread-safe by design, zero lock, allineato con Elm Architecture** | **Più boilerplate per i tipi messaggio** |

## Conseguenze

- **Positive**: zero rischio di data race, debugging lineare (ogni stato è il risultato di una sequenza di messaggi), testabilità (si possono iniettare messaggi nel Model)
- **Negative**: più tipi `tea.Msg` da definire e gestire, latenza indiretta (un evento Telegram passa per il channel prima di essere processato)
- **Rischi**: il channel di `p.Send()` potrebbe accumulare messaggi se il TUI loop è bloccato su un render lungo — mitigato dalla velocità di bubbletea
