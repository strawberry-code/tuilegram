# ADR-003: Gestione sicurezza session file

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

Il file di sessione Telegram contiene una auth key di 256 byte che permette l'impersonazione completa dell'account utente. È il segreto più critico dell'applicazione.

## Decisione

1. **Permessi file**: `0600` (solo il proprietario può leggere/scrivere). `session.FileStorage` di gotd/td fa questo automaticamente.
2. **Percorso**: `./session.json` nella directory corrente (configurabile).
3. **Gitignore**: `*.session`, `session.json`, `*.session.json` nel `.gitignore`.
4. **No encryption at rest nella v1**: la session è salvata in chiaro. L'encryption è demandata alla crittografia del disco (FileVault, LUKS).
5. **No session string**: non supportiamo session strings (come gotgproto). Il file è l'unico formato.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **File con permessi 0600** | **Semplice, standard gotd/td, funziona subito** | **Leggibile se il disco non è cifrato** |
| Encryption con password utente | Protezione extra | UX peggiore (password ad ogni avvio), complessità crypto |
| Keychain del sistema operativo | Integrazione nativa | Diverso per ogni OS, complessità |
| Session string in env var | Nessun file su disco | UX difficile, string lunga, facilmente esposta in shell history |

## Conseguenze

- **Positive**: zero complessità aggiuntiva, allineato con le pratiche di gotd/td
- **Negative**: se il disco non è cifrato, chiunque acceda al file può impersonare l'utente
- **Rischi**: commit accidentale del file — mitigato da `.gitignore`. In futuro, v2 potrebbe aggiungere encryption opzionale
