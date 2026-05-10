package views

// SearchInChatOpenMsg apre la barra inline (Ctrl+F).
type SearchInChatOpenMsg struct{}

// SearchInChatCloseMsg chiude la barra e ripristina ReturnTo.
type SearchInChatCloseMsg struct{}

// SearchInChatNextMsg naviga al match successivo (Enter/n).
type SearchInChatNextMsg struct{}

// SearchInChatPrevMsg naviga al match precedente (Shift+Tab/N).
type SearchInChatPrevMsg struct{}
