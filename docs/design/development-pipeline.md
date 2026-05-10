# Development Pipeline — 33 Step

Pipeline di sviluppo incrementale rigida. Ogni step produce un'app eseguibile con una feature aggiuntiva.

**Regola**: nessuno step viene saltato, nessuno step viene accorpato. Alla fine di ogni step il codice compila, l'app si avvia e la feature è testabile dall'utente.

---

## Fase A — Foundation (Step 1-5)

### Step 1: Skeleton bubbletea eseguibile
**Scope**: App Go minimale con bubbletea. Schermata vuota, si avvia e si chiude.
**Deliverable**:
- `cmd/tuilegram/main.go` — entry point
- `internal/ui/app.go` — root `tea.Model` con Init/Update/View
- `go.mod` con dipendenze bubbletea + lipgloss
**Test**:
- `go run ./cmd/tuilegram` → schermata vuota con bordo
- `Ctrl+Q` → l'app si chiude pulitamente
- Ridimensiona il terminale → nessun crash

### Step 2: Login screen — titolo figlet + phone input
**Scope**: Schermata di login con titolo ASCII art centrato e campo telefono.
**Deliverable**:
- `internal/ui/views/auth.go` — AuthModel con PhoneInput
- Figlet title "Tuilegram" centrato
- Label "Enter your phone number:" + `textinput` inline + hint `↵`
**Test**:
- Avvia → vedi titolo figlet centrato + campo telefono
- Digita un numero → appare nel campo
- Enter → (per ora) messaggio "Not implemented" nella status bar
- Ridimensiona → il layout si riadatta

### Step 3: Auth flow — OTP input + password input
**Scope**: Componente custom OTP a celle + input password mascherato. Navigazione tra i 3 step.
**Deliverable**:
- `internal/ui/components/otpinput.go` — OTPInputModel custom
- Aggiornamento AuthModel con step Phone → Code → Password
- Transizioni tra step (per ora simulate, senza Telegram)
**Test**:
- Avvia → phone input
- Enter → passa a OTP input (6 celle vuote)
- Digita cifre → celle si riempiono, auto-advance
- Backspace → torna alla cella precedente
- Enter (tutte piene) → passa a password input
- Digita password → caratteri mascherati `*`
- Enter → (per ora) torna alla phone input (ciclo demo)
- Esc su qualsiasi step → torna allo step precedente

### Step 4: Integrazione Telegram — connessione + session
**Scope**: Client gotd/td, session storage, connessione reale ai server Telegram.
**Deliverable**:
- `internal/telegram/client.go` — TelegramBridge wrapper
- `internal/telegram/session.go` — session storage setup
- Connessione reale, AuthRequired detection
- Loading spinner durante la connessione
**Test**:
- Avvia senza `TELEGRAM_APP_ID` → errore chiaro
- Imposta env vars → l'app si connette (spinner visibile)
- Se nessuna session → mostra login screen
- Se session valida → (per ora) mostra "Connected!" e si chiude

### Step 5: Auth flow end-to-end con Telegram
**Scope**: Login reale con Telegram. Phone → code via SMS/app → 2FA → session salvata.
**Deliverable**:
- `internal/telegram/auth.go` — integrazione auth.Flow con la UI
- Gestione errori (wrong code, wrong password, flood wait)
- Session persistita su disco dopo login riuscito
- Toast/status bar per errori
**Test**:
- Avvia → inserisci il tuo numero reale → ricevi il codice su Telegram
- Inserisci il codice → se hai 2FA, appare il campo password
- Login completato → session.json creato (verifica con `ls -la session.json`)
- Riavvia l'app → login saltato (session restore), va diretta al prossimo step
- Inserisci codice sbagliato → errore mostrato, puoi riprovare
- Inserisci password sbagliata → errore, puoi riprovare

---

## Fase B — Layout Core (Step 6-10)

