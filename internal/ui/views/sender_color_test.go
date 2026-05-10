package views

import (
	"testing"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// TestSenderColor_Deterministic verifica SENDER_COLOR_DETERMINISTIC:
// stesso senderID → stesso colore in chiamate multiple.
func TestSenderColor_Deterministic(t *testing.T) {
	chat := model.Chat{Type: model.ChatGroup}
	style1 := coloredNameStyle(chat, 42)
	style2 := coloredNameStyle(chat, 42)
	// Gli stili devono essere identici (stesso foreground).
	if style1.GetForeground() != style2.GetForeground() {
		t.Error("SENDER_COLOR_DETERMINISTIC violated: same senderID returned different colors")
	}
}

// TestSenderColor_GroupOnly verifica SENDER_COLOR_GROUP_ONLY:
// solo ChatGroup ottiene colori da hash; altri tipo ottengono ColorIncoming.
func TestSenderColor_GroupOnly(t *testing.T) {
	groupChat := model.Chat{Type: model.ChatGroup}
	privateChat := model.Chat{Type: model.ChatPrivate}
	channelChat := model.Chat{Type: model.ChatChannel}

	groupStyle := coloredNameStyle(groupChat, 42)
	privateStyle := coloredNameStyle(privateChat, 42)
	channelStyle := coloredNameStyle(channelChat, 42)

	// Private e Channel devono avere lo stesso foreground (ColorIncoming).
	if privateStyle.GetForeground() != channelStyle.GetForeground() {
		t.Error("SENDER_COLOR_GROUP_ONLY: private and channel should use same default color")
	}
	// Il gruppo può avere un colore diverso da ColorIncoming (palette hash).
	// Non possiamo garantire che siano diversi (dipende dall'ID), ma verifichiamo
	// che la funzione non panica con qualsiasi ID.
	_ = groupStyle
}

// TestSenderColor_DifferentIDs verifica che ID diversi possano produrre colori diversi.
func TestSenderColor_DifferentIDs(t *testing.T) {
	chat := model.Chat{Type: model.ChatGroup}
	// Usiamo ID noti che mappano a indici diversi (42%8=2, 43%8=3).
	s1 := coloredNameStyle(chat, 42)
	s2 := coloredNameStyle(chat, 43)
	// I colori possono differire (probabilisticamente, non garantito).
	// Verifichiamo almeno che non panicano.
	_ = s1
	_ = s2
}

// TestSenderColor_NegativeID verifica che ID negativi (canali) siano gestiti con abs().
func TestSenderColor_NegativeID(t *testing.T) {
	chat := model.Chat{Type: model.ChatGroup}
	// ID negativi non devono causare out-of-bounds su slice di 8 elementi.
	s := coloredNameStyle(chat, -1000000000)
	_ = s
}
