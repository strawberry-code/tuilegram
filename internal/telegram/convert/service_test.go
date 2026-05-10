package convert_test

import (
	"strings"
	"testing"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

var actorMap = map[int64]string{1: "Alice", 2: "Bob", 3: "Carol"}

func TestFormatAction_AddSingleUser_IsSelfJoin(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionChatAddUser{Users: []int64{1}}, actorMap, 1)
	if got != "Alice joined" {
		t.Fatalf("expected 'Alice joined', got %q", got)
	}
}

func TestFormatAction_AddMultipleUsers_ListsTargets(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionChatAddUser{Users: []int64{2, 3}}, actorMap, 1)
	if !strings.Contains(got, "Alice added") || !strings.Contains(got, "Bob") || !strings.Contains(got, "Carol") {
		t.Fatalf("expected 'Alice added Bob, Carol' shape, got %q", got)
	}
}

func TestFormatAction_DeleteUser_SelfBecomesLeave(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionChatDeleteUser{UserID: 1}, actorMap, 1)
	if got != "Alice left" {
		t.Fatalf("expected 'Alice left', got %q", got)
	}
}

func TestFormatAction_DeleteUser_OtherIsRemoval(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionChatDeleteUser{UserID: 2}, actorMap, 1)
	if got != "Alice removed Bob" {
		t.Fatalf("expected 'Alice removed Bob', got %q", got)
	}
}

func TestFormatAction_PinMessage(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionPinMessage{}, actorMap, 1)
	if got != "Alice pinned a message" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatAction_EditTitle_QuotesNewTitle(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionChatEditTitle{Title: "General"}, actorMap, 1)
	if !strings.Contains(got, `"General"`) {
		t.Fatalf("expected quoted title, got %q", got)
	}
}

func TestFormatAction_PhoneCall_Missed(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionPhoneCall{
		Reason: &tg.PhoneCallDiscardReasonMissed{},
	}, actorMap, 1)
	if got != "Missed call" {
		t.Fatalf("expected 'Missed call', got %q", got)
	}
}

func TestFormatAction_PhoneCall_DurationFormatted(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionPhoneCall{Duration: 125}, actorMap, 1)
	if got != "Call · 2:05" {
		t.Fatalf("expected 'Call · 2:05', got %q", got)
	}
}

func TestFormatAction_PhoneCall_NoDuration(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionPhoneCall{}, actorMap, 1)
	if got != "Call" {
		t.Fatalf("expected 'Call', got %q", got)
	}
}

func TestFormatAction_UnknownVariant_ReturnsFallback(t *testing.T) {
	// Use a variant we do not explicitly handle (e.g. SetChatTheme).
	got := convert.FormatAction(&tg.MessageActionSetChatTheme{}, actorMap, 1)
	if got != "Service message" {
		t.Fatalf("expected fallback 'Service message', got %q", got)
	}
}

func TestFormatAction_UnknownActor_ResolvesToUnknown(t *testing.T) {
	got := convert.FormatAction(&tg.MessageActionPinMessage{}, actorMap, 999)
	if got != "Unknown pinned a message" {
		t.Fatalf("expected actor=Unknown, got %q", got)
	}
}

func TestFormatAction_AllReturnsNonEmpty_TotalityInvariant(t *testing.T) {
	cases := []tg.MessageActionClass{
		&tg.MessageActionChatAddUser{Users: []int64{1}},
		&tg.MessageActionChatJoinedByLink{},
		&tg.MessageActionChatJoinedByRequest{},
		&tg.MessageActionChatDeleteUser{UserID: 2},
		&tg.MessageActionChatEditTitle{Title: "x"},
		&tg.MessageActionChatEditPhoto{},
		&tg.MessageActionChatDeletePhoto{},
		&tg.MessageActionPinMessage{},
		&tg.MessageActionChatCreate{},
		&tg.MessageActionChannelCreate{},
		&tg.MessageActionPhoneCall{},
		&tg.MessageActionScreenshotTaken{},
		&tg.MessageActionSetChatTheme{},
	}
	for i, c := range cases {
		if got := convert.FormatAction(c, actorMap, 1); got == "" {
			t.Fatalf("case %d (%T) returned empty string", i, c)
		}
	}
}
