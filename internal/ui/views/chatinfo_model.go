package views

import (
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// ChatInfoModel manages the chat info overlay (Step 29, ADR-017).
// Reuses Modal primitive (compact: true, placement: right).
// Value-receiver pattern: root owns state via returned copies.
type ChatInfoModel struct {
	// Active tracks whether the overlay is mounted (Open state).
	Active bool
	// target is the chat whose info is displayed.
	// nil ⟺ overlay closed (invariant INFO_TARGET_COHERENCE in TLA+).
	target *model.ChatID
	// card is the rendered data snapshot (cache-first, ADR-017 §D2).
	card ChatInfoCard
	// completionInFlight is true from open until ChatInfoCompletionMsg arrives.
	completionInFlight bool
	// vp is the scrollable body viewport.
	vp     viewport.Model
	Width  int
	Height int
}

// NewChatInfoModel returns a closed, zero-state ChatInfoModel.
func NewChatInfoModel() ChatInfoModel {
	return ChatInfoModel{}
}

// Target returns the current target ChatID (nil when closed).
func (m ChatInfoModel) Target() *model.ChatID { return m.target }

// Open mounts the overlay for the given chat.
// Returns the mutated model and whether a lazy completion fetch is needed.
func (m ChatInfoModel) Open(chat model.Chat) (ChatInfoModel, bool) {
	id := chat.ID
	m.Active = true
	m.target = &id
	m.card = buildInfoCard(chat)
	needsFetch := m.card.Bio == "" && chat.Type == model.ChatPrivate
	m.completionInFlight = needsFetch
	m.vp.GotoTop()
	return m, needsFetch
}

// Close dismounts the overlay and resets target.
func (m ChatInfoModel) Close() ChatInfoModel {
	m.Active = false
	m.target = nil
	m.card = ChatInfoCard{}
	m.completionInFlight = false
	return m
}

// buildInfoCard constructs a ChatInfoCard from a Chat domain object.
// Only basic fields available from the Chat type at Step 29 scope.
// Bio and detailed fields are populated by lazy completion (ADR-017 §D2).
func buildInfoCard(c model.Chat) ChatInfoCard {
	card := ChatInfoCard{
		Name:             c.Title,
		ChatType:         c.Type,
		SharedMediaCount: -1, // stub [?] per ADR-017 §D3
		SharedFilesCount: -1,
		SharedLinksCount: -1,
	}
	if c.IsOnline {
		card.OnlineStatus = "● Online"
	}
	return card
}

// MergeCompletion applies fields from ChatInfoCompletionMsg into the card.
// Caller must guard chatID == target before calling.
func (m *ChatInfoModel) MergeCompletion(msg ChatInfoCompletionMsg) {
	if msg.Bio != "" {
		m.card.Bio = msg.Bio
	}
	m.completionInFlight = false
}
