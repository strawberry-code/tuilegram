// Package convert — service.go
// Formats tg.MessageActionClass → human-readable ServiceText string (Step 25).
//
// Mapped variants: 12 common action types + "Service message" fallback.
// Totality invariant: every action, including unknown future variants, returns
// a non-empty string (no panic, no empty string).
//
// Spec: docs/design/phase-5-data/entity-mapping.md §System Message Mapping
// Spec: docs/design/phase-2-behavioral/reactions-and-system.md §System Message Classification
package convert

import (
	"fmt"

	"github.com/gotd/td/tg"
)

// FormatAction converts a tg.MessageActionClass to a pre-formatted human-readable
// string. The names map resolves user IDs to display names; actorID is the
// message sender (used as fallback when the action does not embed a user ID).
//
// ADR-012 B1: ServiceText is computed once at convert time (entities available
// in the MTProto response). Render-time lookup is avoided.
func FormatAction(action tg.MessageActionClass, names map[int64]string, actorID int64) string {
	actor := ResolveName(names, actorID)
	switch a := action.(type) {
	case *tg.MessageActionChatAddUser:
		return formatJoined(a.Users, actor, names)
	case *tg.MessageActionChatJoinedByLink:
		return actor + " joined via invite link"
	case *tg.MessageActionChatJoinedByRequest:
		return actor + " joined"
	case *tg.MessageActionChatDeleteUser:
		target := ResolveName(names, a.UserID)
		if target == actor {
			return actor + " left"
		}
		return actor + " removed " + target
	case *tg.MessageActionChatEditTitle:
		return fmt.Sprintf("%s changed title to %q", actor, a.Title)
	case *tg.MessageActionChatEditPhoto:
		return actor + " updated group photo"
	case *tg.MessageActionChatDeletePhoto:
		return actor + " removed group photo"
	case *tg.MessageActionPinMessage:
		return actor + " pinned a message"
	case *tg.MessageActionChatCreate:
		return actor + " created the chat"
	case *tg.MessageActionChannelCreate:
		return actor + " created the channel"
	case *tg.MessageActionPhoneCall:
		return formatCall(a)
	case *tg.MessageActionScreenshotTaken:
		return actor + " took a screenshot"
	default:
		// Totality guarantee: unknown future variants receive generic fallback.
		return "Service message"
	}
}

// formatJoined builds the join text for MessageActionChatAddUser.
// Single user == actor: "{actor} joined". Multiple users: "{actor} added {target1}, {target2}".
func formatJoined(users []int64, actor string, names map[int64]string) string {
	if len(users) == 1 {
		return actor + " joined"
	}
	targets := ""
	for i, uid := range users {
		if i > 0 {
			targets += ", "
		}
		targets += ResolveName(names, uid)
	}
	return actor + " added " + targets
}

// formatCall produces the call description for MessageActionPhoneCall.
// Missed call has a specific Reason; otherwise duration is shown.
func formatCall(a *tg.MessageActionPhoneCall) string {
	if _, missed := a.Reason.(*tg.PhoneCallDiscardReasonMissed); missed {
		return "Missed call"
	}
	if a.Duration > 0 {
		mins := a.Duration / 60
		secs := a.Duration % 60
		return fmt.Sprintf("Call · %d:%02d", mins, secs)
	}
	return "Call"
}
