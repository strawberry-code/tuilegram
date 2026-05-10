package views

import (
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// openLinkChordMsg è il messaggio interno emesso dal chord gx (ADR-021 §DB4).
// Ricevuto da handleOverlayMsg → delegato a handleGxChord su MainModel.
type openLinkChordMsg struct{}

// openLinkCmd crea un tea.Cmd che apre url nel browser di sistema (ADR-021 §DB3).
// Dispatch via runtime.GOOS: darwin=open, linux/other=xdg-open, windows=rundll32.
// Invariante LINK_OPEN_HTTP_ONLY: chiamato solo per http(s) (filtro in gx handler).
// Fire-and-forget: nessun OpenLinkResultMsg — errori loggati a stderr.
func openLinkCmd(url string) tea.Cmd {
	return func() tea.Msg {
		_ = spawnBrowser(url)
		return nil // nessun ResultMsg: fire-and-forget (ADR-021 §DB3)
	}
}

// spawnBrowser lancia il comando di sistema per aprire url.
// Windows: usa rundll32 url.dll,FileProtocolHandler (no shell metachar parsing).
func spawnBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		// WARNING #3: rundll32 evita cmd.exe shell parsing di metacaratteri URL.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, freebsd, ecc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start() // Start, non Run: non aspettiamo la fine del browser
}

// handleOpenLink processa OpenLinkMsg: valida schema, spawna browser o scrive hint.
// Invariante LINK_OPEN_HTTP_ONLY: solo http(s) vengono aperti.
func (m MainModel) handleOpenLink(msg telegram.OpenLinkMsg) (MainModel, tea.Cmd) {
	if !strings.HasPrefix(msg.URL, "http://") && !strings.HasPrefix(msg.URL, "https://") {
		// BLOCKING #3: non echeggiare URL untrusted nello status bar (injection risk).
		m.statusMsg = "scheme not supported (non-http URL)"
		return m, nil
	}
	return m, openLinkCmd(msg.URL)
}

// handleGxChord gestisce il chord gx: apre il primo link del messaggio selezionato.
// BLOCKING #2: restituisce tea.Cmd invece di chiamare spawnBrowser direttamente
// per non bloccare la goroutine bubbletea durante Update().
// Se nessun link → status hint (no-op esplicito, ADR-021 §DB4).
func (m MainModel) handleGxChord() (MainModel, tea.Cmd) {
	cv := m.conversation
	if cv.cursor < 0 || cv.cursor >= len(cv.messages) {
		return m, nil
	}
	msg := cv.messages[cv.cursor]
	if len(msg.Links) == 0 {
		m.statusMsg = "no links in selected message"
		return m, nil
	}
	url := msg.Links[0].URL
	// openLinkCmd contiene già il filtro http(s) via spawnBrowser; l'URL
	// è già stato validato da ExtractLinks (LINK_OPEN_HTTP_ONLY invariant).
	return m, openLinkCmd(url)
}