### Step 6: Layout 2-panel vuoto
**Scope**: Dopo il login, mostra il layout a 2 pannelli: chat list (sinistra) + conversation (destra). Entrambi vuoti.
**Deliverable**:
- `internal/ui/views/main.go` — MainViewModel
- Layout lipgloss con JoinHorizontal
- Header `● CHATS` a sinistra, pannello vuoto con logo a destra
- Status bar in basso (1 riga)
**Test**:
- Login → vedi 2 pannelli con bordi
- Header sinistro: `● CHATS`
- Pannello destro: logo "Tuilegram" + "Select a chat"
- Status bar in basso con "? help"
- Ridimensiona → i pannelli si riadattano
- `Ctrl+Q` → chiude

### Step 7: Chat list — caricamento dialogs
**Scope**: Carica le chat da Telegram e mostra la lista con box bordered.
**Deliverable**:
- `internal/ui/views/chatlist.go` — ChatListModel
- `internal/telegram/dialogs.go` — caricamento dialogs
- `internal/model/` — tipi Chat, User, ChatID
- Chat items come box con bordo, solo nome
**Test**:
- Login → le tue chat reali appaiono nella lista sinistra
- Ogni chat è un box con bordo
- La lista è scrollabile (j/k)
- Spinner durante il caricamento

### Step 8: Chat list — colori bordo + dot system
**Scope**: Bordo colorato per tipo (viola user, verde group, blu channel, arancio bot). Dot online/unread.
**Deliverable**:
- Colori bordo per tipo di chat
- Dot verde (online) e blu (unread) a sinistra del nome
- Bordo rosso per chat selezionata
- Chat muted: dimmed + 🔇
**Test**:
- Chat private hanno bordo viola
- Gruppi hanno bordo verde
- Canali hanno bordo blu
- La chat sotto il cursore ha bordo rosso
- Chat con messaggi non letti hanno dot blu
- Utenti online hanno dot verde
- Chat muted sono attenuate con icona 🔇

### Step 9: Chat list — sorting + scrolling
**Scope**: Ordinamento corretto (pinned > unread > last message). Scrolling con viewport.
**Deliverable**:
- Sorting logic
- Viewport scrolling con `Ctrl+U`/`Ctrl+D`
- `g,g` → top, `G` → bottom
**Test**:
- Le chat pinnate sono in cima
- Le chat con unread sono sopra quelle lette
- Dentro ogni gruppo, ordinate per recenza
- `j`/`k` naviga, `Ctrl+D` scorre mezza pagina
- `g,g` va in cima, `G` in fondo

### Step 10: Chat list — connection status
**Scope**: Indicatore di connessione nell'header CHATS.
**Deliverable**:
- Gestione `ConnectedMsg`, `DisconnectedMsg`, `ReconnectingMsg`
- Header `● CHATS` (verde), `○ CHATS` (giallo), `✕ CHATS` (rosso)
**Test**:
- All'avvio → `○ CHATS` (connecting)
- Connesso → `● CHATS` (verde)
- Disattiva WiFi → `✕ CHATS` (rosso) dopo qualche secondo
- Riattiva WiFi → `○ CHATS` poi `● CHATS`

---

## Fase C — Conversation Basics (Step 11-15)

### Step 11: Aprire una conversazione — caricamento messaggi
**Scope**: Enter su una chat carica i messaggi e li mostra nel pannello destro.
**Deliverable**:
- `internal/ui/views/conversation.go` — ConversationModel
- `internal/telegram/messages.go` — caricamento history
- `internal/model/message.go` — tipo Message
- Messaggi come testo semplice nel viewport
**Test**:
- Seleziona una chat → Enter → i messaggi appaiono a destra
- Spinner durante il caricamento
- Scrollabile con j/k
- `h` o `Esc` → torna alla chat list (pannello destro torna al logo)

### Step 12: Message rendering — allineamento incoming/outgoing
**Scope**: Messaggi incoming a sinistra (teal), outgoing a destra (viola).
**Deliverable**:
- Rendering differenziato per `IsOutgoing`
- Colori: incoming teal `#38BDF8`, outgoing viola `#7D56F4`
- Allineamento corretto
**Test**:
- I tuoi messaggi appaiono a destra, in viola
- I messaggi degli altri appaiono a sinistra, in teal
- I colori sono visibili e distinguibili

