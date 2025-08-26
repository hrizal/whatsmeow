#!/usr/bin/env python3
"""
WhatsApp API Example Client

This script demonstrates how to use the WhatsApp API endpoints.
Make sure the API server is running on localhost:8080 before running this script.
"""

import requests
import json
import time
import qrcode
from io import BytesIO
from PIL import Image

class WhatsAppAPIClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
    
    def health_check(self):
        """Check if the API server is healthy"""
        try:
            response = self.session.get(f"{self.base_url}/health")
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error connecting to API server: {e}")
            return None
    
    def get_auth_status(self):
        """Get authentication status"""
        response = self.session.get(f"{self.base_url}/auth/status")
        return response.json()
    
    def get_qr_code(self):
        """Get QR code for authentication"""
        response = self.session.get(f"{self.base_url}/auth/qr")
        data = response.json()
        
        if data.get("success") and data.get("data", {}).get("qr_code"):
            # Generate QR code image
            qr = qrcode.QRCode(version=1, box_size=10, border=5)
            qr.add_data(data["data"]["qr_code"])
            qr.make(fit=True)
            
            img = qr.make_image(fill_color="black", back_color="white")
            img.save("whatsapp_qr.png")
            print("QR code saved as 'whatsapp_qr.png'")
            print(f"Scan this QR code with WhatsApp to authenticate")
            print(f"QR code expires at: {data['data']['expires_at']}")
        
        return data
    
    def logout(self):
        """Logout from WhatsApp"""
        response = self.session.post(f"{self.base_url}/auth/logout")
        return response.json()
    
    def send_message(self, to_jid, message):
        """Send a text message"""
        payload = {
            "to": to_jid,
            "message": message,
            "type": "text"
        }
        response = self.session.post(
            f"{self.base_url}/messages/send",
            json=payload
        )
        return response.json()
    
    def send_media_message(self, to_jid, media_type, caption, media_url):
        """Send a media message"""
        payload = {
            "to": to_jid,
            "type": media_type,
            "caption": caption,
            "url": media_url
        }
        response = self.session.post(
            f"{self.base_url}/messages/send-media",
            json=payload
        )
        return response.json()
    
    def delete_message(self, chat_jid, message_id, for_everyone=True):
        """Delete a message"""
        payload = {
            "chat_jid": chat_jid,
            "message_id": message_id,
            "for_everyone": for_everyone
        }
        response = self.session.delete(
            f"{self.base_url}/messages/delete",
            json=payload
        )
        return response.json()
    
    def create_group(self, name, participants):
        """Create a new group"""
        payload = {
            "name": name,
            "participants": participants
        }
        response = self.session.post(
            f"{self.base_url}/groups/create",
            json=payload
        )
        return response.json()
    
    def join_group(self, invite_code):
        """Join a group using invite code"""
        payload = {
            "invite_code": invite_code
        }
        response = self.session.post(
            f"{self.base_url}/groups/join",
            json=payload
        )
        return response.json()
    
    def get_group_info(self, group_id):
        """Get group information"""
        response = self.session.get(
            f"{self.base_url}/groups/info",
            params={"group_id": group_id}
        )
        return response.json()
    
    def get_group_invite_link(self, group_id):
        """Get group invite link"""
        response = self.session.get(
            f"{self.base_url}/groups/invite-link",
            params={"group_id": group_id}
        )
        return response.json()
    
    def add_group_participants(self, group_id, users):
        """Add participants to a group"""
        payload = {
            "group_id": group_id,
            "users": users
        }
        response = self.session.post(
            f"{self.base_url}/groups/participants/add",
            json=payload
        )
        return response.json()
    
    def remove_group_participants(self, group_id, users):
        """Remove participants from a group"""
        payload = {
            "group_id": group_id,
            "users": users
        }
        response = self.session.post(
            f"{self.base_url}/groups/participants/remove",
            json=payload
        )
        return response.json()
    
    def get_contacts(self):
        """Get all contacts"""
        response = self.session.get(f"{self.base_url}/contacts")
        return response.json()
    
    def get_contact_info(self, jid):
        """Get specific contact information"""
        response = self.session.get(f"{self.base_url}/contacts/{jid}")
        return response.json()
    
    def upload_media(self, file_path):
        """Upload a media file"""
        with open(file_path, 'rb') as f:
            files = {'file': f}
            response = self.session.post(f"{self.base_url}/media/upload", files=files)
        return response.json()
    
    def set_status(self, status):
        """Set status message"""
        payload = {"status": status}
        response = self.session.post(f"{self.base_url}/status/set", json=payload)
        return response.json()
    
    def set_presence(self, presence, to_jid=None):
        """Set presence status"""
        payload = {"presence": presence}
        if to_jid:
            payload["to"] = to_jid
        response = self.session.post(f"{self.base_url}/presence/set", json=payload)
        return response.json()
    
    def listen_to_events(self, duration=60):
        """Listen to real-time events for a specified duration"""
        try:
            response = self.session.get(
                f"{self.base_url}/events",
                stream=True,
                timeout=duration
            )
            
            print(f"Listening to events for {duration} seconds...")
            start_time = time.time()
            
            for line in response.iter_lines():
                if time.time() - start_time > duration:
                    break
                    
                if line:
                    line = line.decode('utf-8')
                    if line.startswith('data: '):
                        try:
                            event_data = json.loads(line[6:])
                            print(f"Event: {json.dumps(event_data, indent=2)}")
                        except json.JSONDecodeError:
                            print(f"Raw event: {line}")
                            
        except requests.exceptions.RequestException as e:
            print(f"Error listening to events: {e}")

