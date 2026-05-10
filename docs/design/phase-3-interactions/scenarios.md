# Interaction Scenarios

Sequence diagrams per gli scenari chiave del sistema.

## Scenario 1 — Startup con session valida

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App Model
    participant TG as Telegram Client
    participant SRV as Telegram Server
    participant FS as File System

    U->>APP: avvia tuilegram
    APP->>APP: Init() → tickCmd, startTelegramCmd

    APP->>FS: LoadSession("session.json")
    FS-->>APP: session data (auth key)
    APP->>TG: client.Run(ctx, callback)
    TG->>SRV: restore connection (auth key)
    SRV-->>TG: connection established

    TG->>TG: auth.IfNecessary → already authorized
    TG-->>APP: p.Send(ConnectedMsg)
    APP->>APP: state = Connecting

    TG->>SRV: query.GetDialogs()
    SRV-->>TG: dialogs list
    TG-->>APP: p.Send(DialogsLoadedMsg{chats})
    APP->>APP: state = MainView, populate ChatList

    APP->>U: render 2-panel layout (chatlist + empty conv)
```

## Scenario 2 — Aprire una conversazione

```mermaid
sequenceDiagram
    participant U as User
    participant CL as ChatList
    participant APP as App
    participant TG as Telegram Client
    participant SRV as Telegram Server
    participant CV as Conversation

    U->>CL: j/k navigate to "John Doe"
    CL->>CL: update selected index, re-render
    U->>CL: Enter
    CL->>APP: ChatSelectedMsg{chatID}
    APP->>APP: focus = Conversation, show spinner
    APP->>TG: loadChatCmd → api.MessagesGetHistory(chatID, limit=50)
    TG->>SRV: messages.getHistory
    SRV-->>TG: messages
    TG-->>APP: MessagesLoadedMsg{chatID, messages}
    APP->>CV: setMessages(messages)
    CV->>CV: render messages in viewport, scroll to bottom
    APP->>U: render conversation with messages
```

## Scenario 3 — Inviare un messaggio

```mermaid
sequenceDiagram
    participant U as User
    participant IN as Input
    participant APP as App
    participant CV as Conversation
    participant TG as Telegram Client
    participant SRV as Telegram Server

    U->>IN: Tab (focus input)
    U->>IN: types "Hello!"
    U->>IN: Enter

    IN->>APP: extract text, clear input
    APP->>CV: append optimistic message (status=Sending)
    APP->>TG: sendMessageCmd → api.MessagesSendMessage

    Note over CV: message shown with no receipt icon

    TG->>SRV: messages.sendMessage
    SRV-->>TG: UpdateShortSentMessage{id, date}
    TG-->>APP: SendResultMsg{msgID, nil}
    APP->>CV: update message status → Sent (✓)

    Note over CV: ✓ appears

    SRV-->>TG: UpdateReadHistoryOutbox{maxID}
    TG-->>APP: ReadHistoryMsg{chatID, maxID}
    APP->>CV: update status → Read (✓✓ blu)
```

## Scenario 4 — Ricevere un messaggio in tempo reale

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Client
    participant APP as App
    participant CV as Conversation
    participant CL as ChatList

    SRV->>TG: UpdateNewMessage{message, entities}
    TG->>TG: parse tg.Message → domain.Message
    TG->>TG: extract sender from entities
    TG-->>APP: p.Send(NewMessageMsg{message})

    APP->>CV: is message for open chat?
    alt chat is open
        CV->>CV: append message to viewport
        alt viewport at bottom
            CV->>CV: auto-scroll to new message
        else viewport scrolled up
            CV->>CV: increment "↓ N" counter
        end
    end

    APP->>CL: update chat.LastMessage, chat.UnreadCount++
    CL->>CL: re-sort chat list
    APP->>APP: re-render
```

## Scenario 5 — Reply a un messaggio

```mermaid
sequenceDiagram
    participant U as User
    participant CV as Conversation
    participant IN as Input
    participant APP as App
    participant TG as Telegram Client

    U->>CV: j/k to message, press 'r'
    CV->>APP: ReplyToMsg{message}
    APP->>IN: activate reply mode
    IN->>IN: show "┃ John: Hey how are..." inline

    U->>IN: types reply text
    U->>IN: Enter

    IN->>APP: send with replyToMsgID
    APP->>TG: sendMessageCmd with ReplyTo field
    APP->>IN: clear, deactivate reply mode
```

