# ADR-005: Componenti custom vs bubbles standard

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

Il design richiede alcuni componenti UI che non esistono nella libreria bubbles standard:
1. **OTPInput** — input a celle individuali per il codice 2FA
2. **ButtonModel** — bottone SEND cliccabile con mouse
3. **StatusBar** — barra di stato con zona shortcuts e zona errori

## Decisione

Implementare questi come componenti custom nel package `internal/ui/components/`, seguendo il pattern bubbles standard (implementano `tea.Model` con Init/Update/View).

Per tutti gli altri componenti, usare bubbles standard:
- `viewport.Model` per scrolling (chat list, messaggi)
- `textinput.Model` per input single-line (phone, password, search)
- `textarea.Model` per input multiline (composizione messaggi)
- `spinner.Model` per loading
- `help.Model` per l'overlay help
- `key.Binding` per keybindings

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| Fork di bubbles | Componenti modificati a piacere | Manutenzione fork, divergenza upstream |
| Tutto custom | Controllo totale | Reinventare la ruota, molto più codice |
| **Custom solo dove serve** | **Minimo codice custom, massimo riuso** | **Inconsistenza stilistica possibile** |

## Conseguenze

- **Positive**: codebase snella, aggiornamenti bubbles facili, solo 3 componenti custom da mantenere
- **Negative**: i componenti custom devono rispettare manualmente le convenzioni di bubbles (Init/Update/View, messaggi, focus/blur)
