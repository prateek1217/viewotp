# 🚀 Local Development Guide

Get the **Shared OTP Dashboard** running on your machine!

## Prerequisites

- **Go** 1.21+ - [Download](https://golang.org/dl/)
- **Node.js** 16+ - [Download](https://nodejs.org/)
- **npm** or **yarn**
- **Git**

## 📋 Setup Steps

### 1️⃣ Clone the Repository

```bash
git clone https://github.com/yourusername/otp-dashboard.git
cd otp-dashboard
```

### 2️⃣ Setup Environment Variables

```bash
# Copy the template
cp .env.example .env

# Edit with your Google OAuth credentials
nano .env  # or your preferred editor
```

**Fill in .env:**
```
GOOGLE_CLIENT_ID=your_client_id_here
GOOGLE_CLIENT_SECRET=your_client_secret_here
BACKEND_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000
GMAIL_USER=your_email@gmail.com
```

See `GMAIL_SETUP.md` for how to get these credentials.

### 3️⃣ Start Backend (Terminal 1)

```bash
# Download dependencies
go mod download

# Run backend
go run main.go
```

**Expected output:**
```
✅ Gmail OAuth configured successfully
🚀 Starting server on :8080
📧 Gmail integration available
Listening and serving HTTP on :8080
```

### 4️⃣ Start Frontend (Terminal 2)

```bash
# Install dependencies
npm install

# Start dev server
npm start
```

**Opens automatically at:** http://localhost:3000

## 🎮 Using the Dashboard

### First Time Setup

1. **Go to** http://localhost:3000
2. **Click** "📧 Connect Gmail" button
3. **Authorize** with your Google account
4. **Grant** Gmail read permissions

### Testing (Without Real Emails)

Use the **Test Panel** (bottom-right):

```
Service: Amazon
OTP: 527341
[Send OTP]
```

### Share with Team

1. **Click** "👥 Invite Members"
2. **Copy** the invite link
3. **Share** with team members
4. They join and see the same OTPs!

## 📊 Dashboard Features

| Feature | How to Use |
|---------|-----------|
| **Real-time OTPs** | OTPs auto-appear from Gmail |
| **Copy to Clipboard** | Click the copy button on any OTP card |
| **Countdown Timer** | See when OTPs expire (5 minutes) |
| **Activity Log** | Track who viewed each OTP |
| **Workspace Invite** | Share link with team members |
| **Test Panel** | Simulate OTPs without real emails |

## 🔍 Debugging

### Check if backend is running:

```bash
curl http://localhost:8080/health
```

**Expected:** `{"status":"ok"}`

### View backend logs in real-time:

```bash
# Terminal 1 shows all logs
# Look for [GMAIL FETCH] messages
```

### Check browser console errors:

1. Open http://localhost:3000
2. Press **F12** or **Ctrl+Shift+I**
3. Go to **Console** tab

### Common Issues

| Issue | Fix |
|-------|-----|
| Port 8080 in use | `lsof -i :8080` then kill process |
| Port 3000 in use | Kill Node process or use different port |
| Gmail not connecting | Check .env credentials |
| OTPs not showing | Wait 30 seconds, check console logs |
| WebSocket errors | Refresh page, restart both servers |

## 📁 Key Files to Know

```
main.go                   - All backend logic
src/App.js               - Dashboard component
src/components/OTPCard.js - OTP display component
.env                     - Your credentials (DON'T COMMIT!)
.gitignore               - Files to ignore in git
```

## 🚀 Next Steps

1. **Connect Gmail** - See real OTPs from your inbox
2. **Invite teammates** - Test the workspace sharing
3. **Deploy** - Push to GitHub, deploy to production
4. **Monitor** - Check activity logs to see who's using it

## 📝 Development Tips

### Auto-reload on code changes

- **Backend**: Go compiler auto-reloads... just save and re-run
- **Frontend**: React auto-refreshes... changes appear instantly

### Test different roles

1. Open **Tab 1** - Admin (connected to Gmail)
2. Open **Tab 2** - Member (joined via invite link)
3. See how they view the same OTPs!

### Enable verbose logging

Add `DEBUG=*` before commands:
```bash
DEBUG=* npm start
```

## 🔒 Security Reminders

- ✅ **Never commit .env** - It's in .gitignore
- ✅ **Use HTTPS in production** - Not HTTP
- ✅ **Rotate credentials** - If accidentally exposed
- ✅ **Test in private window** - For clean sessions

## 🆘 Need Help?

1. Check error messages in console/logs
2. Read the README.md for architecture
3. See GMAIL_SETUP.md for OAuth issues
4. Create an issue on GitHub

---

**Happy coding! 🎉**