## Scenario 6 — Search globale

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App
    participant OVL as Search Overlay
    participant TG as Telegram Client
    participant SRV as Telegram Server

    U->>APP: press '/'
    APP->>OVL: open SearchOverlay

    U->>OVL: types "meeting"
    OVL->>OVL: debounce 300ms
    OVL->>TG: searchCmd → api.MessagesSearchGlobal("meeting")
    TG->>SRV: messages.searchGlobal
    SRV-->>TG: search results
    TG-->>OVL: SearchResultMsg{hits}
    OVL->>OVL: render results list

    U->>OVL: j/k to result, Enter
    OVL->>APP: OverlayCloseMsg + ChatSelectedMsg{chatID, scrollToMsgID}
    APP->>APP: open chat, scroll to message, highlight
```

## Scenario 7 — Reconnection dopo disconnect

```mermaid
sequenceDiagram
    participant TG as Telegram Client
    participant APP as App
    participant CL as ChatList
    participant SRV as Telegram Server

    Note over TG: connection drops

    TG-->>APP: p.Send(DisconnectedMsg)
    APP->>CL: header → "✕ CHATS"
    APP->>APP: show "Disconnected" in status bar

    loop exponential backoff
        TG-->>APP: p.Send(ReconnectingMsg{attempt: N})
        APP->>CL: header → "○ CHATS"

        TG->>SRV: reconnect attempt
        alt success
            SRV-->>TG: connected
            TG->>TG: restore session, restart update engine
            TG-->>APP: p.Send(ConnectedMsg)
            APP->>CL: header → "● CHATS"
            APP->>APP: clear error in status bar
        else failure
            Note over TG: wait 2^N * 100ms (max 5s)
        end
    end
```

## Scenario 8 — Typing indicator

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Client
    participant APP as App
    participant CL as ChatList
    participant CV as Conversation

    SRV->>TG: UpdateUserTyping{userID, chatID}
    TG-->>APP: p.Send(TypingMsg{chatID, userID})

    APP->>CL: find chat item, show "typing..."
    alt chat is currently open
        APP->>CV: update header → "John Doe — typing..."
    end

    Note over APP: after 5s timeout with no new TypingMsg
    APP->>CL: restore original chat name
    APP->>CV: restore header → "John Doe ● online"
```

> **Step 22 update**: batch forward / batch delete (multi-selezione) sono
> documentati in [`multi-select-flow.md`](multi-select-flow.md). Lo Scenario 9
> qui sotto descrive il flow single-msg (Step 21).

## Scenario 9 — Forward messaggio

```mermaid
sequenceDiagram
    participant U as User
    participant CV as Conversation
    participant OVL as Forward Picker
    participant APP as App
    participant TG as Telegram Client

    U->>CV: cursor on message, press 'f'
    CV->>APP: ForwardRequestMsg{message}
    APP->>OVL: open ForwardPicker

    U->>OVL: types "team" (filter)
    OVL->>OVL: fuzzy filter chat list
    U->>OVL: select "Team Dev", Enter

    OVL->>APP: forward to chatID
    APP->>TG: forwardMessageCmd
    APP->>OVL: close
    APP->>APP: status bar: "Message forwarded to Team Dev"
```

## Scenario 10 — Edit messaggio

```mermaid
sequenceDiagram
    participant U as User
    participant CV as Conversation
    participant OVL as Edit Overlay
    participant APP as App
    participant TG as Telegram Client

    U->>CV: cursor on own message, press 'e'
    CV->>APP: EditRequestMsg{message}
    APP->>OVL: open EditOverlay with message.Text

    U->>OVL: modifies text
    U->>OVL: Enter (save)

    OVL->>APP: editMessageCmd(chatID, msgID, newText)
    APP->>TG: api.MessagesEditMessage
    TG-->>APP: EditResultMsg{nil}
    APP->>OVL: close
    APP->>CV: update message text in viewport
```
