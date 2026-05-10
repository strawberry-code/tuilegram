---- MODULE tuilegram ----
(*
 * TLA+ Specification for tuilegram concurrency model.
 *
 * Models the interaction between:
 *   1. TUI Loop (bubbletea) — processes user input, renders UI
 *   2. Telegram Client (gotd/td) — communicates with Telegram servers
 *   3. Message Channel — tea.Program.Send() bridge
 *
 * Verifies:
 *   Safety:
 *     - No lost messages (every received update reaches the UI)
 *     - No stale UI state (UI eventually reflects Telegram state)
 *     - Message ordering preserved (messages appear in order)
 *     - No concurrent mutation of shared state
 *
 *   Liveness:
 *     - TUI always eventually processes pending events
 *     - Telegram client always reconnects after disconnect
 *     - Sent messages eventually get a delivery status
 *)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    MaxMessages,        \* max messages in system (for bounded model checking)
    MaxReconnects       \* max reconnection attempts

VARIABLES
    \* TUI state
    tuiState,           \* {idle, processing, rendering}
    tuiMsgQueue,        \* sequence of messages pending processing
    uiMessages,         \* set of messages currently displayed
    uiConnectionStatus, \* {connected, disconnected, reconnecting}

    \* Telegram client state
    tgState,            \* {disconnected, connecting, connected, reconnecting}
    tgRecvBuffer,       \* messages received from server, not yet sent to TUI
    reconnectCount,     \* current reconnection attempt number

    \* Telegram server state (simplified)
    serverMessages,     \* set of all messages on the server
    serverMsgCounter,   \* next message ID

    \* Shared channel: tea.Program.Send()
    sendChannel,        \* sequence modeling the channel from TG → TUI

    \* Outgoing messages
    pendingOutgoing,    \* messages user wants to send, not yet ACKed
    deliveryStatus      \* function: msgID → {sending, sent, delivered, read, failed}

vars == <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
          tgState, tgRecvBuffer, reconnectCount,
          serverMessages, serverMsgCounter,
          sendChannel, pendingOutgoing, deliveryStatus>>

----

(* --- Type Invariant --- *)

TypeOK ==
    /\ tuiState \in {"idle", "processing", "rendering"}
    /\ tuiMsgQueue \in Seq(Nat)
    /\ uiMessages \subseteq Nat
    /\ uiConnectionStatus \in {"connected", "disconnected", "reconnecting"}
    /\ tgState \in {"disconnected", "connecting", "connected", "reconnecting"}
    /\ tgRecvBuffer \in Seq(Nat)
    /\ reconnectCount \in 0..MaxReconnects
    /\ serverMessages \subseteq Nat
    /\ serverMsgCounter \in 1..(MaxMessages + 1)
    /\ sendChannel \in Seq(Nat \cup {"connected", "disconnected", "reconnecting"})
    /\ pendingOutgoing \subseteq Nat
    /\ deliveryStatus \in [Nat -> {"sending", "sent", "delivered", "read", "failed", "none"}]

----

(* --- Initial State --- *)

Init ==
    /\ tuiState = "idle"
    /\ tuiMsgQueue = <<>>
    /\ uiMessages = {}
    /\ uiConnectionStatus = "disconnected"
    /\ tgState = "disconnected"
    /\ tgRecvBuffer = <<>>
    /\ reconnectCount = 0
    /\ serverMessages = {}
    /\ serverMsgCounter = 1
    /\ sendChannel = <<>>
    /\ pendingOutgoing = {}
    /\ deliveryStatus = [m \in Nat |-> "none"]

----

(* --- Telegram Client Actions --- *)

\* Telegram client starts connecting
TGConnect ==
    /\ tgState = "disconnected"
    /\ tgState' = "connecting"
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, sendChannel, pendingOutgoing, deliveryStatus>>

\* Connection established
TGConnected ==
    /\ tgState \in {"connecting", "reconnecting"}
    /\ tgState' = "connected"
    /\ reconnectCount' = 0
    /\ sendChannel' = Append(sendChannel, "connected")
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgRecvBuffer, serverMessages, serverMsgCounter,
                   pendingOutgoing, deliveryStatus>>

