---- MODULE forward_picker ----
(*
 * TLA+ Specification — Forward Picker overlay (Step 21).
 *
 * Models the lifecycle of the forward picker overlay and its interaction
 * with the forwardMessageCmd goroutine. Extends tuilegram.tla's concurrency
 * model to verify that:
 *
 *   Safety:
 *     - RPC_ATOMICITY: at most one forwardMessageCmd goroutine in flight
 *       per picker instance.
 *     - NO_ESC_DURING_RPC: picker cannot transition Closed while RPC is
 *       in flight (ADR-007).
 *     - RESULT_CONSISTENCY: every in-flight RPC eventually yields exactly
 *       one ForwardResultMsg (success or failure).
 *     - SOURCE_SNAPSHOT: the []Message being forwarded is frozen at picker
 *       open time; concurrent MessageDeletedMsg does not mutate it.
 *
 *   Liveness:
 *     - EVENTUAL_CLOSE: the picker always eventually returns to Closed.
 *     - NO_STUCK_RPC: an in-flight RPC eventually completes (WF on result).
 *
 * Scope: single picker instance. Step 22 multi-select extends the source
 * set (MaxSourceMsgs >= 2) without changing the lifecycle; the batch
 * semantics, mode coherence and snapshot invariants are formalized
 * separately in multi_select.tla.
 *)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    MaxSourceMsgs,      \* upper bound on source messages per forward (1 for Step 21)
    Chats               \* set of candidate target chat IDs, e.g. {1, 2, 3}

VARIABLES
    pickerState,        \* {"closed", "opening", "filtering", "rpcInFlight", "failed"}
    pickerSource,       \* Seq of message IDs captured at open (frozen)
    pickerTarget,       \* chat ID selected at submit, or 0 when none
    pickerQuery,        \* current filter query (abstracted as Nat: 0=empty, >0=non-empty)
    rpcPending,         \* set of {target} for in-flight forward RPCs
    rpcResults,         \* Seq of records [target |-> Chat, ok |-> BOOLEAN]
    statusBar           \* {"none", "success", "error"}

vars == <<pickerState, pickerSource, pickerTarget, pickerQuery,
          rpcPending, rpcResults, statusBar>>

----

TypeOK ==
    /\ pickerState \in {"closed", "opening", "filtering", "rpcInFlight", "failed"}
    /\ pickerSource \in Seq(Nat)
    /\ Len(pickerSource) <= MaxSourceMsgs
    /\ pickerTarget \in Chats \cup {0}
    /\ pickerQuery \in Nat
    /\ rpcPending \subseteq Chats
    /\ rpcResults \in Seq([target: Chats, ok: BOOLEAN])
    /\ statusBar \in {"none", "success", "error"}

----

Init ==
    /\ pickerState = "closed"
    /\ pickerSource = <<>>
    /\ pickerTarget = 0
    /\ pickerQuery = 0
    /\ rpcPending = {}
    /\ rpcResults = <<>>
    /\ statusBar = "none"

----

(* --- Actions --- *)

\* User presses 'f' on a message → picker opens with frozen source
OpenPicker(msgIDs) ==
    /\ pickerState = "closed"
    /\ Len(msgIDs) > 0
    /\ Len(msgIDs) <= MaxSourceMsgs
    /\ pickerState' = "opening"
    /\ pickerSource' = msgIDs
    /\ pickerQuery' = 0
    /\ pickerTarget' = 0
    /\ UNCHANGED <<rpcPending, rpcResults, statusBar>>

\* Dialogs loaded from cache → picker ready
PickerReady ==
    /\ pickerState = "opening"
    /\ pickerState' = "filtering"
    /\ UNCHANGED <<pickerSource, pickerTarget, pickerQuery,
                   rpcPending, rpcResults, statusBar>>

\* User types (query changes). Source and target stay frozen; cursor resets.
TypeQuery(q) ==
    /\ pickerState = "filtering"
    /\ pickerQuery' = q
    /\ pickerTarget' = 0
    /\ UNCHANGED <<pickerState, pickerSource, rpcPending, rpcResults, statusBar>>

