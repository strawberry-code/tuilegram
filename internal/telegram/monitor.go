package telegram

import (
	"context"
	"net"
	"time"
)

const (
	healthCheckInterval = 5 * time.Second
	healthCheckTimeout  = 2 * time.Second
	// DC2 di Telegram (Amsterdam) — usato per il connectivity check.
	telegramDC2Addr = "149.154.167.50:443"
)

// startHealthMonitor avvia un goroutine che controlla periodicamente
// la connettività di rete verso i server Telegram.
func (b *Bridge) startHealthMonitor(ctx context.Context) {
	go b.healthLoop(ctx)
}

func (b *Bridge) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	wasConnected := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			alive := checkTCPConnectivity(telegramDC2Addr, healthCheckTimeout)
			if alive && !wasConnected {
				wasConnected = true
				if b.OnConnected != nil {
					b.OnConnected()
				}
			} else if !alive && wasConnected {
				wasConnected = false
				if b.OnReconnecting != nil {
					b.OnReconnecting()
				}
			}
		}
	}
}

// checkTCPConnectivity tenta una connessione TCP diretta a un server Telegram.
func checkTCPConnectivity(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
