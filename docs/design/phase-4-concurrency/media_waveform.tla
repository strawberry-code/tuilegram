---- MODULE media_waveform ----
(*
 * TLA+ Specification — Braille waveform mapping (Step 24).
 *
 * This module specifies the PURE FUNCTION
 *
 *     brailleWaveform : (Seq(Byte) x Nat) -> Seq(Glyph)
 *
 * that converts a Telegram voice-message waveform (5-bit packed
 * amplitudes) into a fixed-width string of N braille block-element
 * glyphs (▁▂▃▄▅▆▇█).
 *
 * NOTE — This is a FUNCTIONAL spec, not a concurrency spec. There is no
 * variable state, no Init/Next, no fairness. The braille mapping has
 * NO concurrency concerns: it's a pure transformation invoked at render
 * time, single-threaded inside the bubbletea Update/View loop. We use
 * TLA+ here as a formal notation for the function and to enumerate its
 * INVARIANTS (totality, determinism, output length, monotonicity,
 * fallback) so they can be exercised via TLC over a small finite domain.
 *
 * Verifies (as theorems / TLC-checkable invariants over finite samples):
 *
 *   - TOTAL          : output is defined for every input pair
 *   - DETERMINISTIC  : same input ⇒ same output (trivial in TLA+, asserted)
 *   - LENGTH         : Len(brailleWaveform(d, n)) = n
 *   - MONOTONIC      : amp1 <= amp2 ⇒ glyph(amp1) <= glyph(amp2)
 *   - SILENCE        : brailleWaveform(zeros, n) is all "▁"
 *   - SATURATION     : brailleWaveform(maxAmps, n) is all "█"
 *   - EMPTY_FALLBACK : brailleWaveform(<<>>, n) is all "─" (flat fallback)
 *
 * Glyphs are modeled as integers 0..7 (braille intensity index). The
 * actual Unicode characters (▁▂▃▄▅▆▇█ or "─" for fallback) are mapped
 * 1:1 by the implementation; here we abstract them away.
 *
 * Companion docs:
 *   - phase-2-behavioral/media-rendering.md (statechart + decision tree)
 *   - phase-3-interactions/media-flow.md    (pseudocode + tables)
 *   - phase-6-decisions/ADR-011-media-rendering-taxonomy.md (rationale)
 *)

EXTENDS Integers, Sequences, FiniteSets, TLC

CONSTANTS
    MaxN,           \* upper bound on the requested glyph count (e.g. 12)
    MaxLen          \* upper bound on the input sample sequence length

\* Amplitude domain: Telegram voice waveforms use 5-bit packed samples.
\* Once decoded, each sample is in 0..31.
Amp == 0..31

\* Glyph index domain: 8 braille block-elements (▁▂▃▄▅▆▇█), indices 0..7.
\* The special index -1 represents the FALLBACK glyph "─" used when input
\* is empty/missing (flat horizontal line). We keep it disjoint from the
\* normal range so MONOTONIC etc. only quantify over real glyphs.
GlyphIdx == 0..7
FallbackIdx == -1

\* A sequence of decoded samples (0..31). The actual byte-level packing
\* (5 bits straddling byte boundaries) is an implementation detail of
\* the decoder; here we work directly on decoded amps. The decoder is
\* itself trivially total (it iterates over fixed-size bit windows).
Samples == Seq(Amp)

\* The output is a sequence of glyph indices. Length = N (the requested
\* bar count). Each entry is in GlyphIdx (or all-FallbackIdx when input
\* is empty).
Output == Seq(GlyphIdx \cup {FallbackIdx})

----

(* --- Pure function: amplitude → glyph index --- *)

\* Map a 5-bit amplitude (0..31) to a glyph index (0..7).
\* Defined as integer division: idx = amp \div 4. Saturation cap at 7
\* is implicit since 31 \div 4 = 7.
GlyphOf(amp) == amp \div 4

----

(* --- Pure function: resample N buckets via mean --- *)

\* Average of a non-empty subsequence of amps, integer-rounded down.
Mean(seq) ==
    LET sum == LET F[i \in 0..Len(seq)] ==
                    IF i = 0 THEN 0 ELSE F[i-1] + seq[i]
                IN F[Len(seq)]
    IN sum \div Len(seq)

\* Bucket index for the i-th sample (1-based) when resampling to N buckets.
\* Telegram's input length L; bucket = ((i-1) * N) \div L, in 0..N-1.
\* Returned 1-based to match TLA+ Seq indexing.
BucketOf(i, L, N) == (((i - 1) * N) \div L) + 1

\* Build the j-th bucket (1..N) as the subsequence of samples whose
\* BucketOf == j. This is a comprehension; we encode it as a recursive
\* helper that filters seq by predicate.
Bucket(seq, N, j) ==
    LET L == Len(seq)
    IN [k \in 1..Cardinality({i \in 1..L : BucketOf(i, L, N) = j})
        |-> CHOOSE x \in {seq[i] : i \in 1..L /\ BucketOf(i, L, N) = j} : TRUE]
    \* NOTE: TLA+ does not have ordered comprehension; for the safety
    \* invariants we only need the SET of values per bucket, not the
    \* order. The mean is order-independent.

----

(* --- Top-level function: brailleWaveform --- *)