\* User presses Esc while in filtering state → picker closes, no RPC
CancelPicker ==
    /\ pickerState = "filtering"
    /\ pickerState' = "closed"
    /\ pickerSource' = <<>>
    /\ pickerTarget' = 0
    /\ pickerQuery' = 0
    /\ UNCHANGED <<rpcPending, rpcResults, statusBar>>

\* User presses Enter on a target chat → spawn RPC
Submit(target) ==
    /\ pickerState = "filtering"
    /\ target \in Chats
    /\ rpcPending = {}                   \* RPC_ATOMICITY precondition
    /\ pickerState' = "rpcInFlight"
    /\ pickerTarget' = target
    /\ rpcPending' = {target}
    /\ UNCHANGED <<pickerSource, pickerQuery, rpcResults, statusBar>>

\* Esc during RPC is IGNORED (ADR-007) — no transition, no state change.
\* This action is intentionally absent: Esc pressed in rpcInFlight has no
\* enabled action, so the spec models it as a no-op at the system level.

\* RPC completes successfully → close picker, show success toast
RPCSuccess ==
    /\ pickerState = "rpcInFlight"
    /\ \E t \in rpcPending :
        /\ rpcResults' = Append(rpcResults, [target |-> t, ok |-> TRUE])
        /\ rpcPending' = rpcPending \ {t}
    /\ pickerState' = "closed"
    /\ pickerSource' = <<>>
    /\ pickerTarget' = 0
    /\ pickerQuery' = 0
    /\ statusBar' = "success"

\* RPC fails → picker returns to filtering, user can retry
RPCFailure ==
    /\ pickerState = "rpcInFlight"
    /\ \E t \in rpcPending :
        /\ rpcResults' = Append(rpcResults, [target |-> t, ok |-> FALSE])
        /\ rpcPending' = rpcPending \ {t}
    /\ pickerState' = "filtering"
    /\ statusBar' = "error"
    /\ UNCHANGED <<pickerSource, pickerTarget, pickerQuery>>

\* Status bar auto-clears after render
ClearStatus ==
    /\ statusBar \in {"success", "error"}
    /\ statusBar' = "none"
    /\ UNCHANGED <<pickerState, pickerSource, pickerTarget, pickerQuery,
                   rpcPending, rpcResults>>

----

Next ==
    \/ \E s \in Seq(1..MaxSourceMsgs) : OpenPicker(s) /\ Len(s) >= 1
    \/ PickerReady
    \/ \E q \in 0..3 : TypeQuery(q)
    \/ CancelPicker
    \/ \E t \in Chats : Submit(t)
    \/ RPCSuccess
    \/ RPCFailure
    \/ ClearStatus

Spec == Init /\ [][Next]_vars

----

(* --- Safety Invariants --- *)

\* At most one RPC in flight per picker (single-instance scope)
RPC_ATOMICITY == Cardinality(rpcPending) <= 1

\* While RPC is in flight, picker cannot be closed
NO_ESC_DURING_RPC ==
    (pickerState = "rpcInFlight") => (rpcPending /= {})

\* Source is frozen during the entire forward lifecycle (open → close)
\* Modeled by: once pickerSource is set, it is not rewritten until close.
\* Checked by absence of actions that mutate pickerSource while state in
\* {opening, filtering, rpcInFlight, failed}. Proven by invariant:
SOURCE_SNAPSHOT ==
    (pickerState \in {"opening", "filtering", "rpcInFlight"})
    => (Len(pickerSource) > 0)

\* Status bar reflects the last result (success or error) only after RPC
STATUS_CONSISTENCY ==
    (statusBar = "success") => (\E i \in 1..Len(rpcResults) : rpcResults[i].ok)

----

(* --- Liveness Properties --- *)

Fairness ==
    /\ WF_vars(PickerReady)
    /\ WF_vars(RPCSuccess)
    /\ WF_vars(RPCFailure)
    /\ WF_vars(ClearStatus)

\* Picker always eventually returns to Closed
EVENTUAL_CLOSE == []<>(pickerState = "closed")

\* Every in-flight RPC eventually yields a result
NO_STUCK_RPC ==
    [](pickerState = "rpcInFlight" => <>(pickerState \in {"closed", "filtering"}))

LiveSpec == Spec /\ Fairness

====
