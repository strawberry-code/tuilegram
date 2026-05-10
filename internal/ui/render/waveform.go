// Package render contiene funzioni di rendering pure (no I/O, no tea.Cmd).
// Separato da views/ per evitare dipendenze circolari (model → render).
package render

import "strings"

// glyphs è il charset block-element (U+2581..U+2588), 8 livelli di intensità.
// Scelto per monospace garantito e fontmap universale (ADR-011 §Charset).
var glyphs = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// Waveform converte un waveform Telegram (5-bit packed, little-endian) in una
// stringa di esattamente n glifi block-element.
//
// Spec formale: phase-4-concurrency/media_waveform.tla
// Invarianti garantite:
//   - TOTAL: definito per ogni input (nil, vuoto, lunghezze arbitrarie).
//   - LENGTH: output ha esattamente n rune (glifi block-element o "─").
//   - EMPTY_FALLBACK: data vuoto → n × "─".
//   - SILENCE: tutti 0 → tutti "▁".
//   - SATURATION: tutti 31 → tutti "█".
//   - MONOTONIC: ampiezza maggiore → glifo più alto (amp>>2 monotono).
func Waveform(data []byte, n int) string {
	if n <= 0 {
		return ""
	}
	if len(data) == 0 {
		return strings.Repeat("─", n)
	}

	samples := decode5bit(data)
	if len(samples) == 0 {
		return strings.Repeat("─", n)
	}

	buckets := resample(samples, n)
	var sb strings.Builder
	for _, mean := range buckets {
		idx := int(mean) >> 2 // amp/4, range 0..7 (31>>2 == 7)
		if idx > 7 {
			idx = 7 // clamp difensivo
		}
		sb.WriteString(glyphs[idx])
	}
	return sb.String()
}

// decode5bit decodifica campioni a 5 bit packed little-endian da data.
// Ogni campione occupa bit [i*5 .. i*5+4]; può straddle due byte.
// Non legge oltre len(data). Output: slice di ampiezza in 0..31.
func decode5bit(data []byte) []uint8 {
	totalBits := len(data) * 8
	count := totalBits / 5
	if count == 0 {
		return nil
	}
	out := make([]uint8, 0, count)
	for i := 0; i < count; i++ {
		bitPos := i * 5
		byteIdx := bitPos / 8
		bitOff := uint(bitPos % 8)
		v := uint8((data[byteIdx] >> bitOff) & 0x1F)
		if bitOff > 3 && byteIdx+1 < len(data) {
			// campione straddling due byte: aggiunge i bit superiori
			v |= (data[byteIdx+1] << (8 - bitOff)) & 0x1F
		}
		out = append(out, v)
	}
	return out
}

// resample distribuisce samples in n bucket uguali e restituisce la media
// intera di ciascun bucket. Bucket vuoti (n > len(samples)) → 0 ("▁").
// Algoritmo: per ogni sample i, bucket b = (i*n)/len(samples).
func resample(samples []uint8, n int) []uint8 {
	L := len(samples)
	sums := make([]int, n)
	counts := make([]int, n)
	for i, s := range samples {
		b := (i * n) / L // floor, range 0..n-1
		sums[b] += int(s)
		counts[b]++
	}
	out := make([]uint8, n)
	for b := range out {
		if counts[b] > 0 {
			out[b] = uint8(sums[b] / counts[b])
		}
		// counts[b]==0 → out[b]=0 già per zero-value → glifo "▁"
	}
	return out
}
