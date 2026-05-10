# Authentication Flow

Dettaglio del flusso di autenticazione Telegram con tutti gli edge case.

## State Machine Completa

```mermaid
stateDiagram-v2
    [*] --> CheckSession

    CheckSession --> PhoneInput : no session
    CheckSession --> ValidateSession : session exists

    ValidateSession --> PhoneInput : session expired
    ValidateSession --> Connected : session valid (IfNecessary returns)

    state PhoneInput {
        [*] --> WaitingPhoneInput
        WaitingPhoneInput --> ValidatingPhone : Enter
        ValidatingPhone --> WaitingPhoneInput : invalid format
        ValidatingPhone --> SendingCode : valid phone
    }

    SendingCode --> CodeInput : AuthSentCode received
    SendingCode --> PhoneInput : error (invalid phone / banned)
    SendingCode --> PhoneInput : PHONE_NUMBER_FLOOD

    state CodeInput {
        [*] --> WaitingCodeInput
        WaitingCodeInput --> VerifyingCode : Enter (all digits filled)

        state WaitingCodeInput {
            [*] --> DigitCell1
            DigitCell1 --> DigitCell2 : digit
            DigitCell2 --> DigitCell3 : digit
            DigitCell3 --> DigitCell4 : digit
            DigitCell4 --> DigitCell5 : digit
            DigitCell5 --> AllFilled : digit
            AllFilled --> DigitCell5 : backspace
            DigitCell5 --> DigitCell4 : backspace
            DigitCell4 --> DigitCell3 : backspace
            DigitCell3 --> DigitCell2 : backspace
            DigitCell2 --> DigitCell1 : backspace
        }
    }

    VerifyingCode --> PasswordInput : SESSION_PASSWORD_NEEDED
    VerifyingCode --> Connected : auth success
    VerifyingCode --> CodeInput : PHONE_CODE_INVALID
    VerifyingCode --> CodeInput : PHONE_CODE_EXPIRED (toast)
    VerifyingCode --> PhoneInput : PHONE_NUMBER_UNOCCUPIED (sign up not supported)

    state PasswordInput {
        [*] --> WaitingPasswordInput
        WaitingPasswordInput --> VerifyingPassword : Enter
    }

    VerifyingPassword --> Connected : auth success
    VerifyingPassword --> PasswordInput : PASSWORD_HASH_INVALID
    VerifyingPassword --> PasswordInput : SRP_ID_INVALID (retry SRP)

    Connected --> [*]
```

## Validazioni

### Phone Number
- Deve iniziare con `+`
- Solo cifre dopo il `+` (spazi e trattini rimossi automaticamente)
- Lunghezza: 7-15 cifre (standard ITU-T E.164)
- Validazione locale prima dell'invio al server

### 2FA Code
- Esattamente N cifre (N definito da `AuthSentCode.type.length`)
- Tipicamente 5 o 6 cifre
- Solo cifre 0-9
- Auto-submit possibile quando tutte le celle sono piene

### Password
- Qualsiasi stringa non vuota
- SRP (Secure Remote Password) negotiation con il server
- La password non viene mai inviata in chiaro

## Error Handling

| Errore Telegram | UI Response | Recovery |
|----------------|-------------|----------|
| `PHONE_NUMBER_INVALID` | Toast: "Invalid phone number" | Torna a PhoneInput |
| `PHONE_NUMBER_BANNED` | Toast: "Phone number banned" | Torna a PhoneInput |
| `PHONE_NUMBER_FLOOD` | Toast: "Too many attempts. Wait." | Torna a PhoneInput, timer |
| `PHONE_CODE_INVALID` | Toast: "Wrong code" | Svuota CodeInput, refocus |
| `PHONE_CODE_EXPIRED` | Toast: "Code expired. Resending..." | Re-invia codice |
| `SESSION_PASSWORD_NEEDED` | Transizione a PasswordInput | — |
| `PASSWORD_HASH_INVALID` | Toast: "Wrong password" | Svuota PasswordInput |
| `PHONE_NUMBER_UNOCCUPIED` | Toast: "Account not found" | Torna a PhoneInput |
| Network error | Toast: "Connection failed" | Retry con backoff |

## Sequence Diagram — Happy Path

```mermaid
sequenceDiagram
    participant U as User
    participant TUI as Auth View
    participant TG as Telegram Client
    participant SRV as Telegram Server

    U->>TUI: enters phone number + Enter
    TUI->>TG: auth.SendCode(phone)
    TG->>SRV: auth.sendCode
    SRV-->>TG: AuthSentCode{length: 5}
    TG-->>TUI: CodeInput state
    TUI->>U: show OTP cells (5)

    U->>TUI: enters code digits
    TUI->>TG: auth.SignIn(phone, code)
    TG->>SRV: auth.signIn
    SRV-->>TG: auth.Authorization{user}
    TG-->>TUI: AuthSuccessMsg{user}
    TUI->>U: transition to Loading → MainView
```

## Sequence Diagram — 2FA Path

```mermaid
sequenceDiagram
    participant U as User
    participant TUI as Auth View
    participant TG as Telegram Client
    participant SRV as Telegram Server

    U->>TUI: enters code digits + Enter
    TUI->>TG: auth.SignIn(phone, code)
    TG->>SRV: auth.signIn
    SRV-->>TG: SESSION_PASSWORD_NEEDED
    TG-->>TUI: transition to PasswordInput

    U->>TUI: enters password + Enter
    TUI->>TG: auth.Password(password)
    TG->>SRV: auth.checkPassword (SRP)
    SRV-->>TG: auth.Authorization{user}
    TG-->>TUI: AuthSuccessMsg{user}
```

## OTP Component — Behavior Detail

```
State: [1] [2] [_] [_] [_]    cursor on cell 3
                 ↑

Input '7':  [1] [2] [7] [_] [_]    auto-advance to cell 4
                      ↑

Backspace:  [1] [2] [_] [_] [_]    back to cell 3, clear
                 ↑

All filled: [1] [2] [7] [9] [8]    ready for Enter
                              ↑
```

| Input | Azione |
|-------|--------|
| Digit 0-9 | Scrivi nella cella corrente, advance |
| Backspace | Cancella cella corrente, retreat |
| Enter | Submit se tutte le celle sono piene |
| Esc | Torna allo step precedente |
| Left/Right | Naviga tra le celle manualmente |
