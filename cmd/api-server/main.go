package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// App encapsulates the server state.
type App struct {
	container *sqlstore.Container
	client    *whatsmeow.Client
	clientMu  sync.RWMutex
	qrMu      sync.Mutex
	qrChan    <-chan whatsmeow.QRChannelItem
	qrCancel  context.CancelFunc
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (a *App) getClient() *whatsmeow.Client {
	a.clientMu.RLock()
	defer a.clientMu.RUnlock()
	return a.client
}

func (a *App) setClient(c *whatsmeow.Client) {
	a.clientMu.Lock()
	defer a.clientMu.Unlock()
	a.client = c
}

func (a *App) init(ctx context.Context) error {
	logLevel := getenv("LOG_LEVEL", "INFO")
	dbDialect := getenv("DB_DIALECT", "sqlite3")
	dbURL := getenv("DB_URL", "file:whatsmeow.db?_foreign_keys=on")

	dbLog := waLog.Stdout("Database", logLevel, true)
	container, err := sqlstore.New(ctx, dbDialect, dbURL, dbLog)
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}
	a.container = container

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", logLevel, true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			clientLog.Infof("Received message from %s: %s", v.Info.Sender.String(), v.Message.GetConversation())
		case *events.Disconnected:
			clientLog.Warnf("Disconnected")
		}
	})
	a.setClient(client)
	return nil
}

// POST /connect - connects the websocket. If not paired, returns 409 and a hint to pair.
func (a *App) handleConnect(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	if client == nil {
		http.Error(w, "client not ready", http.StatusServiceUnavailable)
		return
	}
	if client.Store.ID == nil {
		http.Error(w, "not paired. call /pair/qr first", http.StatusConflict)
		return
	}
	if err := client.Connect(); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /disconnect
func (a *App) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	if client == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	client.Disconnect()
	w.WriteHeader(http.StatusNoContent)
}

// GET /status
func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	type status struct {
		Paired      bool        `json:"paired"`
		LoggedIn    bool        `json:"loggedIn"`
		User        string      `json:"user"`
		LastConnect time.Time   `json:"lastConnect"`
		Retries     int         `json:"retries"`
		ID          string      `json:"id"`
		LID         string      `json:"lid"`
		Devices     []types.JID `json:"devices,omitempty"`
	}
	s := status{}
	if client != nil {
		s.Paired = client.Store.ID != nil
		s.LoggedIn = client.IsConnected()
		if client.Store.ID != nil {
			s.User = client.Store.ID.String()
			s.ID = client.Store.ID.String()
			s.LID = client.Store.LID.String()
		}
		s.LastConnect = client.LastSuccessfulConnect
		s.Retries = client.AutoReconnectErrors
	}
	_ = json.NewEncoder(w).Encode(s)
}

// POST /pair/qr/start - begin QR pairing, returns a session id
func (a *App) handlePairQRStart(w http.ResponseWriter, r *http.Request) {
	a.qrMu.Lock()
	defer a.qrMu.Unlock()

	client := a.getClient()
	if client == nil {
		http.Error(w, "client not ready", http.StatusServiceUnavailable)
		return
	}
	if client.Store.ID != nil {
		http.Error(w, "already paired", http.StatusBadRequest)
		return
	}
	if a.qrChan != nil {
		http.Error(w, "pairing already in progress", http.StatusConflict)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		cancel()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Start connection in background
	go func() {
		_ = client.Connect()
	}()

	a.qrChan = qrChan
	a.qrCancel = cancel

	resp := struct{ Session string `json:"session"` }{Session: uuid.NewString()}
	_ = json.NewEncoder(w).Encode(resp)
}

// GET /pair/qr/next - stream next QR/code event
func (a *App) handlePairQRNext(w http.ResponseWriter, r *http.Request) {
	a.qrMu.Lock()
	qrChan := a.qrChan
	a.qrMu.Unlock()
	if qrChan == nil {
		http.Error(w, "no active pairing", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Wait for next item with timeout
	select {
	case item, ok := <-qrChan:
		if !ok {
			json.NewEncoder(w).Encode(map[string]string{"event": "closed"})
			return
		}
		json.NewEncoder(w).Encode(item)
		flusher.Flush()
		if item.Event == whatsmeow.QRChannelSuccess.Event || item.Error != nil || item.Event == whatsmeow.QRChannelTimeout.Event {
			// pairing finished, clear channel
			a.qrMu.Lock()
			if a.qrCancel != nil {
				a.qrCancel()
			}
			a.qrChan = nil
			a.qrCancel = nil
			a.qrMu.Unlock()
		}
	case <-time.After(60 * time.Second):
		http.Error(w, "timeout", http.StatusGatewayTimeout)
	}
}

// POST /messages/text {"to":"123@s.whatsapp.net","message":"hi"}
type sendTextRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func (a *App) handleSendText(w http.ResponseWriter, r *http.Request) {
	client := a.getClient()
	if client == nil {
		http.Error(w, "client not ready", http.StatusServiceUnavailable)
		return
	}
	if !client.IsConnected() {
		http.Error(w, "not connected", http.StatusBadRequest)
		return
	}
	var req sendTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	jid, err := types.ParseJID(req.To)
	if err != nil {
		http.Error(w, "invalid JID", http.StatusBadRequest)
		return
	}
	msg := &waE2E.Message{
		Conversation: &req.Message,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &App{}
	if err := app.init(ctx); err != nil {
		log.Fatalf("init failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/connect", app.handleConnect)
	mux.HandleFunc("/disconnect", app.handleDisconnect)
	mux.HandleFunc("/status", app.handleStatus)
	mux.HandleFunc("/pair/qr/start", app.handlePairQRStart)
	mux.HandleFunc("/pair/qr/next", app.handlePairQRNext)
	mux.HandleFunc("/messages/text", app.handleSendText)

	addr := getenv("ADDR", ":8080")

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("HTTP API listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
}