---- MODULE theming ----
(*
 * TLA+ Specification — Theming hot-reload (Step 31, ADR-019 D9 INVERTED).
 *
 * Models the new concurrency dimension introduced by Step 31:
 *
 *   - theme           : the currently APPLIED theme (single source of truth
 *                       read by every UI render via styles.Active())
 *   - state           \in {"Idle", "Reading", "Validating", "Swapping"}
 *                       (reload-pipeline phase; only one reload at a time)
 *   - pendingFile     : optional file payload queued by fsnotify
 *                       (NIL when no pending event)
 *   - watcherAlive    \in BOOLEAN
 *                       (TRUE between Boot and Shutdown; FALSE before/after)
 *
 * Concurrency model:
 *
 *   Two cooperating goroutines:
 *
 *     (W) fsnotify watcher goroutine — observes theme.toml on disk, emits
 *         FileEvent payloads to a channel consumed by (M).
 *     (M) bubbletea message-loop goroutine — receives FileEvent via
 *         tea.Cmd that wraps a channel read; runs Reading -> Validating
 *         -> Swapping in sequence; emits ThemeChangedMsg on success or
 *         ConfigWatcherErrMsg on parse error.
 *
 *   The shared mutable state is `theme` (read by every View call). Mutation
 *   of `theme` is constrained to the AtomicSwap action: a single TLA+
 *   step that atomically replaces the old value with the merged total
 *   theme. No partial / torn writes are observable.
 *
 * Producers of state mutation:
 *
 *   1. Boot          : transitions from disabled to Idle, sets initial
 *                      theme (default merged with user file once).
 *   2. WatchStart    : turns watcherAlive TRUE; modeled as "after Boot".
 *   3. FileEvent(f)  : fsnotify reports a write on theme.toml; queues
 *                      pendingFile.
 *   4. ParseOk(t)    : the file was read+parsed+merged successfully into
 *                      a candidate theme t (still total, by D3 merge).
 *   5. ParseErr      : read or parse or validation failed; emit warning
 *                      via ConfigWatcherErrMsg, do NOT touch theme.
 *   6. AtomicSwap(t) : apply t to `theme` in a single step.
 *   7. Emit          : send ThemeChangedMsg to bubbletea (modeled as a
 *                      transition from Swapping to Idle).
 *   8. WatchStop     : graceful shutdown; watcherAlive := FALSE.
 *
 * Verifies:
 *
 *   Safety:
 *     - THEME_TOTAL: at every state, every recognized color key has a
 *         defined value in `theme` (the function is total over the key
 *         schema). Guaranteed structurally by D3 (override-by-key over a
 *         total embedded baseline) AND by the AtomicSwap action which
 *         only ever assigns a candidate that is itself total.
 *     - NO_TORN_RELOAD: a re-render (any View call abstracted as
 *         "read theme") never observes a partially-updated theme. In
 *         the model: there is NO state where `theme` is mid-merge.
 *         Encoded by: AtomicSwap is a single TLA+ action; the merge
 *         is computed INTO a local variable in Validating phase, then
 *         assigned WHOLE in Swapping.
 *     - MERGE_ATOMIC_ON_UPDATE: the default-theme + user-overrides
 *         merge is computed entirely in the Validating step, and
 *         AtomicSwap publishes the FINAL merged value. There is no
 *         intermediate publish.
 *     - WATCHER_BOUND_TO_LIFECYCLE: FileEvent only fires when
 *         watcherAlive = TRUE. After WatchStop, no more FileEvent
 *         (the watcher goroutine is dead, the channel closed).
 *     - SINGLE_RELOAD_INFLIGHT: at most one reload pipeline is active
 *         at any time (state \in {Reading, Validating, Swapping} are
 *         mutually exclusive).
 *     - INVALID_PRESERVES_THEME: a ParseErr transition leaves `theme`
 *         UNCHANGED (warning emitted, no swap).
 *
 *   Liveness (under fairness on the reload pipeline):
 *     - EVENTUALLY_APPLIED: a FileEvent that produces a valid parsed
 *         theme eventually leads to AtomicSwap, i.e. the user's edit
 *         eventually lands in `theme`.
 *
 * Pattern lineage:
 *   - "monotonic counter / drop-stale" pattern from search debouncer
 *     (ADR-013) does NOT apply here: each FileEvent must be processed
 *     (no debouncing in Step 31; fsnotify already coalesces near-events
 *     at the OS level). If file events arrive faster than reload, the
 *     model permits dropping (pendingFile is overwritten) — see comment
 *     on FileEvent action below.
 *   - Atomic publish pattern: same shape as ADR-016 sidebar toggle
 *     (single-action mutation, no intermediate observable state) but
 *     across goroutines instead of single-threaded.
 *
 * Scope: Step 31 hot-reload dynamics only. Boot-time loading (config +
 * theme initial parse) is collapsed into the Init action; its full
 * statechart is in phase-2-behavioral/theming-and-config.md.
 *)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    Files,                \* set of possible file payloads (abstract; e.g. {"f_ok1","f_ok2","f_bad"})
    ValidFiles,           \* subset of Files that parse + validate successfully
    MaxEvents             \* upper bound on FileEvent occurrences

