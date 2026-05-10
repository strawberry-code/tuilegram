# Mouse Routing — Sequence Diagrams (Step 32)

Flussi runtime del **mouse support** introdotto nello Step 32.
Complementare allo statechart in
[`../phase-2-behavioral/mouse-routing.md`](../phase-2-behavioral/mouse-routing.md).

Nove scenari coprono i path interessanti:

1. **Wheel su conversation viewport (Wide mode)** — scroll messages
   by cursor position.
2. **Click su chat item (Wide, focus = Conversation)** — focus shift
   ChatList + open chat (atomic).
3. **Click su SEND button** — submit message.
4. **Click su chat item in Compact (`compactVisible = ChatList`)** —
   open chat + auto-switch a `CompactConversation`.
5. **Click outside dismissable overlay (cmdPalette)** — close.
6. **Click outside modal forwardPicker** — no-op.
7. **Wheel su chat list (Wide, cursor over chatList)** — chatList
   scroll.
8. **Wheel in Compact (cursor over visible panel)** — visible panel
   scrolls; hidden panel ignored.
9. **Click in regione hidden in Compact** — no-op (NO_HIDDEN_CLICK).

## 1. Wheel su conversation viewport (Wide mode)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant VP as conversation.viewport (bubbles)
    participant V as View

    Note over APP: state: layoutMode=Wide, activePanel=ChatList,<br/>activeOverlay=none, bboxes valid

    U->>TEA: scroll wheel down (cursor at X=80, Y=15)
    TEA->>APP: tea.MouseMsg{X:80, Y:15, Action:Press, Button:WheelDown}
    APP->>ROUT: dispatch MouseMsg
    ROUT->>ROUT: msg.IsWheel() == true → WheelDispatch
    ROUT->>BBOX: resolveWheel(80, 15)
    BBOX-->>ROUT: ConversationViewport (bbox match)
    ROUT->>VP: viewport.Update(MouseMsg{WheelDown})
    VP->>VP: scroll down by N lines (bubbles internal)
    VP-->>ROUT: updated viewport (offset advanced)
    ROUT-->>APP: (no msg emitted; in-place mutation)
    APP->>V: re-render Wide layout with scrolled viewport
    V-->>U: messaggi scorrono giù
    Note over APP: activePanel UNCHANGED (= ChatList);<br/>WHEEL_BY_POSITION invariant: focus indipendente<br/>dal target del wheel
```

**Punto chiave**: `activePanel = ChatList` ma il wheel sopra il
viewport scrolla il viewport. Wheel by cursor position, non by focus
(D3, ADR-020).

## 2. Click su chat item (Wide, focus = Conversation)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant CL as ChatListModel
    participant CONV as ConversationModel
    participant V as View

    Note over APP: state: layoutMode=Wide, activePanel=Conversation,<br/>inputFocus=true, activeChatID="John",<br/>activeOverlay=none

    U->>TEA: left click at (X=10, Y=8) — over chat row "Mom" (index 2)
    TEA->>APP: tea.MouseMsg{X:10, Y:8, Action:Press, Button:Left}
    APP->>ROUT: dispatch MouseMsg
    ROUT->>ROUT: Action=Press, Button=Left → ClickDispatch<br/>activeOverlay==none → ResolvingWidget
    ROUT->>BBOX: resolveClick(10, 8)
    BBOX-->>ROUT: Resolved{ChatList} (bbox match)
    ROUT->>ROUT: derive chat index from Y - chatList.bbox.y0<br/>row index = 2 → chats[2] = "Mom"
    Note over ROUT: D6: focus shift first (atomic)<br/>activePanel := ChatList<br/>inputFocus := false
    ROUT->>CL: SelectIndex(2) [internal call]
    CL->>CL: selected := 2; ensureVisible()
    ROUT->>APP: emit ChatSelectedMsg{ChatID("Mom")}
    APP->>CONV: handle ChatSelectedMsg → load history<br/>(loadChatCmd, MessagesLoadedMsg eventually)
    APP->>V: re-render Wide layout: chatList (cursor at Mom),<br/>conversation (loading Mom's messages)
    V-->>U: cursor jumps to "Mom"; conversation header changes;<br/>messages start loading
    Note over APP: activePanel: Conversation → ChatList (focus shifted);<br/>activeChatID: "John" → "Mom" (atomic, ChatSelectedMsg);<br/>inputFocus: true → false (focus moved to chatList)
```