### Step 13: Message grouping + timestamps
**Scope**: Messaggi consecutivi raggruppati, timestamp sotto l'ultimo del gruppo (dim).
**Deliverable**:
- Logica di grouping (stesso sender, entro 5 minuti)
- Timestamp dim sotto il gruppo
- Date separator centrato con linee
**Test**:
- 3 messaggi consecutivi dello stesso utente → nessun timestamp sui primi 2, timestamp sotto il 3°
- Messaggi di giorni diversi → separatore `────── Apr 8, 2026 ──────`
- Timestamp in grigio, non invadente

### Step 14: Conversation header
**Scope**: Header fisso in cima al pannello conversazione con nome + status.
**Deliverable**:
- Header con nome chat + online status
- Diverso per tipo: "● online" (private), "12 members" (group), "5.2k subs" (channel)
**Test**:
- Apri chat privata → "John Doe  ● online" o "last seen at 14:30"
- Apri gruppo → "Team Dev  12 members"
- Apri canale → "News  5.2k subscribers"

### Step 15: Input area + invio messaggi
**Scope**: Textarea multiline + bottone SEND. Invio messaggi reale.
**Deliverable**:
- `internal/ui/components/button.go` — ButtonModel SEND
- Textarea espandibile (1-5 righe) con scrollview
- Enter = invia, Shift+Enter = newline
- Invio messaggio reale via gotd/td
**Test**:
- `Tab` o `i` → focus sull'input
- Digita testo → appare nel campo
- Shift+Enter → nuova riga, il campo si espande
- Enter → messaggio inviato (verifica su un altro client Telegram)
- Il campo si svuota dopo l'invio
- Il messaggio appare nella conversazione (optimistic)
- Click su SEND → stessa cosa di Enter
- Esc → esci dall'input

---

## Fase D — Message Features (Step 16-20)

### Step 16: Delivery receipts
**Scope**: ✓ sent, ✓✓ delivered, ✓✓ (blu) read sui messaggi outgoing.
**Deliverable**:
- DeliveryStatus tracking
- Rendering receipt accanto al timestamp
- Update da `ReadHistoryMsg`
**Test**:
- Invia messaggio → appare `✓` dopo qualche istante
- Se il destinatario è online e lo legge → `✓✓` diventa blu
- I vecchi messaggi mostrano il loro status corretto

### Step 17: Ricezione messaggi real-time
**Scope**: Nuovi messaggi arrivano in tempo reale senza refresh.
**Deliverable**:
- `internal/telegram/updates.go` — update dispatcher
- Gestione `UpdateNewMessage` → `NewMessageMsg`
- Auto-scroll se il viewport è in fondo
- Aggiornamento unread count e LastMessage nella chat list
**Test**:
- Apri una chat → chiedi a qualcuno di mandarti un messaggio
- Il messaggio appare in tempo reale
- Se sei in fondo → auto-scroll
- Se sei scrollato su → il messaggio appare ma non scorre
- La chat list si riordina (il chat con nuovo messaggio sale)
- Il dot unread appare su chat non aperte

### Step 18: Reply a messaggi
**Scope**: Seleziona un messaggio con `r`, mostra reply bar nell'input, invia reply.
**Deliverable**:
- Message cursor nella conversazione
- `r` → attiva reply mode
- Reply bar inline nell'input (┃ preview)
- Invio con `replyToMsgID`
- Display reply nei messaggi (┃ barra colorata)
**Test**:
- `j`/`k` nella conversazione → il cursore evidenzia un messaggio
- `r` → input si attiva con `┃ John: testo originale...` sopra
- Scrivi e Enter → il messaggio viene inviato come reply
- Il reply appare con la barra ┃ e la preview del messaggio citato
- Esc → cancella il reply mode
- Verifica su un altro client che il reply è corretto

