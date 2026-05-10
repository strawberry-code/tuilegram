# ADR-002: Tipi di dominio separati da gotd/td

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

gotd/td genera migliaia di tipi Go dal schema TL di Telegram. Questi tipi sono verbosi, con molti campi opzionali e union types complessi. L'UI ha bisogno di tipi semplificati ottimizzati per il rendering.

## Decisione

Definire tipi di dominio interni (`internal/model/`) separati dai tipi gotd/td (`tg.*`). Il mapping avviene in un package dedicato (`internal/telegram/convert/`).

Il codice UI non importa mai il package `tg` direttamente.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| Usare direttamente `tg.*` ovunque | Zero mapping code | UI accoppiata allo schema Telegram, tipi enormi, breaking changes ad ogni update schema |
| Wrapper sottile su `tg.*` | Meno duplicazione | Leaky abstraction, UI comunque esposta alla complessità |
| **Tipi dominio separati + mapping** | **Disaccoppiamento, UI semplice, testabile** | **Boilerplate di conversione** |

## Conseguenze

- **Positive**: l'UI lavora con tipi puliti e predicibili, gli aggiornamenti dello schema TL richiedono modifiche solo nel layer di conversione, i tipi di dominio possono essere testati senza connessione Telegram
- **Negative**: codice di mapping da mantenere, possibile drift tra schema Telegram e dominio interno
- **Rischi**: dimenticare di mappare nuovi campi quando lo schema Telegram viene aggiornato — mitigato con test di integrazione