**Punto chiave**: click+open atomico (D4) + focus shift atomico (D6).
L'utente che era in input mode su John si ritrova focus su chatList
con Mom selezionata e in caricamento. Equivalente di "Tab → Tab → j → j → Enter"
da keyboard (KEYBOARD_PARITY).

## 3. Click su SEND button

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant CONV as ConversationModel
    participant TG as Telegram bridge
    participant V as View

    Note over APP: state: layoutMode=Wide, activeChatID="John",<br/>activePanel=Conversation, inputFocus=true,<br/>textarea.Value()="Hello!", sendBtn.Active=true

    U->>TEA: left click at (X=88, Y=22) — over SEND button bbox
    TEA->>APP: tea.MouseMsg{X:88, Y:22, Action:Press, Button:Left}
    APP->>ROUT: dispatch MouseMsg
    ROUT->>ROUT: ClickDispatch → ResolvingWidget
    ROUT->>BBOX: resolveClick(88, 22)
    Note over BBOX: z-order: SendButton tested before InputArea<br/>(sendButton ⊂ inputArea)
    BBOX-->>ROUT: Resolved{SendButton}
    ROUT->>ROUT: check sendBtn.Active == true → proceed<br/>(if false: SENDBUTTON_INACTIVE_NO_OP)
    ROUT->>CONV: appendOptimistic("Hello!") [same path as Enter]
    CONV->>CONV: textarea.Reset(); replyTo := nil;<br/>append optimistic msg with StatusSending
    CONV->>APP: return tea.Cmd → SendMessageCmd
    APP->>TG: bridge.SendMessage("John", "Hello!")
    TG-->>APP: (later) MessageSentMsg{id, ok}
    APP->>CONV: markLastSent(StatusDelivered)
    APP->>V: re-render conversation (msg appeared, then ✓ delivered)
    V-->>U: "Hello!" appears outgoing; ✓ delivery tick
    Note over APP: KEYBOARD_PARITY: identical effect to Enter<br/>in textarea with non-empty value
```

**Punto chiave**: click su SEND = stesso path semantico di Enter
(`appendOptimistic`). Nessun nuovo Cmd, nessuna nuova msg type.

### Variante: click su SEND con textarea vuota

```mermaid
sequenceDiagram
    participant U as User
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache

    Note over ROUT: state: textarea.Value()="", sendBtn.Active=false

    U->>ROUT: tea.MouseMsg{Press, Left} on SEND bbox
    ROUT->>BBOX: resolveClick → Resolved{SendButton}
    ROUT->>ROUT: sendBtn.Active == false<br/>→ no-op (SENDBUTTON_INACTIVE_NO_OP)
    Note over ROUT: state UNCHANGED; user sees no visual feedback<br/>(button is rendered in disabled tone)
```

**Punto chiave**: SENDBUTTON_INACTIVE_NO_OP (invariante 10) — coerente
con visual disabled.

## 4. Click su chat item in Compact (`compactVisible = ChatList`)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant CL as ChatListModel
    participant V as View

    Note over APP: state: layoutMode=Compact, compactVisible=ChatList,<br/>activeOverlay=none, activeChatID=nil,<br/>bboxes: ChatList valid, Conversation=nil (hidden)

    U->>TEA: left click at (X=10, Y=5) — over chat row "Anna"
    TEA->>APP: tea.MouseMsg{X:10, Y:5, Action:Press, Button:Left}
    APP->>ROUT: dispatch
    ROUT->>BBOX: resolveClick(10, 5)
    BBOX-->>ROUT: Resolved{ChatList}<br/>(Conversation bbox is nil → not tested)
    ROUT->>ROUT: derive chat index = 0 → chats[0] = "Anna"
    Note over ROUT: D6 focus shift: activePanel := ChatList (already)
    ROUT->>CL: SelectIndex(0)
    ROUT->>APP: emit ChatSelectedMsg{ChatID("Anna")}
    APP->>APP: handle ChatSelectedMsg in Compact:<br/>load history + compactVisible := CompactConversation<br/>(eredita scenario 5 di responsive-layout-flow.md)
    APP->>APP: recomputeBboxes() (LayoutPanelSwitch invalidates bbox)
    APP->>V: re-render Compact.ShowingConversation
    V-->>U: chat list disappears; "Anna" conversation full-width
    Note over APP: bbox cache invalidated and re-computed:<br/>Conversation bbox now valid;<br/>ChatList bbox now nil (hidden in this mode)
```

