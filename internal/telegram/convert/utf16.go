package convert

import "unicode/utf16"

// Utf16Slice estrae text[offset:offset+length] usando UTF-16 code units.
// Telegram MTProto MessageEntity.Offset/Length sono UTF-16, non rune Go.
// Robusto a offset/length malformati: clamp ai bordi senza panic.
func Utf16Slice(text string, offset, length int) string {
	if offset < 0 || length <= 0 {
		return ""
	}
	units := utf16.Encode([]rune(text))
	if offset >= len(units) {
		return ""
	}
	end := offset + length
	if end > len(units) {
		end = len(units)
	}
	return string(utf16.Decode(units[offset:end]))
}

// Utf16Length conta UTF-16 code units in s.
// Necessario per segmentare il testo in renderTextWithLinks con corretto cursore.
func Utf16Length(s string) int {
	return len(utf16.Encode([]rune(s)))
}
