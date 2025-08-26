package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mau.fi/whatsmeow/types"
)

type ContactInfo struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	PushName    string `json:"push_name"`
	BusinessName string `json:"business_name,omitempty"`
	VerifiedName string `json:"verified_name,omitempty"`
	Status      string `json:"status,omitempty"`
}

func (api *WhatsAppAPI) getContacts(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	// Get all contacts from store
	contacts := api.client.Store.Contacts
	if contacts == nil {
		api.writeSuccess(w, []ContactInfo{})
		return
	}

	// Convert to response format
	contactList := make([]ContactInfo, 0, len(contacts))
	for jid, contact := range contacts {
		contactInfo := ContactInfo{
			JID:      jid.String(),
			Name:     contact.FirstName,
			FullName: contact.FullName,
			PushName: contact.PushName,
		}

		// Add business info if available
		if contact.BusinessName != "" {
			contactInfo.BusinessName = contact.BusinessName
		}
		if contact.VerifiedName != "" {
			contactInfo.VerifiedName = contact.VerifiedName
		}

		contactList = append(contactList, contactInfo)
	}

	api.writeSuccess(w, contactList)
}

func (api *WhatsAppAPI) getContactInfo(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	vars := mux.Vars(r)
	jidStr := vars["jid"]

	// Parse JID
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid JID: "+err.Error())
		return
	}

	// Get contact info
	contact, exists := api.client.Store.Contacts[jid]
	if !exists {
		api.writeError(w, http.StatusNotFound, "Contact not found")
		return
	}

	contactInfo := ContactInfo{
		JID:      jid.String(),
		Name:     contact.FirstName,
		FullName: contact.FullName,
		PushName: contact.PushName,
	}

	// Add business info if available
	if contact.BusinessName != "" {
		contactInfo.BusinessName = contact.BusinessName
	}
	if contact.VerifiedName != "" {
		contactInfo.VerifiedName = contact.VerifiedName
	}

	api.writeSuccess(w, contactInfo)
}