**Punto chiave**: click in Compact non solo apre la chat ma triggera
anche il panel switch (Step 30). L'invariante `MOUSE_NEVER_FLIPS_LAYOUTMODE`
resta intatta (resta `Compact`); solo `compactVisible` cambia.

## 5. Click outside dismissable overlay (cmdPalette)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant V as View

    Note over APP: state: activeOverlay=cmdPalette,<br/>palette bbox = (centered, ~60×20 cols)

    U->>TEA: left click at (X=5, Y=2) — top-left, outside palette
    TEA->>APP: tea.MouseMsg{X:5, Y:2, Action:Press, Button:Left}
    APP->>ROUT: dispatch
    ROUT->>ROUT: ClickDispatch → CheckingOverlay<br/>activeOverlay == cmdPalette
    ROUT->>BBOX: inside(overlayBbox(cmdPalette), 5, 2)?
    BBOX-->>ROUT: false → OverlayOutside
    ROUT->>ROUT: isDismissable(cmdPalette) == true<br/>→ ClosingOverlay
    ROUT->>APP: emit CmdPaletteCloseMsg
    APP->>APP: handle CmdPaletteCloseMsg<br/>activeOverlay := none
    APP->>APP: recomputeBboxes() (overlay disappeared)
    APP->>V: re-render base layout (palette gone)
    V-->>U: cmd palette disappears; underlying layout visible
    Note over APP: equivalent to pressing Esc (KEYBOARD_PARITY)
```

**Punto chiave**: click outside dismissable = `Esc` equivalent. Stessa
msg type emessa (`CmdPaletteCloseMsg`), stesso effetto.

## 6. Click outside modal forwardPicker (no-op)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant V as View

    Note over APP: state: forwardPicker.Active=true,<br/>(picker is modal — ADR-007/008)

    U->>TEA: left click at (X=5, Y=2) — outside picker
    TEA->>APP: tea.MouseMsg{X:5, Y:2, Action:Press, Button:Left}
    APP->>ROUT: dispatch
    ROUT->>BBOX: inside(forwardPicker.bbox, 5, 2)?
    BBOX-->>ROUT: false → OverlayOutside
    ROUT->>ROUT: isDismissable(forwardPicker) == false<br/>→ NoOpModal
    ROUT-->>APP: (nessun msg emesso)
    APP->>V: re-render unchanged (no state mutation)
    V-->>U: forwardPicker still visible; nothing happened
    Note over APP: state UNCHANGED;<br/>user must press Esc or Enter to interact;<br/>respects ADR-007 (in-flight RPC) + ADR-008 (batch semantics)
```

**Punto chiave**: modal overlay = no-op silenzioso su click outside.
Coerente con ADR-007 (cancel deve essere gesture esplicita).

## 7. Wheel su chat list (Wide, cursor over chatList)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant CL as ChatListModel
    participant V as View

    Note over APP: state: layoutMode=Wide, activePanel=Conversation,<br/>inputFocus=true (writing message),<br/>chatList: 50 chats, selected=10, offset=5

    U->>TEA: scroll wheel down (cursor at X=10, Y=12, over chatList)
    TEA->>APP: tea.MouseMsg{X:10, Y:12, Action:Press, Button:WheelDown}
    APP->>ROUT: dispatch
    ROUT->>ROUT: IsWheel() → WheelDispatch
    ROUT->>BBOX: resolveWheel(10, 12)
    BBOX-->>ROUT: ChatList (bbox match)
    ROUT->>CL: chatList.handleMouse(MouseMsg{WheelDown})<br/>[existing handler in chatlist_nav.go]
    CL->>CL: selected := 11; ensureVisible() (offset adjusts)
    ROUT-->>APP: (no msg)
    APP->>V: re-render: chatList cursor advanced;<br/>conversation panel UNCHANGED (still showing previous chat)
    V-->>U: chat list scrolls down by 1 row
    Note over APP: activePanel UNCHANGED (= Conversation);<br/>activeChatID UNCHANGED;<br/>WHEEL_BY_POSITION respected