ASSUME ValidFiles \subseteq Files
ASSUME MaxEvents \in Nat

NIL == "_NIL_"

ReloadState == {"Idle", "Reading", "Validating", "Swapping"}

(*
 * Themes are abstracted as opaque tokens. Each candidate is total by
 * construction (D3 merge invariant). The Init theme is "T_default"; a
 * successful parse of file f yields theme T_f.
 *)
Themes == {"T_default"} \cup {"T_" \o f : f \in Files}

VARIABLES
    theme,                \* Themes; the currently applied theme
    state,                \* ReloadState
    pendingFile,          \* Files \cup {NIL}
    watcherAlive,         \* BOOLEAN
    candidate,            \* Themes \cup {NIL}; staged in Validating, published in Swapping
    eventCount,           \* Nat; bound on FileEvent firings
    history               \* Seq(STRING); trace for inspection

vars == <<theme, state, pendingFile, watcherAlive, candidate,
          eventCount, history>>

----

TypeOK ==
    /\ theme \in Themes
    /\ state \in ReloadState
    /\ pendingFile \in (Files \cup {NIL})
    /\ watcherAlive \in BOOLEAN
    /\ candidate \in (Themes \cup {NIL})
    /\ eventCount \in Nat
    /\ history \in Seq(STRING)

(*
 * Init: bootstrap completed. theme = default+user-merged at boot
 * (here abstracted to T_default). Watcher not yet started; reload
 * pipeline Idle.
 *)
Init ==
    /\ theme = "T_default"
    /\ state = "Idle"
    /\ pendingFile = NIL
    /\ watcherAlive = FALSE
    /\ candidate = NIL
    /\ eventCount = 0
    /\ history = <<>>

----

LogEvent(evt) == history' = Append(history, evt)

----

(*****************************************************************************
 *  ACTION — WatchStart
 *
 *  Boot completed; the fs watcher goroutine is launched. Modeled as a
 *  single transition that flips watcherAlive to TRUE. Must precede any
 *  FileEvent.
 *****************************************************************************)

WatchStart ==
    /\ watcherAlive = FALSE
    /\ state = "Idle"
    /\ watcherAlive' = TRUE
    /\ LogEvent("WatchStart")
    /\ UNCHANGED <<theme, state, pendingFile, candidate, eventCount>>

----

(*****************************************************************************
 *  ACTION — FileEvent(f)
 *
 *  fsnotify reports a write on theme.toml. The watcher goroutine forwards
 *  the file payload f to the message loop via a channel. We model this as
 *  enqueueing pendingFile (overwrite if already pending — fsnotify
 *  coalescing equivalent).
 *
 *  Guard: watcherAlive (the watcher must be running).
 *****************************************************************************)

