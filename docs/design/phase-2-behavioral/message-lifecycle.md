# Message Lifecycle

State machine del ciclo di vita di un messaggio, dalla composizione alla consegna.

## Outgoing Message Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Composing

    state Composing {
        [*] --> Empty
        Empty --> HasText : user types
        HasText --> Empty : clear
        HasText --> HasText : edit text
    }

    Composing --> Sending : Enter (send)

    state Sending {
        [*] --> Queued
        Queued --> InFlight : cmd started
        InFlight --> Sent : server ACK (UpdateShortSentMessage)
        InFlight --> Failed : error
    }

    Sent --> Delivered : UpdateMessageID received
    Delivered --> Read : UpdateReadHistoryOutbox (maxID >= msgID)

    Failed --> Sending : retry (r)
    Failed --> Discarded : dismiss

    state "Post-Send" as PS {
        Sent --> Edited : edit action
        Delivered --> Edited : edit action
        Read --> Edited : edit action

        Sent --> Deleted : delete action
        Delivered --> Deleted : delete action
        Read --> Deleted : delete action
    }

    Edited --> Sent : edit confirmed by server
    Deleted --> [*]
    Discarded --> [*]
```

## Incoming Message Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Received

    state Received {
        [*] --> Unread
        Unread --> Read : chat opened + scroll to msg / markRead
    }

    Received --> Edited : UpdateEditMessage
    Received --> Deleted : UpdateDeleteMessages

    Edited --> Read : already read
    Deleted --> [*]
```

## Delivery Status Transitions

```mermaid
stateDiagram-v2
    direction LR

    [*] --> Sending
    Sending --> Sent : ✓
    Sent --> Delivered : ✓✓
    Delivered --> Read : ✓✓ (blu)
    Sending --> Failed : ✕
    Failed --> Sending : retry
```

| Transizione | Trigger | Telegram Update |
|-------------|---------|-----------------|
| → Sending | User presses Enter | (local) |
| Sending → Sent | Server conferma | `UpdateShortSentMessage` o `UpdateNewMessage` (own) |
| Sent → Delivered | Consegnato al peer | `UpdateMessageID` (in alcuni casi) |
| Delivered → Read | Peer apre la chat | `UpdateReadHistoryOutbox{maxID}` |
| Sending → Failed | Errore di rete o API | error da `MessagesSendMessage` |

**Nota**: Telegram non ha un concetto esplicito di "Delivered" per tutti i tipi di chat. Per le chat private, `Sent` e `Delivered` sono spesso lo stesso evento. La distinzione è più rilevante per i gruppi. Nella UI, semplifichiamo: `✓` = server ha ricevuto, `✓✓` = consegnato/letto.

## Message Edit Flow

```mermaid
stateDiagram-v2
    [*] --> Normal

    Normal --> EditRequested : user presses 'e' on own message
    EditRequested --> EditOverlayOpen : overlay opens with original text
    EditOverlayOpen --> EditSubmitting : Enter (save)
    EditOverlayOpen --> Normal : Esc (cancel)
    EditSubmitting --> EditConfirmed : server confirms
    EditSubmitting --> EditOverlayOpen : error (show in status bar)
    EditConfirmed --> Normal : message updated in viewport
```

Vincoli edit:
- Solo messaggi con `IsOutgoing = true`
- Solo messaggi di meno di 48 ore (limitazione Telegram)
- Solo messaggi di testo (non media)
- Il messaggio editato mostra "(edited)" o un indicatore

## Message Delete Flow

```mermaid
stateDiagram-v2
    [*] --> Normal

    Normal --> DeleteRequested : user presses 'D' on own message
    DeleteRequested --> ConfirmDialogOpen : overlay "Delete this message?"
    ConfirmDialogOpen --> Deleting : Y
    ConfirmDialogOpen --> Normal : N
    Deleting --> Deleted : server confirms
    Deleting --> Normal : error
    Deleted --> [*] : message removed from viewport
```

Vincoli delete:
- Messaggi propri: sempre cancellabili
- Messaggi altrui: cancellabili solo in chat dove l'utente è admin
- Delete "for everyone" vs "for me" — nella v1 usiamo "for everyone" di default per i propri messaggi

## Message Forward Flow

```mermaid
stateDiagram-v2
    [*] --> Normal

    Normal --> SelectMessages : user presses 'f' (single) or Space+f (multi)
    SelectMessages --> ForwardPickerOpen : overlay with chat list
    ForwardPickerOpen --> TargetSelected : user picks destination
    ForwardPickerOpen --> Normal : Esc
    TargetSelected --> Forwarding : confirm
    Forwarding --> Forwarded : server confirms
    Forwarding --> ForwardPickerOpen : error
    Forwarded --> Normal : close overlay, optionally open target chat
```

## Reply Flow

```mermaid
stateDiagram-v2
    [*] --> Normal

    Normal --> ReplyActivated : user presses 'r' on message
    ReplyActivated --> InputFocusedWithReply : input gains focus, reply bar shown

    state InputFocusedWithReply {
        [*] --> ComposingReply
        ComposingReply --> Sending : Enter
        ComposingReply --> Normal : Esc (cancel reply)
    }

    Sending --> Normal : send result
```

## Reactions Flow

```mermaid
stateDiagram-v2
    [*] --> NoReaction

    NoReaction --> ReactionPicker : user presses reaction key (TBD)
    ReactionPicker --> HasReaction : emoji selected
    ReactionPicker --> NoReaction : Esc
    HasReaction --> NoReaction : toggle same reaction off
    HasReaction --> HasReaction : change reaction

    note right of HasReaction : Reactions are displayed\nas emoji row under message
```

**Nota**: Il meccanismo per aggiungere reazioni (keybinding, picker emoji) è da definire nella v1. La visualizzazione delle reazioni ricevute è già specificata.