def main():
    """Example usage of the WhatsApp API client"""
    client = WhatsAppAPIClient()
    
    print("WhatsApp API Example Client")
    print("=" * 40)
    
    # Check server health
    print("\n1. Checking server health...")
    health = client.health_check()
    if health:
        print(f"Server status: {health}")
    else:
        print("Server is not responding. Make sure it's running on localhost:8080")
        return
    
    # Check authentication status
    print("\n2. Checking authentication status...")
    auth_status = client.get_auth_status()
    print(f"Auth status: {auth_status}")
    
    if not auth_status.get("data", {}).get("is_logged_in"):
        print("\n3. Getting QR code for authentication...")
        qr_data = client.get_qr_code()
        if qr_data.get("success"):
            print("Please scan the QR code with WhatsApp to authenticate")
            print("Waiting for authentication...")
            
            # Wait for authentication
            for i in range(30):  # Wait up to 30 seconds
                time.sleep(1)
                auth_status = client.get_auth_status()
                if auth_status.get("data", {}).get("is_logged_in"):
                    print("Successfully authenticated!")
                    break
            else:
                print("Authentication timeout. Please try again.")
                return
        else:
            print(f"Failed to get QR code: {qr_data}")
            return
    
    # Example: Send a message
    print("\n4. Example: Sending a message...")
    # Replace with actual JID
    recipient_jid = "1234567890@s.whatsapp.net"  # Replace with actual JID
    message_result = client.send_message(recipient_jid, "Hello from WhatsApp API!")
    print(f"Message result: {message_result}")
    
    # Example: Get contacts
    print("\n5. Example: Getting contacts...")
    contacts = client.get_contacts()
    if contacts.get("success"):
        contact_list = contacts.get("data", [])
        print(f"Found {len(contact_list)} contacts")
        for contact in contact_list[:3]:  # Show first 3 contacts
            print(f"  - {contact.get('name', 'Unknown')} ({contact.get('jid')})")
    
    # Example: Set status
    print("\n6. Example: Setting status...")
    status_result = client.set_status("Available via API")
    print(f"Status result: {status_result}")
    
    # Example: Listen to events for 10 seconds
    print("\n7. Example: Listening to events for 10 seconds...")
    client.listen_to_events(duration=10)
    
    print("\nExample completed!")

if __name__ == "__main__":
    main()