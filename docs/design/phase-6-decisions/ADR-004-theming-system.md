# ADR-004: Sistema di theming con file TOML

**Stato**: accettato
**Data**: 2026-04-09

## Contesto

L'utente ha richiesto un sistema di theming completo con file `.theme`, ispirato a btop. I colori sono fondamentali per la UX: distinguono tipi di chat, stati dei messaggi, online/offline, pannelli focused/unfocused.

## Decisione

1. **Formato**: TOML, con sezioni `[colors]` e `[borders]`.
2. **Percorso**: `~/.config/tuilegram/theme.toml`.
3. **Default theme**: embedded nel binario, non richiede file esterno.
4. **Hot reload**: no nella v1. Cambio tema richiede restart.
5. **Colori**: hex truecolor (`#RRGGBB`). lipgloss gestisce il downsampling automatico a 256/16 colori.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| YAML | Più comune | Più verboso, indentazione significativa |
| JSON | Universale | No commenti, scomodo da editare manualmente |
| **TOML** | **Chiaro, commenti, sezioni, standard per config Go** | **Meno diffuso di JSON/YAML** |

## Conseguenze

- **Positive**: gli utenti possono personalizzare completamente la palette, temi condivisibili come file, allineato con il config.toml
- **Negative**: complessità nel garantire che tutti i colori siano definiti (fallback al default per chiavi mancanti)
