# 🔐 Shared OTP Dashboard

A centralized dashboard for teams to view and manage OTP codes from shared accounts. Perfect for startups and teams using collective subscriptions.

## 🎯 Problem It Solves

When a team shares accounts (AWS, social media, testing services, etc.), OTP codes get sent to the account owner's email. If the owner is offline, other team members are stuck waiting.

**Solution:** This dashboard automatically extracts OTPs from Gmail and displays them in real-time to all workspace members. No more "Can someone send the OTP?" messages! 📱✅

## ✨ Features

- ✅ **Real-time OTP Display** - WebSocket-powered instant updates
- ✅ **Google OAuth Integration** - Secure Gmail access via OAuth2
- ✅ **Workspace Sharing** - Share dashboard with team members via invite links
- ✅ **Role-Based Access** - Admin (manage Gmail) & Member (view OTPs only)
- ✅ **Auto OTP Extraction** - Regex-based parsing for common OTP formats
- ✅ **Live Countdown** - Shows OTP expiration time (default: 5 minutes)
- ✅ **Activity Tracking** - Logs who viewed OTPs
- ✅ **In-Memory Storage** - Fast, no database needed
- ✅ **One-Click Copy** - Copy OTP to clipboard instantly
- ✅ **Test Panel** - Simulate OTPs for testing without Gmail

## 🛠 Tech Stack

- **Backend:** Go (Gin framework, WebSocket)
- **Frontend:** React 18 (real-time updates)
- **Real-time:** WebSocket for instant OTP push
- **Storage:** In-memory (no database)
- **Authentication:** Google OAuth2
- **Gmail Integration:** Gmail API v1
- **Development:** Hot-reload enabled

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Node.js 16+
- npm or yarn
- Google Cloud account (for Gmail API credentials)

### 1️⃣ Clone & Setup

```bash
git clone https://github.com/yourusername/otp-dashboard.git
cd otp-dashboard

# Copy environment template
cp .env.example .env

# Edit .env with your Google OAuth credentials
nano .env
```

### 2️⃣ Start Backend

```bash
go mod download
go run main.go
```

Server runs on **http://localhost:8080**

### 3️⃣ Start Frontend (new terminal)

```bash
npm install
npm start
```

Dashboard opens at **http://localhost:3000**

### 4️⃣ Use It!

1. Click **"📧 Connect Gmail"** to authenticate with Google
2. Grant Gmail read access
3. Dashboard automatically fetches OTPs from emails
4. Share workspace via **"👥 Invite Members"** link
5. Team members can see OTPs without needing Gmail access

## 🔌 API Endpoints

### Workspaces
```
POST   /api/workspace                    - Create workspace
GET    /api/workspace/:id/members        - Get workspace members
GET    /api/workspace/:id/invite-link    - Get invite link
POST   /api/workspace/join/:code         - Join workspace with invite
```

### OTPs (Real-time via Gmail)
```
GET    /api/otps                         - Get all current OTPs
POST   /api/otps/simulate                - Test OTP (for testing)
POST   /api/otps/manual                  - Manually add OTP
POST   /api/otps/view                    - Mark OTP as viewed
```

### Users
```
POST   /api/users                        - Register user
GET    /api/users/:id                    - Get user details
```

### Authentication
```
GET    /auth/login                       - Get Google OAuth URL
GET    /auth/callback                    - OAuth callback
GET    /                                 - OAuth redirect handler
```

### Real-time
```
WS     /ws                               - WebSocket for live OTP updates
GET    /health                           - Health check
```

## 📁 Project Structure