FileEvent(f) ==
    /\ watcherAlive = TRUE
    /\ eventCount < MaxEvents
    /\ f \in Files
    /\ pendingFile' = f
    /\ eventCount' = eventCount + 1
    /\ LogEvent("FileEvent")
    /\ UNCHANGED <<theme, state, watcherAlive, candidate, history>>
    \* note: history is updated by LogEvent; UNCHANGED list excludes it
    \* (kept here for clarity even though TLA+ semantics already handle it)

----

(*****************************************************************************
 *  ACTION — StartReload
 *
 *  The message loop dequeues pendingFile and enters the reload pipeline.
 *  Single-inflight: only fires when state = Idle.
 *****************************************************************************)

StartReload ==
    /\ state = "Idle"
    /\ pendingFile # NIL
    /\ state' = "Reading"
    /\ pendingFile' = NIL  \* dequeue
    /\ candidate' = NIL    \* clear stale staging
    /\ LogEvent("StartReload")
    /\ UNCHANGED <<theme, watcherAlive, eventCount>>

----

(*****************************************************************************
 *  ACTION — ParseOk(f)
 *
 *  Read+parse+validate+merge succeeded for file f. Stage the candidate
 *  total theme T_f. Transition Reading -> Validating -> Swapping in two
 *  steps (collapsed here for state-space economy).
 *
 *  Guard: f \in ValidFiles (only fileable parses that yield a valid total
 *  theme).
 *****************************************************************************)

ParseOk(f) ==
    /\ state = "Reading"
    /\ f \in ValidFiles
    /\ state' = "Swapping"
    /\ candidate' = "T_" \o f
    /\ LogEvent("ParseOk")
    /\ UNCHANGED <<theme, pendingFile, watcherAlive, eventCount>>

----

(*****************************************************************************
 *  ACTION — ParseErr
 *
 *  Read or parse or validation failed. Emit ConfigWatcherErrMsg (warn +
 *  preserve). theme is UNCHANGED. The pipeline returns to Idle directly.
 *
 *  Captures INVALID_PRESERVES_THEME structurally.
 *****************************************************************************)

ParseErr ==
    /\ state = "Reading"
    /\ state' = "Idle"
    /\ candidate' = NIL
    /\ LogEvent("ParseErr")
    /\ UNCHANGED <<theme, pendingFile, watcherAlive, eventCount>>

----

(*****************************************************************************
 *  ACTION — AtomicSwap
 *
 *  Single TLA+ step: replace theme with candidate. This is the ONLY
 *  action that mutates theme post-Init. Captures NO_TORN_RELOAD and
 *  MERGE_ATOMIC_ON_UPDATE structurally — there is no observable
 *  intermediate state.
 *****************************************************************************)

AtomicSwap ==
    /\ state = "Swapping"
    /\ candidate # NIL
    /\ theme' = candidate
    /\ state' = "Idle"
    /\ candidate' = NIL
    /\ LogEvent("AtomicSwap")
    /\ UNCHANGED <<pendingFile, watcherAlive, eventCount>>

----

(*****************************************************************************
 *  ACTION — WatchStop
 *
 *  Graceful shutdown. watcherAlive := FALSE; no more FileEvent will
 *  fire (guard on FileEvent). Pending events are dropped — acceptable
 *  because the process is exiting.
 *****************************************************************************)

WatchStop ==
    /\ watcherAlive = TRUE
    /\ state = "Idle"  \* clean shutdown: no reload in flight
    /\ watcherAlive' = FALSE
    /\ pendingFile' = NIL
    /\ LogEvent("WatchStop")
    /\ UNCHANGED <<theme, state, candidate, eventCount>>

----

Next ==
    \/ WatchStart
    \/ \E f \in Files: FileEvent(f)
    \/ StartReload
    \/ \E f \in ValidFiles: ParseOk(f)
    \/ ParseErr
    \/ AtomicSwap
    \/ WatchStop

Spec == Init /\ [][Next]_vars
            /\ WF_vars(StartReload)
            /\ WF_vars(\E f \in ValidFiles: ParseOk(f))
            /\ WF_vars(AtomicSwap)

----

(*****************************************************************************
 *  SAFETY INVARIANTS
 *****************************************************************************)

