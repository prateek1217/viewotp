# 📧 Gmail Setup Guide

## How to Connect Your Gmail Account

### **Prerequisites**

You need Google OAuth credentials. Follow these steps:

### **Step 1: Create Google Cloud Project**

1. Go to: https://console.cloud.google.com
2. Click on **Select a Project** (top-left)
3. Click **NEW PROJECT**
4. Name: `OTP Dashboard`
5. Click **CREATE**

### **Step 2: Enable Gmail API**

1. In the left sidebar, go to **APIs & Services** → **Library**
2. Search for **"Gmail API"**
3. Click on it and press **ENABLE**

### **Step 3: Create OAuth Credentials**

1. Go to **APIs & Services** → **Credentials**
2. Click **+ CREATE CREDENTIALS** → **OAuth client ID**
3. If prompted, set up OAuth consent screen:
   - **User Type:** External
   - **App name:** OTP Dashboard
   - **User support email:** Your email
   - Add yourself as a test user
   - Save and continue
4. Back to Credentials, click **+ CREATE CREDENTIALS** → **OAuth client ID**
5. **Application type:** Web application
6. **Authorized redirect URIs:** Add these:
   ```
   http://localhost:8080/auth/callback
   http://localhost:3000
   ```
7. Click **CREATE**
8. Copy the **Client ID** and **Client Secret**

### **Step 4: Add Credentials to .env**

Open `.env` file in project root:

```
GOOGLE_CLIENT_ID=YOUR_CLIENT_ID_HERE
GOOGLE_CLIENT_SECRET=YOUR_CLIENT_SECRET_HERE
BACKEND_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000
GMAIL_USER=your_email@gmail.com
```

Replace with your actual values from Google Cloud.

### **Step 5: Restart Backend**

1. Stop the backend (Ctrl+C)
2. Run: `go run main.go`
3. Should see: "📧 Gmail integration available"

### **Step 6: Connect on Dashboard**

1. Go to http://localhost:3000
2. Click **"📧 Connect Gmail"** button (top-right)
3. A Google login popup appears
4. Sign in with your Gmail account
5. Grant permissions when prompted
6. Button turns green: **"✅ Gmail Connected"**

### **Step 7: Test It**

Send yourself an OTP email (from any service):
- Amazon verification code
- Google authentication code
- GitHub OTP
- Any service that sends 4-8 digit codes

The OTP will appear on your dashboard automatically within 1 minute!

---

## **Troubleshooting**

### **"Failed to connect Gmail"**
- Check .env file has correct `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`
- Restart backend after adding credentials
- Make sure Gmail API is enabled in Google Cloud

### **OTPs not appearing**
- Wait 1 minute (backend checks every 60 seconds)
- Check that Gmail has "Label as unread" or "Mark as unread"
- Send yourself a test email with a 6-digit code

### **Popup blocked**
- Allow popups for localhost:3000 in browser settings
- Or check browser's popup blocker

---

## **How It Works**

```
1. You click "Connect Gmail"
2. Browser opens Google login
3. You authenticate
4. Backend gets permission to read Gmail
5. Every 60 seconds, backend:
   - Fetches latest emails
   - Searches for OTP patterns
   - Extracts codes (6-8 digits)
   - Adds to dashboard
   - WebSocket notifies all connected clients
6. Dashboard shows OTP instantly with countdown (5 min expiry)
```

---

**Ready to connect? Let's go! 🚀**
