package views

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// inputAreaHeight = 2 (border-top + 1 riga testo). +1 con reply bar (handled in viewMessages).
const inputAreaHeight = 2

// ConversationModel gestisce il pannello destro con la conversazione.
type ConversationModel struct {
	Width, Height int
	chat          model.Chat
	active        bool
	loading       bool
	messages      []model.Message
	spinner       spinner.Model
	viewport      viewport.Model
	textarea      textarea.Model
	sendBtn       components.ButtonModel
	inputFocus    bool
	cursor        int
	replyTo       *model.Message
	editMode      bool
	editMsgIdx    int
	deleteMode    bool
	deleteMsgIDs  []int
	forwardPicker components.ForwardPickerModel
	forwardSource []model.Message
	selection     map[int]struct{} // set S (MODE_COHERENCE: multiSelect = len(selection)>0)
	multiSelect   bool
	Typing        bool              // typing indicator (Step 23, ADR-010)
	searchBar     SearchInChatState // barra inline Ctrl+F (Step 27, ADR-014)
	searchInput   textinput.Model
	// Step 33: snapshot del messaggio pinnato (DA1). nil = nessun pin.
	// Invariante PINNED_SINGLE_PER_CHAT: sostituito a ogni chat switch.
	pinnedMsg *model.Message
}

// NewConversationModel crea un nuovo modello per la conversazione.
func NewConversationModel() ConversationModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	ta := textarea.New()
	ta.Placeholder = "Message..."
	ta.CharLimit = 4096
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))

	return ConversationModel{
		spinner:       sp,
		textarea:      ta,
		sendBtn:       components.NewButton("SEND"),
		forwardPicker: components.NewForwardPicker(),
		selection:     make(map[int]struct{}),
	}
}

// OpenChat apre una chat e avvia lo spinner di caricamento.
// Resetta multi-select, search, pinned (SELECTION_SCOPE + cross-chat reset ADR-014).
// Invariante PINNED_SINGLE_PER_CHAT: pinnedMsg azzerato prima di caricare il nuovo.
func (m *ConversationModel) OpenChat(chat model.Chat) tea.Cmd {
	m.chat = chat
	m.active = true
	m.loading = true
	m.messages = nil
	m.inputFocus = false
	m.sendBtn.Active = false
	m.cursor = -1
	m.replyTo = nil
	m.editMode = false
	m.editMsgIdx = 0
	m.deleteMode = false
	m.deleteMsgIDs = nil
	m.forwardPicker = m.forwardPicker.Close()
	m.forwardSource = nil
	m.selection = make(map[int]struct{})
	m.multiSelect = false
	m.searchBar = SearchInChatState{}
	m.searchInput.Blur()
	m.textarea.Reset()
	m.textarea.Blur()
	m.viewport.SetContent("")
	m.pinnedMsg = nil // PINNED_SINGLE_PER_CHAT reset
	return m.spinner.Tick
}

// Close chiude la conversazione e ripristina lo stato vuoto.
func (m *ConversationModel) Close() {
	m.active = false
	m.chat = model.Chat{}
	m.messages = nil
	m.loading = false
	m.inputFocus = false
	m.sendBtn.Active = false
	m.cursor = -1
	m.replyTo = nil
	m.editMode = false
	m.editMsgIdx = 0
	m.deleteMode = false
	m.deleteMsgIDs = nil
	m.forwardPicker = m.forwardPicker.Close()
	m.forwardSource = nil
	m.selection = make(map[int]struct{})
	m.multiSelect = false
	m.searchBar = SearchInChatState{} // cross-chat reset (ADR-014)
	m.searchInput.Blur()
	m.pinnedMsg = nil
}
