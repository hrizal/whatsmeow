package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type WhatsAppAPI struct {
	client *whatsmeow.Client
	router *mux.Router
	store  *sqlstore.Container
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SendMessageRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"` // text, image, video, audio, document
}

type CreateGroupRequest struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}

type GroupActionRequest struct {
	GroupID string   `json:"group_id"`
	Users   []string `json:"users,omitempty"`
}

func main() {
	// Initialize database
	db, err := sql.Open("sqlite3", "whatsapp.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize store
	container, err := sqlstore.New("sqlite3", "file:whatsapp.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Get device store
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Fatal("Failed to get device:", err)
	}

	// Initialize client
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)
	
	// Create API instance
	api := &WhatsAppAPI{
		client: client,
		router: mux.NewRouter(),
		store:  container,
	}

	// Setup routes
	api.setupRoutes()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: api.router,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("WhatsApp API server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}

func (api *WhatsAppAPI) setupRoutes() {
	// Health check
	api.router.HandleFunc("/health", api.healthCheck).Methods("GET")

	// Authentication routes
	api.router.HandleFunc("/auth/status", api.getAuthStatus).Methods("GET")
	api.router.HandleFunc("/auth/qr", api.getQRCode).Methods("GET")
	api.router.HandleFunc("/auth/logout", api.logout).Methods("POST")

	// Message routes
	api.router.HandleFunc("/messages/send", api.sendMessage).Methods("POST")
	api.router.HandleFunc("/messages/send-media", api.sendMedia).Methods("POST")
	api.router.HandleFunc("/messages/delete", api.deleteMessage).Methods("DELETE")

	// Group routes
	api.router.HandleFunc("/groups/create", api.createGroup).Methods("POST")
	api.router.HandleFunc("/groups/join", api.joinGroup).Methods("POST")
	api.router.HandleFunc("/groups/leave", api.leaveGroup).Methods("POST")
	api.router.HandleFunc("/groups/info", api.getGroupInfo).Methods("GET")
	api.router.HandleFunc("/groups/participants/add", api.addGroupParticipants).Methods("POST")
	api.router.HandleFunc("/groups/participants/remove", api.removeGroupParticipants).Methods("POST")
	api.router.HandleFunc("/groups/invite-link", api.getGroupInviteLink).Methods("GET")

	// Contact routes
	api.router.HandleFunc("/contacts", api.getContacts).Methods("GET")
	api.router.HandleFunc("/contacts/{jid}", api.getContactInfo).Methods("GET")

	// Media routes
	api.router.HandleFunc("/media/upload", api.uploadMedia).Methods("POST")
	api.router.HandleFunc("/media/download/{messageID}", api.downloadMedia).Methods("GET")

	// Status routes
	api.router.HandleFunc("/status/set", api.setStatus).Methods("POST")
	api.router.HandleFunc("/presence/set", api.setPresence).Methods("POST")

	// Event stream
	api.router.HandleFunc("/events", api.eventStream).Methods("GET")

	// Add middleware
	api.router.Use(api.loggingMiddleware)
	api.router.Use(api.corsMiddleware)
}

func (api *WhatsAppAPI) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (api *WhatsAppAPI) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (api *WhatsAppAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *WhatsAppAPI) writeError(w http.ResponseWriter, status int, message string) {
	api.writeJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}

func (api *WhatsAppAPI) writeSuccess(w http.ResponseWriter, data interface{}) {
	api.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}