### Step 19: Edit messaggi
**Scope**: `e` su un proprio messaggio apre l'overlay di edit.
**Deliverable**:
- Edit overlay centrato con textarea
- Invio edit reale via API
- Aggiornamento messaggio nel viewport
**Test**:
- Cursore su un tuo messaggio → `e` → overlay con testo originale
- Modifica il testo → Enter → overlay si chiude
- Il messaggio mostra il testo aggiornato
- Verifica su un altro client
- `e` su un messaggio altrui → non succede nulla
- Esc → chiude l'overlay senza salvare

### Step 20: Delete messaggi + confirm dialog
**Scope**: `D` su un messaggio mostra confirm dialog, poi cancella.
**Deliverable**:
- Confirm dialog overlay (Y/N)
- Delete reale via API
- Rimozione dal viewport
**Test**:
- Cursore su un tuo messaggio → `D` → overlay "Delete this message? [Y] [N]"
- `Y` → messaggio scompare
- `N` → overlay si chiude, messaggio resta
- Verifica su un altro client che è stato cancellato
- `D` su un messaggio altrui (non admin) → non succede nulla

---

## Fase E — Advanced Messaging (Step 21-25)

### Step 21: Forward messaggi
**Scope**: `f` su un messaggio apre il forward picker, seleziona destinazione.
**Deliverable**:
- Forward picker overlay con fuzzy search
- Forward reale via API
**Test**:
- Cursore su un messaggio → `f` → overlay con lista chat
- Digita per filtrare → lista si restringe
- Enter su una chat → messaggio forwardato
- Verifica su un altro client
- Esc → chiude senza forwardare

### Step 22: Message cursor + multi-select
**Scope**: Cursore visibile, `Space` per toggle selezione, azioni batch.
**Deliverable**:
- Evidenziazione del messaggio sotto il cursore
- Checkbox toggle con Space
- Barra info "N selected | f forward | D delete"
- Forward/delete batch
**Test**:
- `j`/`k` → il cursore evidenzia un messaggio alla volta
- `Space` → checkbox `[✓]` appare
- `Space` su altri → multi-selezione
- `f` con selezione → forward di tutti i selezionati
- `D` con selezione → delete di tutti
- `Esc` → deseleziona tutto

### Step 23: Typing indicator
**Scope**: "typing..." nella chat list e nell'header conversazione.
**Deliverable**:
- Gestione `UpdateUserTyping`
- Chat list: nome sostituito con "typing..."
- Header conversazione: "John Doe — typing..."
- Timeout 5 secondi
**Test**:
- Chiedi a qualcuno di iniziare a scrivere → "typing..." appare nella lista e nell'header
- Dopo 5 secondi senza digitazione → torna al nome normale

### Step 24: Media messages
**Scope**: Rendering di foto, video, documenti, voice (waveform braille), sticker.
**Deliverable**:
- `internal/model/media.go` — MessageMedia con Icon() e Summary()
- Rendering inline: `📷 photo.jpg (1.2 MB)`
- Voice waveform con caratteri braille
- Sticker: emoji + pack name
**Test**:
- Invia una foto da un altro client → appare come `📷 photo.jpg (1.2 MB)`
- Invia un voice message → appare con waveform braille `🎤 ▁▂▃▅▇█▆▄▃▂ 0:42`
- Invia un document → `📎 file.pdf (2.4 MB)`
- Invia uno sticker → emoji + pack name

### Step 25: Reactions + system messages
**Scope**: Reazioni sotto i messaggi, system messages centrati.
**Deliverable**:
- Rendering reazioni: `👍 3  ❤️ 2  😂 1`
- System messages centrati dim: `── Alice joined ──`
- Gestione `UpdateMessageReactions`
**Test**:
- Messaggi con reazioni → riga emoji sotto
- Un utente entra nel gruppo → system message centrato
- Aggiungi una reazione da un altro client → appare in tempo reale

---

## Fase F — Overlays & Search (Step 26-28)