```
otp-dashboard/
├── main.go                         # Go backend (Gmail API, WebSocket)
├── go.mod                          # Go dependencies
├── go.sum                          # Dependency checksums
├── package.json                    # React dependencies
├── .env.example                    # Environment template (RENAME TO .env)
├── .gitignore                      # Git ignore rules
├── README.md                       # This file
├── STARTUP.md                      # Local development guide
├── GMAIL_SETUP.md                  # Gmail OAuth setup
├── public/
│   └── index.html                  # HTML entry point
└── src/
    ├── index.js                    # React entry point
    ├── index.css                   # Global styles
    ├── App.js                      # Main dashboard component
    ├── App.css                      # Dashboard styles
    └── components/
        ├── OTPCard.js              # OTP display component
        ├── OTPCard.css             # OTP card styles
        ├── TestPanel.js            # Test panel for simulating OTPs
        └── TestPanel.css           # Test panel styles
```

## 🔄 How It Works

### Architecture

```
┌─────────────┐
│   Browser   │
│  (React)    │
└──────┬──────┘
       │ WebSocket
       ▼
┌─────────────┐       ┌──────────────┐
│   Backend   │──────▶│  Gmail API   │
│   (Go)      │       │   (OAuth2)   │
└─────────────┘       └──────────────┘
       │
       ▼ (In-memory)
   ┌───────┐
   │ OTPs  │
   └───────┘
```

### OTP Extraction Flow

```
1. User connects Gmail via OAuth
   ↓
2. Backend receives access token
   ↓
3. Every 30 seconds: Query Gmail for unread emails
   ↓
4. Find emails with OTP keywords (code, verification, etc.)
   ↓
5. Extract 4-8 digit numbers using regex
   ↓
6. Save to in-memory store with 5-min expiry
   ↓
7. Broadcast via WebSocket to all connected clients
   ↓
8. Dashboard displays OTP with countdown timer
```

### Sharing with Team

```
Admin (Gmail Owner)           Team Members
│                             │
├─ Connects Gmail OAuth       │
├─ OTPs auto-extracted        │
├─ Clicks "Invite Members"    │
├─ Shares invite link ─────────▶ Members open link
│                             │ Members join workspace
│                             │ Members see dashboard
└─ OTPs visible to all ◄──────┘ Members view OTPs (read-only)
```

## 🔒 Security & Privacy

- ✅ **OAuth2**: Secure Google authentication, no password storage
- ✅ **In-Memory Only**: OTPs never persisted to disk
- ✅ **Auto-Expiry**: OTPs deleted after 5 minutes
- ✅ **Activity Logs**: Track who viewed each OTP
- ✅ **Role-Based Access**: Members can only view, not connect Gmail
- ✅ **.env in .gitignore**: Credentials never committed
- ✅ **HTTPS Ready**: Deploy with SSL/TLS in production

## 🧪 Testing Without Gmail

The dashboard includes a **Test Panel** for development:

```
Bottom-right corner:
┌─────────────────────────────┐
│ 🧪 Test OTP Simulation      │
├─────────────────────────────┤
│ Service: [Amazon          ] │
│ OTP:     [527341          ] │
│                             │
│      [Send OTP]             │
└─────────────────────────────┘
```

Use this to test without sending real emails.

## 🚀 Deployment

### Production Checklist

- [ ] Set `GIN_MODE=release` environment variable
- [ ] Use a real database (PostgreSQL recommended)
- [ ] Enable HTTPS/SSL certificates
- [ ] Set up proper CORS for your domain
- [ ] Configure Gmail webhook notifications (optional, uses polling by default)
- [ ] Add rate limiting
- [ ] Set up monitoring & alerting

### Deploy to Railway or Render

1. Push to GitHub
2. Connect repository
3. Set environment variables (GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET)
4. Deploy frontend to Vercel, backend to Railway/Render

## 💡 Use Cases

- ✅ Shared AWS accounts in startups
- ✅ Team social media accounts
- ✅ Collective cloud service subscriptions
- ✅ Shared testing accounts
- ✅ Multi-user SaaS dashboards

## 🤝 Contributing

Found a bug or have a feature idea?

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## 📝 License

MIT - Feel free to use, modify, and share!

---


**Developed by Prateek Khandelwal**
