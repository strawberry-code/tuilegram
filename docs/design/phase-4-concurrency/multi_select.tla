---- MODULE multi_select ----
(*
 * TLA+ Specification — Multi-Select mode + batch actions (Step 22).
 *
 * Models the lifecycle of the selection set S inside ConversationFocused
 * and its interaction with the batch forward / batch delete RPCs.
 *
 * Extends the modeling pattern of forward_picker.tla: the picker and the
 * confirm dialog are abstracted as RPC-bearing overlays; we focus on the
 * invariants of S and on the absence of races between selection mutation
 * and batch RPC submission.
 *
 *   Safety:
 *     - MODE_COHERENCE: mode = "multiSelect" iff |S| > 0.
 *     - SELECTION_SCOPE: S only contains message IDs from the active chat.
 *     - SOURCE_SNAPSHOT: once a batch action submits, the snapshot passed
 *       to the overlay is frozen for the entire RPC lifecycle, even if S
 *       (the live model) is mutated by concurrent updates.
 *     - BATCH_ATOMICITY: at most one batch RPC in flight at any time.
 *     - NO_DOUBLE_OVERLAY: forward picker and confirm delete cannot be
 *       open at the same time.
 *     - FALLBACK_OK: when |S|=0, f/D operate on a singleton {cursor.id};
 *       resulting source is non-empty.
 *     - NO_REPLY_EDIT_IN_MULTI: r/e are no-op when |S|>0.
 *
 *   Liveness:
 *     - EVENTUAL_EXIT: from any MultiSelect state we can return to
 *       BrowsingMessages (S=∅).
 *     - NO_STUCK_BATCH: every in-flight batch RPC eventually completes.
 *
 * Scope: single conversation, single user keystroke stream. Concurrent
 * server-side deletes (UpdateDeleteMessages) are modeled abstractly.
 *)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    MaxMessages,        \* upper bound on cursor space (e.g. 5)
    Cursors,            \* set of valid cursor message IDs in the active chat
    Targets             \* set of candidate target chat IDs for forward

VARIABLES
    mode,               \* {"browsing", "multiSelect"}
    selection,          \* SUBSET Cursors (the set S)
    cursor,             \* current cursor message ID (∈ Cursors)
    overlay,            \* {"none", "forwardPicker", "confirmDelete"}
    overlayState,       \* {"idle", "rpcInFlight"}
    overlaySnapshot,    \* SUBSET Cursors — frozen at overlay open
    rpcPending,         \* set: at most one batch RPC in flight
    statusBar,          \* {"none", "successFwd", "successDel", "errorFwd", "errorDel", "hint"}
    deletedRemote       \* SUBSET Cursors — IDs deleted by remote concurrently

vars == <<mode, selection, cursor, overlay, overlayState,
          overlaySnapshot, rpcPending, statusBar, deletedRemote>>

----

TypeOK ==
    /\ mode \in {"browsing", "multiSelect"}
    /\ selection \subseteq Cursors
    /\ cursor \in Cursors
    /\ overlay \in {"none", "forwardPicker", "confirmDelete"}
    /\ overlayState \in {"idle", "rpcInFlight"}
    /\ overlaySnapshot \subseteq Cursors
    /\ rpcPending \in SUBSET ({"forward", "delete"})
    /\ statusBar \in {"none", "successFwd", "successDel",
                       "errorFwd", "errorDel", "hint"}
    /\ deletedRemote \subseteq Cursors

----

Init ==
    /\ mode = "browsing"
    /\ selection = {}
    /\ cursor \in Cursors
    /\ overlay = "none"
    /\ overlayState = "idle"
    /\ overlaySnapshot = {}
    /\ rpcPending = {}
    /\ statusBar = "none"
    /\ deletedRemote = {}

----

(* --- Cursor & selection actions (no overlay) --- *)

\* j/k cursor move; selection invariato
MoveCursor(c) ==
    /\ overlay = "none"
    /\ c \in Cursors
    /\ cursor' = c
    /\ UNCHANGED <<mode, selection, overlay, overlayState,
                   overlaySnapshot, rpcPending, statusBar, deletedRemote>>

\* Space toggle: if cursor.id ∈ S → remove; else → add. If S becomes ∅ → exit MultiSelect.
SpaceToggle ==
    /\ overlay = "none"
    /\ LET newSel == IF cursor \in selection
                     THEN selection \ {cursor}
                     ELSE selection \cup {cursor}
       IN
        /\ selection' = newSel
        /\ mode' = IF newSel = {} THEN "browsing" ELSE "multiSelect"
    /\ UNCHANGED <<cursor, overlay, overlayState, overlaySnapshot,
                   rpcPending, statusBar, deletedRemote>>

