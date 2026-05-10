package telegram

import (
	"context"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// SendCodeResult contiene i dati restituiti da Telegram dopo l'invio del codice.
type SendCodeResult struct {
	CodeHash   string
	CodeLength int
}

// SendCode invia il codice di verifica al numero di telefono.
func (b *Bridge) SendCode(ctx context.Context, phone string) (SendCodeResult, error) {
	sent, err := b.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return SendCodeResult{}, err
	}

	sc, ok := sent.(*tg.AuthSentCode)
	if !ok {
		return SendCodeResult{CodeLength: 5}, nil
	}

	return SendCodeResult{
		CodeHash:   sc.PhoneCodeHash,
		CodeLength: extractCodeLength(sc.Type),
	}, nil
}

// extractCodeLength estrae la lunghezza del codice da qualsiasi tipo di SentCode.
func extractCodeLength(t tg.AuthSentCodeTypeClass) int {
	switch v := t.(type) {
	case *tg.AuthSentCodeTypeApp:
		return v.Length
	case *tg.AuthSentCodeTypeSMS:
		return v.Length
	case *tg.AuthSentCodeTypeCall:
		return v.Length
	case *tg.AuthSentCodeTypeFlashCall:
		return 5
	case *tg.AuthSentCodeTypeMissedCall:
		return v.Length
	case *tg.AuthSentCodeTypeFragmentSMS:
		return v.Length
	case *tg.AuthSentCodeTypeFirebaseSMS:
		return v.Length
	default:
		return 5
	}
}

// SignIn tenta il login con phone + code.
func (b *Bridge) SignIn(ctx context.Context, phone, code, codeHash string) error {
	_, err := b.client.Auth().SignIn(ctx, phone, code, codeHash)
	return err
}

// CheckPassword invia la password 2FA.
func (b *Bridge) CheckPassword(ctx context.Context, password string) error {
	_, err := b.client.Auth().Password(ctx, password)
	return err
}

// IsAuthorized controlla se il client è attualmente autenticato.
func (b *Bridge) IsAuthorized(ctx context.Context) (bool, error) {
	status, err := b.client.Auth().Status(ctx)
	if err != nil {
		return false, err
	}
	return status.Authorized, nil
}