```

**Punto chiave**: scrollare la chatlist con il mouse non chiude la
conversation aperta. Il wheel sposta solo il `chatList.selected`
(esattamente come `j` keyboard quando focus = ChatList; ma qui senza
cambiare focus). Per **aprire** un'altra chat serve ancora click o
Enter — wheel = scroll, non commit.

## 8. Wheel in Compact (cursor over visible panel)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant VP as conversation.viewport
    participant V as View

    Note over APP: state: layoutMode=Compact, compactVisible=Conversation,<br/>bboxes: ConversationViewport valid, ChatList=nil

    U->>TEA: wheel down at (X=40, Y=10) — over viewport
    TEA->>APP: tea.MouseMsg{X:40, Y:10, Action:Press, Button:WheelDown}
    APP->>ROUT: dispatch
    ROUT->>BBOX: resolveWheel(40, 10)
    BBOX-->>ROUT: ConversationViewport
    ROUT->>VP: viewport.Update(MouseMsg{WheelDown})
    VP-->>ROUT: scrolled
    APP->>V: re-render Compact.ShowingConversation (scrolled)
    V-->>U: messages scroll

    Note over APP: --- separately: wheel in "ChatList area"<br/>(but ChatList is hidden in compactVisible=Conversation) ---

    U->>TEA: wheel down at (X=5, Y=10) — same screen,<br/>but corresponds to nothing visible (no separate chatList area in Compact)
    TEA->>APP: tea.MouseMsg{X:5, Y:10, Action:Press, Button:WheelDown}
    APP->>ROUT: dispatch
    ROUT->>BBOX: resolveWheel(5, 10)
    Note over BBOX: ChatList bbox = nil; ConversationViewport<br/>covers full width in Compact → maybe matches.<br/>If (5, 10) ∈ ConversationViewport bbox → forward to VP.<br/>If not → no-op.
    BBOX-->>ROUT: ConversationViewport (full-width in Compact)
    ROUT->>VP: viewport.Update(MouseMsg{WheelDown})
```

**Punto chiave**: in Compact il pannello visibile occupa tutta la
larghezza. Il wheel "ovunque sopra il pannello" scrolla il pannello.
Il pannello hidden ha bbox `nil` ⇒ non riceve mai eventi
(NO_HIDDEN_CLICK è una formulazione click-centric ma vale anche per
wheel: NO_HIDDEN_WHEEL implicito).

## 9. Click in regione hidden in Compact (no-op)

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant V as View

    Note over APP: state: layoutMode=Compact, compactVisible=ChatList,<br/>bboxes: ChatList valid, Conversation=nil (hidden)<br/>(any "Conversation area" coords are not in any bbox)

    U->>TEA: left click at (X=10, Y=999) — out of bounds
    TEA->>APP: tea.MouseMsg{X:10, Y:999, Action:Press, Button:Left}
    APP->>ROUT: dispatch
    ROUT->>ROUT: ClickDispatch → ResolvingWidget
    ROUT->>BBOX: resolveClick(10, 999)
    BBOX-->>ROUT: NoOpClick (no bbox contains (10, 999))
    ROUT-->>APP: (no msg)
    APP->>V: re-render unchanged
    V-->>U: nothing happens
    Note over APP: NO_HIDDEN_CLICK invariant respected;<br/>BBOX_TOTAL: resolveClick returns exactly NoOpClick (one terminal)