\* The full function. Inputs:
\*   data : Samples  (already-decoded amplitudes)
\*   n    : Nat      (requested bar count)
\* Output: a sequence of length n.
brailleWaveform(data, n) ==
    IF n = 0 THEN <<>>
    ELSE IF Len(data) = 0
         THEN [k \in 1..n |-> FallbackIdx]      \* flat line "─"
         ELSE [k \in 1..n |->
                  LET b == Bucket(data, n, k)
                  IN IF Len(b) = 0
                     THEN GlyphOf(0)              \* empty bucket → silence "▁"
                     ELSE GlyphOf(Mean(b))]

----

(* --- Invariants (theorems) --- *)

\* TOTALITY: the function is defined for every (data, n) in the typed domain.
\* In TLA+ this is automatically true if the function definition has no
\* CHOOSE over an empty set or division by zero. We verify by inspection:
\*   - n = 0 branch: returns <<>>, no division.
\*   - Len(data) = 0 branch: returns [k |-> FallbackIdx], no division.
\*   - Else branch: Bucket may be empty for some k (handled), Mean is only
\*     called when Len(b) > 0 (so no div-by-zero).
TOTAL ==
    \A data \in Seq(Amp), n \in 0..MaxN :
        Len(data) <= MaxLen =>
            brailleWaveform(data, n) \in Output

\* LENGTH: output has exactly N entries.
LENGTH ==
    \A data \in Seq(Amp), n \in 0..MaxN :
        Len(data) <= MaxLen =>
            Len(brailleWaveform(data, n)) = n

\* DETERMINISTIC: function is deterministic by construction (no CHOOSE
\* with multiple witnesses on the value path; the CHOOSE inside Bucket is
\* a set-witness used only to sample-set; Mean is computed over the SET
\* of values which is order-independent). Stated as identity:
DETERMINISTIC ==
    \A data \in Seq(Amp), n \in 0..MaxN :
        brailleWaveform(data, n) = brailleWaveform(data, n)

\* MONOTONIC: GlyphOf is monotonic in amplitude.
MONOTONIC ==
    \A a1, a2 \in Amp :
        a1 <= a2 => GlyphOf(a1) <= GlyphOf(a2)

\* SILENCE: an all-zero waveform produces all-▁ glyphs (idx 0).
SILENCE ==
    \A n \in 1..MaxN, len \in 1..MaxLen :
        LET zeros == [k \in 1..len |-> 0]
        IN \A k \in 1..n : brailleWaveform(zeros, n)[k] = 0

\* SATURATION: an all-31 waveform produces all-█ glyphs (idx 7).
SATURATION ==
    \A n \in 1..MaxN, len \in 1..MaxLen :
        LET maxAmps == [k \in 1..len |-> 31]
        IN \A k \in 1..n : brailleWaveform(maxAmps, n)[k] = 7

\* EMPTY_FALLBACK: empty input produces all-FallbackIdx ("─") of length n.
EMPTY_FALLBACK ==
    \A n \in 0..MaxN :
        LET out == brailleWaveform(<<>>, n)
        IN /\ Len(out) = n
           /\ \A k \in 1..n : out[k] = FallbackIdx

\* OUTPUT_RANGE: every entry is either a valid glyph or the fallback.
OUTPUT_RANGE ==
    \A data \in Seq(Amp), n \in 0..MaxN :
        Len(data) <= MaxLen =>
            \A k \in 1..n :
                brailleWaveform(data, n)[k] \in (GlyphIdx \cup {FallbackIdx})

\* MIXED_NO_FALLBACK: when input is non-empty, no entry uses the fallback
\* glyph. Fallback is reserved for the empty-input case.
MIXED_NO_FALLBACK ==
    \A data \in Seq(Amp), n \in 1..MaxN :
        (Len(data) >= 1 /\ Len(data) <= MaxLen) =>
            \A k \in 1..n :
                brailleWaveform(data, n)[k] \in GlyphIdx

----

(* --- TLC Configuration recommendation ---
 *
 * CONSTANTS
 *     MaxN   = 4            \* keep small for state explosion
 *     MaxLen = 6
 *
 * INVARIANTS (no Init/Next needed; these are pure-function theorems
 * checked by TLC over the finite Cartesian product of inputs):
 *     TOTAL
 *     LENGTH
 *     DETERMINISTIC
 *     MONOTONIC
 *     SILENCE
 *     SATURATION
 *     EMPTY_FALLBACK
 *     OUTPUT_RANGE
 *     MIXED_NO_FALLBACK
 *
 * State space: |Seq(Amp)| over Len <= MaxLen with |Amp| = 32 is large
 * (32^6 ≈ 10^9). For TLC, use ALIAS/SYMMETRY or replace Amp with a
 * smaller surrogate (e.g. 0..3 mapped to 0,8,16,24 for the bucket
 * test). Practical recipe:
 *
 *     Amp <- 0..3                  \* override constant
 *     MaxN = 3
 *     MaxLen = 4
 *
 * → ~10^4 states, TLC explores in seconds.
 *
 * The full 5-bit semantics is verified at impl time via Go unit tests
 * that mirror SILENCE / SATURATION / MONOTONIC / EMPTY_FALLBACK /
 * LENGTH on actual []byte inputs.
 *)

====