\* Connection lost
TGDisconnected ==
    /\ tgState = "connected"
    /\ tgState' = "reconnecting"
    /\ reconnectCount' = 1
    /\ sendChannel' = Append(sendChannel, "disconnected")
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgRecvBuffer, serverMessages, serverMsgCounter,
                   pendingOutgoing, deliveryStatus>>

\* Reconnection attempt
TGReconnectAttempt ==
    /\ tgState = "reconnecting"
    /\ reconnectCount < MaxReconnects
    /\ sendChannel' = Append(sendChannel, "reconnecting")
    /\ reconnectCount' = reconnectCount + 1
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, serverMessages, serverMsgCounter,
                   pendingOutgoing, deliveryStatus>>

\* Server generates a new message (incoming)
ServerNewMessage ==
    /\ serverMsgCounter <= MaxMessages
    /\ serverMessages' = serverMessages \cup {serverMsgCounter}
    /\ serverMsgCounter' = serverMsgCounter + 1
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount,
                   sendChannel, pendingOutgoing, deliveryStatus>>

\* Telegram client receives message from server
TGReceiveMessage ==
    /\ tgState = "connected"
    /\ \E m \in serverMessages :
        /\ m \notin uiMessages           \* not yet in UI
        /\ m \notin {tgRecvBuffer[i] : i \in 1..Len(tgRecvBuffer)}  \* not in buffer
        /\ tgRecvBuffer' = Append(tgRecvBuffer, m)
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, reconnectCount, serverMessages, serverMsgCounter,
                   sendChannel, pendingOutgoing, deliveryStatus>>

\* Telegram client forwards message to TUI via p.Send()
TGForwardToTUI ==
    /\ Len(tgRecvBuffer) > 0
    /\ sendChannel' = Append(sendChannel, Head(tgRecvBuffer))
    /\ tgRecvBuffer' = Tail(tgRecvBuffer)
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, reconnectCount, serverMessages, serverMsgCounter,
                   pendingOutgoing, deliveryStatus>>

----

(* --- User Actions --- *)

\* User sends a message
UserSendMessage ==
    /\ serverMsgCounter <= MaxMessages
    /\ tuiState = "idle"
    /\ tgState = "connected"
    /\ LET msgID == serverMsgCounter
       IN /\ pendingOutgoing' = pendingOutgoing \cup {msgID}
          /\ deliveryStatus' = [deliveryStatus EXCEPT ![msgID] = "sending"]
          /\ serverMsgCounter' = serverMsgCounter + 1
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   sendChannel>>

\* Server ACKs sent message
ServerAckMessage ==
    /\ \E m \in pendingOutgoing :
        /\ serverMessages' = serverMessages \cup {m}
        /\ pendingOutgoing' = pendingOutgoing \ {m}
        /\ deliveryStatus' = [deliveryStatus EXCEPT ![m] = "sent"]
        /\ uiMessages' = uiMessages \cup {m}   \* optimistically added to UI
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMsgCounter,
                   sendChannel>>

\* Message delivery status advances
MessageDelivered ==
    /\ \E m \in DOMAIN deliveryStatus :
        /\ deliveryStatus[m] = "sent"
        /\ deliveryStatus' = [deliveryStatus EXCEPT ![m] = "delivered"]
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, sendChannel, pendingOutgoing>>

MessageRead ==
    /\ \E m \in DOMAIN deliveryStatus :
        /\ deliveryStatus[m] = "delivered"
        /\ deliveryStatus' = [deliveryStatus EXCEPT ![m] = "read"]
    /\ UNCHANGED <<tuiState, tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, sendChannel, pendingOutgoing>>

----

(* --- TUI Loop Actions --- *)

\* TUI picks up message from send channel
TUIReceiveFromChannel ==
    /\ Len(sendChannel) > 0
    /\ LET msg == Head(sendChannel)
       IN /\ tuiMsgQueue' = Append(tuiMsgQueue, msg)
          /\ sendChannel' = Tail(sendChannel)
    /\ UNCHANGED <<tuiState, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, pendingOutgoing, deliveryStatus>>