```

**Punto chiave**: clicks fuori bbox o su pannelli hidden in Compact =
no-op deterministico (BBOX_TOTAL ⇒ exactly one terminal value;
NO_HIDDEN_CLICK ⇒ hidden bbox is `nil` ⇒ skipped).

## 10. (Bonus) Click su textarea da panel non-focused

```mermaid
sequenceDiagram
    participant U as User
    participant TEA as bubbletea runtime
    participant APP as MainModel.Update
    participant ROUT as MainModel.handleMouseMsg
    participant BBOX as bboxes cache
    participant CONV as ConversationModel
    participant V as View

    Note over APP: state: layoutMode=Wide, activePanel=ChatList,<br/>inputFocus=false, sendBtn.Active=false,<br/>activeChatID="John" (conversation visible)

    U->>TEA: left click at (X=70, Y=22) — over textarea (not SEND)
    TEA->>APP: tea.MouseMsg{X:70, Y:22, Action:Press, Button:Left}
    APP->>ROUT: dispatch
    ROUT->>BBOX: resolveClick(70, 22)
    Note over BBOX: z-order: SendButton tested first → no match<br/>(70 < SendButton.x0); InputArea matches
    BBOX-->>ROUT: Resolved{InputArea}
    Note over ROUT: D10 + D6 atomic:<br/>activePanel := Conversation (focus shift)<br/>inputFocus := true<br/>sendBtn.Active := true (textarea is focused now)
    ROUT->>CONV: textarea.Focus() [via internal call]
    ROUT-->>APP: (textarea now focused; no Telegram-side Cmd)
    APP->>V: re-render: chatList loses focus border;<br/>textarea gains caret + active style;<br/>sendBtn renders Active style
    V-->>U: caret appears in textarea, ready to type
    Note over APP: KEYBOARD_PARITY: equivalent to Tab→Tab + 'i'<br/>(keyboard sequence to focus input)
```

**Punto chiave**: click su textarea = focus shift, non submit. Distinto
da click su SEND (sub-rect a destra). D10 + D6 applicati atomicamente.

## Riepilogo invarianti runtime

| Invariante | Esempio scenario |
|------------|------------------|
| `KEYBOARD_PARITY` (mouse = stesso effetto di una sequenza keyboard) | Scenari 1-10 (tutti) |
| `BBOX_TOTAL` (resolveClick → ≤1 widget) | Scenari 2, 3, 9, 10 |
| `NO_HIDDEN_CLICK` (click su panel hidden in Compact = no-op) | Scenari 8, 9 |
| `OVERLAY_FIRST` (overlay vince hit-test) | Scenari 5, 6 |
| `WHEEL_BY_POSITION` (wheel routes by cursor, not focus) | Scenari 1, 7, 8 |
| `CLICK_FOCUS_SHIFT` (focus shift atomic con azione) | Scenari 2, 4, 10 |
| `NO_PHANTOM_DRAG` (Motion/Release scartati) | (implicito; nessun drag mostrato) |
| `DISMISSABLE_OUTSIDE_CLOSES` (click outside dismissable = close) | Scenario 5 |
| Modal overlay no-op outside | Scenario 6 |
| `SENDBUTTON_INACTIVE_NO_OP` (click su SEND vuoto = no-op) | Scenario 3 (variante) |
| `MOUSE_NEVER_FLIPS_LAYOUTMODE` | Scenario 4 (Compact resta Compact) |
| `MOUSE_NEVER_OPENS_OVERLAY` | (implicito; nessun mouse-trigger overlay-open in Step 32) |

## Cross-links

- Statechart: [`../phase-2-behavioral/mouse-routing.md`](../phase-2-behavioral/mouse-routing.md)
- ADR (decisioni D1..D10 + skip TLA+): [ADR-020](../phase-6-decisions/ADR-020-mouse-support.md)
- Pipeline step: [`../development-pipeline.md` §Step 32](../development-pipeline.md)
- Ereditato da:
  - [ADR-015 §D3](../phase-6-decisions/ADR-015-command-palette-whichkey-help.md) — overlay mutex
  - [ADR-016 §D5](../phase-6-decisions/ADR-016-folder-source-and-filtering.md) — sidebar in compact
  - [ADR-018 §D2/§D4](../phase-6-decisions/ADR-018-responsive-layout-threshold-and-tab.md) — `COMPACT_ONE_PANEL`, cross-threshold trigger di bbox invalidation
  - [ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md) — forwardPicker modal (rispettato da scenario 6)
- Sequence reference parente (forma): [`responsive-layout-flow.md`](responsive-layout-flow.md)
- Tui design canonical: [`../tui-design.md`](../tui-design.md) §"Mouse Support"