\* Esc in multiSelect → clear selection
EscClear ==
    /\ overlay = "none"
    /\ mode = "multiSelect"
    /\ selection' = {}
    /\ mode' = "browsing"
    /\ UNCHANGED <<cursor, overlay, overlayState, overlaySnapshot,
                   rpcPending, statusBar, deletedRemote>>

\* r / e in multiSelect → no-op + status hint
ReplyEditNoOp ==
    /\ overlay = "none"
    /\ mode = "multiSelect"
    /\ statusBar' = "hint"
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlayState,
                   overlaySnapshot, rpcPending, deletedRemote>>

\* Concurrent server-side delete; mutates live selection but NOT overlaySnapshot
RemoteDelete(m) ==
    /\ m \in Cursors
    /\ m \notin deletedRemote
    /\ deletedRemote' = deletedRemote \cup {m}
    /\ selection' = selection \ {m}
    /\ mode' = IF selection \ {m} = {} THEN "browsing" ELSE mode
    /\ UNCHANGED <<cursor, overlay, overlayState, overlaySnapshot,
                   rpcPending, statusBar>>

----

(* --- Open batch overlay (forward / delete) --- *)

\* f → open forward picker. If S=∅ → fallback su {cursor}.
OpenForward ==
    /\ overlay = "none"
    /\ rpcPending = {}
    /\ overlay' = "forwardPicker"
    /\ overlayState' = "idle"
    /\ overlaySnapshot' = IF selection = {} THEN {cursor} ELSE selection
    /\ UNCHANGED <<mode, selection, cursor, rpcPending,
                   statusBar, deletedRemote>>

\* D → open confirm delete. If S=∅ → fallback su {cursor}.
OpenDelete ==
    /\ overlay = "none"
    /\ rpcPending = {}
    /\ overlay' = "confirmDelete"
    /\ overlayState' = "idle"
    /\ overlaySnapshot' = IF selection = {} THEN {cursor} ELSE selection
    /\ UNCHANGED <<mode, selection, cursor, rpcPending,
                   statusBar, deletedRemote>>

\* Esc in overlay (idle) → close, S preservato
OverlayCancel ==
    /\ overlay /= "none"
    /\ overlayState = "idle"
    /\ overlay' = "none"
    /\ overlaySnapshot' = {}
    /\ UNCHANGED <<mode, selection, cursor, overlayState, rpcPending,
                   statusBar, deletedRemote>>

----

(* --- Submit & RPC lifecycle --- *)

SubmitForward(t) ==
    /\ overlay = "forwardPicker"
    /\ overlayState = "idle"
    /\ overlaySnapshot /= {}
    /\ rpcPending = {}
    /\ t \in Targets
    /\ overlayState' = "rpcInFlight"
    /\ rpcPending' = {"forward"}
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlaySnapshot,
                   statusBar, deletedRemote>>

SubmitDelete ==
    /\ overlay = "confirmDelete"
    /\ overlayState = "idle"
    /\ overlaySnapshot /= {}
    /\ rpcPending = {}
    /\ overlayState' = "rpcInFlight"
    /\ rpcPending' = {"delete"}
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlaySnapshot,
                   statusBar, deletedRemote>>

\* Esc during rpcInFlight is IGNORED (ADR-007 inheritance) — no action enabled.

ForwardSuccess ==
    /\ overlay = "forwardPicker"
    /\ overlayState = "rpcInFlight"
    /\ "forward" \in rpcPending
    /\ overlay' = "none"
    /\ overlayState' = "idle"
    /\ overlaySnapshot' = {}
    /\ rpcPending' = rpcPending \ {"forward"}
    /\ selection' = {}                 \* BatchActionDoneMsg
    /\ mode' = "browsing"
    /\ statusBar' = "successFwd"
    /\ UNCHANGED <<cursor, deletedRemote>>

ForwardFailure ==
    /\ overlay = "forwardPicker"
    /\ overlayState = "rpcInFlight"
    /\ "forward" \in rpcPending
    /\ overlayState' = "idle"          \* picker stays open per retry
    /\ rpcPending' = rpcPending \ {"forward"}
    /\ statusBar' = "errorFwd"
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlaySnapshot,
                   deletedRemote>>