### Step 26: Search globale
**Scope**: `/` apre overlay di ricerca globale, risultati cliccabili.
**Deliverable**:
- `internal/ui/views/search.go` — SearchModel overlay
- Ricerca via `api.MessagesSearchGlobal`
- Debounce 300ms
- Jump to result con Enter
**Test**:
- `/` → overlay search appare
- Digita una query → risultati appaiono (con debounce)
- `j`/`k` → naviga tra risultati
- Enter → overlay si chiude, chat aperta al messaggio trovato
- Esc → chiude senza azione

### Step 27: Search in conversazione + Ctrl+F
**Scope**: `Ctrl+F` cerca nella conversazione aperta, highlight matches.
**Deliverable**:
- Ricerca locale nella conversazione
- Highlight delle occorrenze nel viewport
- Navigazione tra risultati
**Test**:
- `Ctrl+F` con una chat aperta → barra di ricerca
- Digita → occorrenze evidenziate nel viewport
- Enter/`n` → salta alla prossima occorrenza
- Esc → chiude

### Step 28: Command palette + which-key + help
**Scope**: `Ctrl+P` command palette, prefix which-key, `?` help overlay.
**Deliverable**:
- Command palette con fuzzy search
- Which-key overlay (300ms timeout su prefix g, z, etc.)
- Help overlay con tutti i keybindings
**Test**:
- `Ctrl+P` → overlay con lista comandi
- Digita "mute" → filtro fuzzy
- Enter → esegui comando
- Premi `g` → dopo 300ms appare which-key con opzioni (g=top, G=bottom, u=unread...)
- `?` → overlay help con tutti i keybindings

---

## Fase G — Panels & Layout (Step 29-30)

### Step 29: Folder sidebar + chat info
**Scope**: `F` toggle folder sidebar. `i` apre chat info overlay.
**Deliverable**:
- `internal/ui/views/folders.go` — FolderModel
- `internal/ui/views/chatinfo.go` — ChatInfoModel overlay
- Sidebar con folder list (All Chats, Personal, Work...)
- Chat info: nome, username, phone, bio, shared media count
**Test**:
- `F` → folder sidebar appare a sinistra
- Seleziona "Work" → la chat list si filtra
- `F` → sidebar sparisce
- `i` con una chat aperta → overlay info a destra
- Mostra nome, @username, stato, phone (se visibile)
- Esc → chiude info

### Step 30: Responsive layout + compact mode
**Scope**: Layout responsive. Sotto 100 colonne: single panel con Tab switching.
**Deliverable**:
- Calcolo layout da `WindowSizeMsg`
- Compact mode: un pannello alla volta
- Tab switching tra lista e conversazione
**Test**:
- Riduci il terminale sotto 100 colonne → passa a single panel
- Tab → alterna tra chat list e conversazione
- Ingrandisci → torna a 2 pannelli
- Tutto funziona in entrambe le modalità

---

## Fase H — Polish & System (Step 31-33)

### Step 31: Theming + config
**Scope**: File `theme.toml` e `config.toml`, tema default embedded.
**Deliverable**:
- Lettura `~/.config/tuilegram/config.toml`
- Lettura `~/.config/tuilegram/theme.toml`
- Tema default embedded nel binario
- Tutti i colori derivati dal tema
**Test**:
- Avvia senza config → tema default funziona
- Crea `theme.toml` con colori diversi → riavvia → colori cambiati
- Modifica `config.toml` → comportamento cambiato

### Step 32: Mouse support
**Scope**: Scroll wheel + click su chat items + click SEND.
**Deliverable**:
- Mouse wheel per scroll viewport e chat list
- Click su chat item → seleziona e apri
- Click su SEND → invia
**Test**:
- Scroll con mouse wheel → viewport scorre
- Click su una chat → si apre
- Click su SEND → messaggio inviato
- Tutto funziona anche senza mouse (keyboard-only)

