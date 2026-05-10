# ADR-006: Gestione credenziali Telegram API

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

Telegram richiede app_id e app_hash per ogni applicazione che usa le MTProto API. Per distribuire tuilegram via Homebrew, gli utenti non devono dover registrarsi su my.telegram.org.

## Decisione

Credenziali **embedded nel binario** come costanti Go, con **override via env var** per sviluppatori.

- Le credenziali identificano l'applicazione, non l'utente — non sono segreti
- Telegram Desktop e altri client open source fanno lo stesso
- `TELEGRAM_APP_ID` e `TELEGRAM_APP_HASH` come env var hanno priorità sulle costanti embedded

## Conseguenze

- **Positive**: `brew install tuilegram && tuilegram` funziona subito, zero configurazione per l'utente finale
- **Negative**: le credenziali sono nel codice sorgente (ma non sono segreti)
- **TODO**: registrare "tuilegram" su my.telegram.org e compilare le costanti in `config.go`