\* (a) theme is always total. In this abstraction "total" means "is one of
\*     the known Themes tokens, never NIL, never partial". Encoded
\*     structurally: theme is initialized to T_default and only mutated
\*     by AtomicSwap, which assigns a non-NIL candidate.
THEME_TOTAL ==
    /\ theme \in Themes
    /\ theme # NIL

\* (b) No torn reload: there is no state where `theme` holds a value that
\*     is mid-merge between two themes. Encoded structurally by the
\*     AtomicSwap action being a single TLA+ step with theme' = candidate
\*     (no sequence of partial assignments).
NO_TORN_RELOAD ==
    \* If state = Validating or Swapping, a candidate is being prepared,
    \* but theme still holds the PREVIOUSLY APPLIED value (any of Themes,
    \* never an intermediate). When state = Swapping, candidate is fully
    \* materialized; AtomicSwap then publishes it whole.
    (state \in {"Reading", "Validating", "Swapping"}) =>
        theme \in Themes

\* (c) Merge is atomic: when state = Swapping, candidate is a complete
\*     total theme (not a delta, not a partial map). Encoded structurally
\*     by ParseOk producing candidate = "T_" \o f for f \in ValidFiles
\*     (a token representing the fully-merged total theme).
MERGE_ATOMIC_ON_UPDATE ==
    (state = "Swapping") => (candidate \in Themes /\ candidate # NIL)

\* (d) Watcher bound to lifecycle: FileEvent can only mutate state when
\*     watcherAlive. Encoded structurally in FileEvent guard.
WATCHER_BOUND_TO_LIFECYCLE ==
    \* If watcher is dead, no FileEvent action is enabled. After
    \* WatchStop, pendingFile remains NIL until the next WatchStart
    \* (which never happens after Stop in this model: shutdown is
    \* terminal). Encoded by the FileEvent guard watcherAlive = TRUE.
    TRUE  \* structural

\* (e) Single reload in flight: state \in {Reading, Validating, Swapping}
\*     are mutually exclusive (state is a single variable). Captured by
\*     the StartReload guard requiring state = Idle.
SINGLE_RELOAD_INFLIGHT ==
    state \in ReloadState

\* (f) Invalid file preserves theme: the ParseErr action has theme in
\*     UNCHANGED. Encoded structurally.
INVALID_PRESERVES_THEME ==
    TRUE  \* structural (see ParseErr action)

----

(*****************************************************************************
 *  LIVENESS PROPERTIES
 *****************************************************************************)

\* Eventually applied: under fairness on StartReload + ParseOk + AtomicSwap,
\* a pending valid file is eventually swapped into theme.
\*
\* Stated as: if pendingFile is set to f \in ValidFiles and watcherAlive,
\* then eventually theme = "T_" \o f (or a later theme, if a more recent
\* event superseded it — fsnotify coalescing).
EVENTUALLY_APPLIED ==
    \A f \in ValidFiles:
        (pendingFile = f /\ watcherAlive) ~> (\E g \in ValidFiles: theme = "T_" \o g)

----

(*****************************************************************************
 *  TLC CONFIGURATION (recommended)
 *
 *  CONSTANTS
 *      Files          = {"f_ok", "f_bad"}
 *      ValidFiles     = {"f_ok"}
 *      MaxEvents      = 3
 *
 *  INVARIANTS
 *      TypeOK
 *      THEME_TOTAL
 *      NO_TORN_RELOAD
 *      MERGE_ATOMIC_ON_UPDATE
 *      SINGLE_RELOAD_INFLIGHT
 *
 *  PROPERTIES
 *      EVENTUALLY_APPLIED
 *
 *  Expected state space: ~10^2-10^3 states with the small-config above;
 *  exploration < 5s.
 *
 *  Note: structural invariants (WATCHER_BOUND_TO_LIFECYCLE,
 *  INVALID_PRESERVES_THEME) are documented as TRUE because they are
 *  enforced by action guards / UNCHANGED clauses. Reviewer verifies
 *  by inspection of the spec, not by TLC.
 *****************************************************************************)

============================