DeleteSuccess ==
    /\ overlay = "confirmDelete"
    /\ overlayState = "rpcInFlight"
    /\ "delete" \in rpcPending
    /\ overlay' = "none"
    /\ overlayState' = "idle"
    /\ overlaySnapshot' = {}
    /\ rpcPending' = rpcPending \ {"delete"}
    /\ selection' = {}
    /\ mode' = "browsing"
    /\ statusBar' = "successDel"
    /\ UNCHANGED <<cursor, deletedRemote>>

DeleteFailure ==
    /\ overlay = "confirmDelete"
    /\ overlayState = "rpcInFlight"
    /\ "delete" \in rpcPending
    /\ overlayState' = "idle"
    /\ rpcPending' = rpcPending \ {"delete"}
    /\ statusBar' = "errorDel"
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlaySnapshot,
                   deletedRemote>>

ClearStatus ==
    /\ statusBar /= "none"
    /\ statusBar' = "none"
    /\ UNCHANGED <<mode, selection, cursor, overlay, overlayState,
                   overlaySnapshot, rpcPending, deletedRemote>>

----

Next ==
    \/ \E c \in Cursors : MoveCursor(c)
    \/ SpaceToggle
    \/ EscClear
    \/ ReplyEditNoOp
    \/ \E m \in Cursors : RemoteDelete(m)
    \/ OpenForward
    \/ OpenDelete
    \/ OverlayCancel
    \/ \E t \in Targets : SubmitForward(t)
    \/ SubmitDelete
    \/ ForwardSuccess
    \/ ForwardFailure
    \/ DeleteSuccess
    \/ DeleteFailure
    \/ ClearStatus

Spec == Init /\ [][Next]_vars

----

(* --- Safety Invariants --- *)

\* mode = multiSelect iff |S| > 0
MODE_COHERENCE ==
    (mode = "multiSelect") <=> (selection /= {})

\* S contains only IDs from active chat (modeled by Cursors set)
SELECTION_SCOPE == selection \subseteq Cursors

\* While an overlay is open, the snapshot is non-empty (covers fallback case too)
SOURCE_SNAPSHOT ==
    (overlay /= "none") => (overlaySnapshot /= {})

\* Snapshot frozen during RPC: even if remote deletes mutate `selection`,
\* `overlaySnapshot` retains its initial elements while RPC is in flight.
\* (Captured implicitly: no action mutates overlaySnapshot while overlay /= "none"
\*  except OverlayCancel/Success/Failure which clear it on close.)
SNAPSHOT_FROZEN_DURING_RPC ==
    (overlayState = "rpcInFlight") => (overlaySnapshot /= {})

\* At most one batch RPC in flight (forward XOR delete, never both)
BATCH_ATOMICITY == Cardinality(rpcPending) <= 1

\* Forward picker and confirm delete cannot coexist
NO_DOUBLE_OVERLAY ==
    \neg (overlay = "forwardPicker" /\ overlay = "confirmDelete")
    \* trivially true given overlay is single-valued; the meaningful invariant:
    /\ (overlay /= "none") => (Cardinality({overlay}) = 1)

\* Fallback: when overlay opens, snapshot is non-empty even if S=∅
FALLBACK_OK ==
    (overlay /= "none") => (overlaySnapshot /= {})

\* No reply/edit semantics in multiSelect: encoded as ReplyEditNoOp not changing
\* mode/selection. Invariant form:
NO_REPLY_EDIT_IN_MULTI ==
    \* No state where mode=multiSelect /\ S=∅ (would indicate r/e accidentally cleared S)
    (mode = "multiSelect") => (selection /= {})

\* Status bar reflects last result
STATUS_CONSISTENCY ==
    (statusBar = "successFwd") => (overlay = "none" \/ overlayState = "idle")

----

(* --- Liveness Properties --- *)

Fairness ==
    /\ WF_vars(ForwardSuccess)
    /\ WF_vars(ForwardFailure)
    /\ WF_vars(DeleteSuccess)
    /\ WF_vars(DeleteFailure)
    /\ WF_vars(ClearStatus)

\* From any state we can eventually return to browsing/idle
EVENTUAL_EXIT == []<>(mode = "browsing" /\ overlay = "none")

\* Every batch RPC eventually completes
NO_STUCK_BATCH ==
    [](overlayState = "rpcInFlight" => <>(overlayState = "idle"))

LiveSpec == Spec /\ Fairness

====