### Step 33: Pinned messages + links + forward display + status bar polish
**Scope**: Barra pinned sotto header, link navigabili, forward con barra ┃, status bar completa.
**Deliverable**:
- Pinned message bar
- Link sottolineati, Enter apre browser
- Forward display con `┃ From @source`
- Status bar: shortcuts sinistra + errori destra
- Sender name colorato nei gruppi
**Test**:
- Chat con messaggio pinnato → barra visibile sotto l'header
- Messaggio con link → sottolineato, navigabile
- Messaggio forwardato → mostra `┃ From @source`
- Errori appaiono nella status bar a destra
- Nei gruppi, nome mittente colorato sul primo messaggio del gruppo

### Step 34: Style revamp — Crush-inspired UI overhaul
**Scope**: Restyling profondo dell'intera UI per allinearla allo stile di Charmland Crush. Palette, bordi, modali, status bar, animazioni.
**Deliverable**:
- Nuovo theme `crush.toml` (dark purple-magenta minimal) come default
- Refactor chat-list: rimozione bordi rosa/azzurri per riga, selezione = bg fill viola scuro
- Bordi unificati: 1px sottile color `border-default` (no varianti per status)
- Modali Crush-style: titolo nel border-top (`╭─ Title ───╮`), padding 2x1
- Status bar bottom: separator middot `·`, dim text, hint compatti
- Empty state: brand ASCII grande + sidebar info pannello
- Animazioni: notify banner slide-in/out (success/error), modal mount fade, focus transition
- Sender colors desaturati (palette monocromatica violet)
**Test**:
- Avvio → palette dark purple, no rosa saturo nelle chat
- Selezione chat → bg fill viola, no border colorato
- Apertura command palette → bordo Crush con titolo inline
- Status bar bottom → middot tra hint, ultimo errore a destra
- Notifica successo/errore → banner verde/rosso slide in dal basso, autohide 3s
- Nessuna regressione: tutti gli step 1-33 continuano a funzionare
- Switch theme: `crush` (default) ↔ `legacy` (vecchio default) tramite config

---

## Status Tracking

| Step | Nome | Stato |
|------|------|-------|
| 1 | Skeleton bubbletea eseguibile | `DONE` |
| 2 | Login screen — titolo figlet + phone input | `DONE` |
| 3 | Auth flow — OTP + password | `DONE` |
| 4 | Integrazione Telegram — connessione + session | `DONE` |
| 5 | Auth flow end-to-end | `DONE` |
| 6 | Layout 2-panel vuoto | `DONE` |
| 7 | Chat list — caricamento dialogs | `DONE` |
| 8 | Chat list — colori + dot system | `DONE` |
| 9 | Chat list — sorting + scrolling | `DONE` |
| 10 | Chat list — connection status | `DONE` |
| 11 | Conversazione — caricamento messaggi | `DONE` |
| 12 | Message rendering — incoming/outgoing | `DONE` |
| 13 | Message grouping + timestamps | `DONE` |
| 14 | Conversation header | `DONE` |
| 15 | Input area + invio messaggi | `DONE` |
| 16 | Delivery receipts | `DONE` |
| 17 | Ricezione messaggi real-time | `DONE` |
| 18 | Reply a messaggi | `DONE` |
| 19 | Edit messaggi | `DONE` |
| 20 | Delete messaggi | `DONE` |
| 21 | Forward messaggi | `DONE` |
| 22 | Message cursor + multi-select | `DONE` |
| 23 | Typing indicator | `DONE` |
| 24 | Media messages | `DONE` |
| 25 | Reactions + system messages | `DONE` |
| 26 | Search globale | `DONE` |
| 27 | Search in conversazione | `DONE` |
| 28 | Command palette + which-key + help | `DONE` |
| 29 | Folder sidebar + chat info | `DONE` |
| 30 | Responsive layout + compact mode | `DONE` |
| 31 | Theming + config | `DONE` |
| 32 | Mouse support | `DONE` |
| 33 | Pinned + links + forward display + polish | `DONE` |
| 34 | Style revamp — Crush-inspired UI overhaul | `DONE` |