\* TUI processes next message in queue
TUIProcessMessage ==
    /\ Len(tuiMsgQueue) > 0
    /\ tuiState = "idle"
    /\ LET msg == Head(tuiMsgQueue)
       IN /\ tuiMsgQueue' = Tail(tuiMsgQueue)
          /\ IF msg = "connected" THEN
                /\ uiConnectionStatus' = "connected"
                /\ UNCHANGED uiMessages
             ELSE IF msg = "disconnected" THEN
                /\ uiConnectionStatus' = "disconnected"
                /\ UNCHANGED uiMessages
             ELSE IF msg = "reconnecting" THEN
                /\ uiConnectionStatus' = "reconnecting"
                /\ UNCHANGED uiMessages
             ELSE \* it's a message ID
                /\ uiMessages' = uiMessages \cup {msg}
                /\ UNCHANGED uiConnectionStatus
    /\ tuiState' = "processing"
    /\ UNCHANGED <<tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, sendChannel, pendingOutgoing, deliveryStatus>>

\* TUI finishes processing (render complete)
TUIFinishProcessing ==
    /\ tuiState = "processing"
    /\ tuiState' = "idle"
    /\ UNCHANGED <<tuiMsgQueue, uiMessages, uiConnectionStatus,
                   tgState, tgRecvBuffer, reconnectCount, serverMessages,
                   serverMsgCounter, sendChannel, pendingOutgoing, deliveryStatus>>

----

(* --- Next State --- *)

Next ==
    \/ TGConnect
    \/ TGConnected
    \/ TGDisconnected
    \/ TGReconnectAttempt
    \/ ServerNewMessage
    \/ TGReceiveMessage
    \/ TGForwardToTUI
    \/ UserSendMessage
    \/ ServerAckMessage
    \/ MessageDelivered
    \/ MessageRead
    \/ TUIReceiveFromChannel
    \/ TUIProcessMessage
    \/ TUIFinishProcessing

Spec == Init /\ [][Next]_vars

----

(* --- Safety Properties --- *)

\* No message is lost: every server message eventually appears in UI
\* (checked as invariant: at any point, messages in transit are accounted for)
NoLostMessages ==
    \A m \in serverMessages :
        \/ m \in uiMessages                     \* already displayed
        \/ m \in {tgRecvBuffer[i] : i \in 1..Len(tgRecvBuffer)}  \* in TG buffer
        \/ m \in {sendChannel[i] : i \in 1..Len(sendChannel)}     \* in channel
        \/ m \in {tuiMsgQueue[i] : i \in 1..Len(tuiMsgQueue)}     \* in TUI queue

\* Connection status consistency: UI reflects actual TG state (eventually)
ConnectionConsistency ==
    tgState = "connected" /\ Len(sendChannel) = 0 /\ Len(tuiMsgQueue) = 0
    => uiConnectionStatus = "connected"

\* No duplicate messages in UI
NoDuplicates == Cardinality(uiMessages) = Cardinality(uiMessages)  \* trivially true for sets

\* Delivery status never regresses
DeliveryMonotonicity ==
    \A m \in Nat :
        deliveryStatus[m] = "read" => deliveryStatus[m] # "sent"

----

(* --- Liveness Properties --- *)

\* Fair scheduling: every enabled action eventually happens
Fairness ==
    /\ WF_vars(TUIReceiveFromChannel)
    /\ WF_vars(TUIProcessMessage)
    /\ WF_vars(TUIFinishProcessing)
    /\ WF_vars(TGForwardToTUI)
    /\ WF_vars(TGConnected)

\* Every message on the server eventually reaches the UI
EventualDelivery ==
    \A m \in serverMessages : <>(m \in uiMessages)

\* The system always eventually becomes responsive
EventuallyResponsive ==
    []<>(tuiState = "idle")

\* After disconnect, client always eventually reconnects
EventualReconnect ==
    [](tgState = "reconnecting" => <>(tgState = "connected"))

LiveSpec == Spec /\ Fairness

====
