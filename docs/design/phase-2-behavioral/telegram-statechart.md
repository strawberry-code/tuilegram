# Telegram Client Statechart

Macchina a stati del client Telegram (gotd/td) come visto dal TUI.

## Connection Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Idle

    Idle --> Connecting : Start()

    state Connecting {
        [*] --> RestoringSession
        RestoringSession --> KeyExchange : no session
        RestoringSession --> Authenticating : session found
        KeyExchange --> Authenticating : key exchanged
    }

    Connecting --> Connected : auth success
    Connecting --> AuthRequired : session invalid / first run

    state AuthRequired {
        [*] --> WaitingForPhone
        WaitingForPhone --> SentCode : phone submitted
        SentCode --> WaitingForCode : code sent to user
        WaitingForCode --> Verifying : code submitted
        Verifying --> WaitingForPassword : 2FA required
        Verifying --> Connected : auth success
        Verifying --> WaitingForCode : wrong code
        WaitingForPassword --> Verifying2FA : password submitted
        Verifying2FA --> Connected : auth success
        Verifying2FA --> WaitingForPassword : wrong password
    }

    state Connected {
        [*] --> Syncing
        Syncing --> Ready : dialogs loaded, updates engine started
        Ready --> Ready : processing updates

        state Ready {
            [*] --> Idle_Ready
            Idle_Ready --> SendingMessage : send request
            SendingMessage --> Idle_Ready : send complete/error
            Idle_Ready --> LoadingHistory : history request
            LoadingHistory --> Idle_Ready : history loaded
            Idle_Ready --> Searching : search request
            Searching --> Idle_Ready : search results
        }
    }

    Connected --> Reconnecting : connection lost
    Reconnecting --> Connected : reconnected
    Reconnecting --> Reconnecting : retry (exponential backoff)
    Reconnecting --> Disconnected : max retries / context cancelled

    Connected --> Disconnected : Shutdown()
    Disconnected --> [*]
```

## Connection Status → UI Mapping

| Client State | UI Connection Indicator | Header CHATS |
|-------------|------------------------|--------------|
| `Idle` | — | — |
| `Connecting` | ○ giallo | `○ CHATS` |
| `AuthRequired` | — | (Auth flow attivo) |
| `Connected.Syncing` | ○ giallo | `○ CHATS` |
| `Connected.Ready` | ● verde | `● CHATS` |
| `Reconnecting` | ○ giallo | `○ CHATS` |
| `Disconnected` | ✕ rosso | `✕ CHATS` |

## Update Processing

```mermaid
stateDiagram-v2
    state "Connected.Ready" as CR

    state CR {
        [*] --> WaitingUpdates
        WaitingUpdates --> ProcessingUpdate : update received

        state ProcessingUpdate {
            [*] --> ClassifyUpdate
            ClassifyUpdate --> HandleNewMessage : UpdateNewMessage
            ClassifyUpdate --> HandleEdit : UpdateEditMessage
            ClassifyUpdate --> HandleDelete : UpdateDeleteMessages
            ClassifyUpdate --> HandleStatus : UpdateUserStatus
            ClassifyUpdate --> HandleTyping : UpdateUserTyping
            ClassifyUpdate --> HandleReadHistory : UpdateReadHistoryOutbox
            ClassifyUpdate --> HandleReactions : UpdateMessageReactions
            ClassifyUpdate --> HandlePinned : UpdatePinnedMessages
            ClassifyUpdate --> HandleOther : other
        }

        ProcessingUpdate --> SendToTUI : p.Send(event)
        SendToTUI --> WaitingUpdates
    }
```

## Flood Wait Handling

```mermaid
stateDiagram-v2
    state "API Call" as AC

    [*] --> AC
    AC --> Success : 200 OK
    AC --> FloodWait : FLOOD_WAIT_X
    AC --> Error : other error

    FloodWait --> Waiting : sleep X seconds
    Waiting --> AC : retry

    Success --> [*]
    Error --> [*]
```

Gestito automaticamente da `gotd/contrib/middleware/floodwait`. Il middleware:
1. Intercetta errori `FLOOD_WAIT_N`
2. Attende N secondi
3. Ritenta la richiesta
4. Opzionalmente notifica il TUI via `p.Send(FloodWaitMsg{duration})`

## Session Lifecycle

```mermaid
stateDiagram-v2
    [*] --> NoSession

    NoSession --> Creating : auth flow starts
    Creating --> Active : auth success (key generated)
    Active --> Active : operations (key reused)
    Active --> Expired : server rejects key
    Expired --> Creating : re-auth
    Active --> Destroyed : logout

    Destroyed --> [*]
```

| Stato | File system | Implicazioni |
|-------|------------|--------------|
| `NoSession` | Nessun file | Auth flow obbligatorio |
| `Creating` | File in scrittura | Auth in corso |
| `Active` | `session.json` presente (0600) | Operazioni normali |
| `Expired` | File presente ma non valido | Re-auth necessario |
| `Destroyed` | File eliminato | Logout completato |
