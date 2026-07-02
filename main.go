package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OTP represents a single OTP entry
type OTP struct {
	ID        string    `json:"id"`
	OTPCode   string    `json:"otp"`
	Sender    string    `json:"sender"`
	ReceivedAt time.Time `json:"received_at"`
	ExpiresAt time.Time `json:"expires_at"`
	ViewedBy  []string  `json:"viewed_by"`
}

// Workspace represents a shared workspace
type Workspace struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	OwnerID    string    `json:"owner_id"`
	Members    []string  `json:"members"`
	InviteCode string    `json:"invite_code"`
	CreatedAt  time.Time `json:"created_at"`
}

// User represents a user
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	WorkspaceID string    `json:"workspace_id"`
	Role        string    `json:"role"` // "admin" or "member"
	CreatedAt   time.Time `json:"created_at"`
}

// Member represents a workspace member with role
type Member struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"` // "admin" or "member"
	JoinedAt time.Time `json:"joined_at"`
}

var (
	// In-memory storage
	otps       = make(map[string]*OTP)
	workspaces = make(map[string]*Workspace)
	users      = make(map[string]*User)
	members    = make(map[string][]Member) // workspace_id -> members
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// WebSocket hub
	wsClients = make(map[*websocket.Conn]bool)
	wsMutex   sync.RWMutex
	
	// Upgrader for WebSocket
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for local development
		},
	}

	// OTP expiration time (5 minutes)
	otpExpirationTime = 5 * time.Minute
)

// Initialize environment
func init() {
	godotenv.Load()
}

// Gmail access token storage
var (
	gmailToken        string
	tokenMutex        sync.RWMutex
	googleOAuthConfig *oauth2.Config
)

// Initialize Gmail OAuth
func initializeGmailOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Println("⚠️  WARNING: Google OAuth credentials not configured")
		log.Println("Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in .env file")
		return
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8080",
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		Endpoint: google.Endpoint,
	}

	log.Println("✅ Gmail OAuth configured successfully")
}

// Broadcast OTP updates to all connected WebSocket clients
func broadcastOTP(otp *OTP) {
	wsMutex.RLock()
	defer wsMutex.RUnlock()

	message := map[string]interface{}{
		"type": "otp_received",
		"data": otp,
	}

	data, _ := json.Marshal(message)

	for client := range wsClients {
		client.WriteMessage(websocket.TextMessage, data)
	}
}

// Extract OTP from email body using regex
func extractOTP(emailBody string) string {
	// Priority: look for common OTP patterns
	patterns := []string{
		`(?i)(?:otp|code|verification|passcode|security)[\s:]*(\d{4,8})`,
		`\b(\d{6})\b`, // 6-digit code
		`\b(\d{4,8})\b`, // 4-8 digit code
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(emailBody)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// Extract sender name from email address
func extractSender(email string) string {
	// Extract domain or name from email
	re := regexp.MustCompile(`([a-zA-Z0-9]+)(?:@|\.)`)
	matches := re.FindStringSubmatch(email)
	if len(matches) > 1 {
		return matches[1]
	}
	return email
}

// Save OTP to memory
func saveOTP(sender, otpCode string) {
	mu.Lock()
	defer mu.Unlock()

	otp := &OTP{
		ID:         fmt.Sprintf("otp_%d", time.Now().Unix()),
		OTPCode:    otpCode,
		Sender:     sender,
		ReceivedAt: time.Now(),
		ExpiresAt:  time.Now().Add(otpExpirationTime),
		ViewedBy:   []string{},
	}

	otps[otp.ID] = otp
	log.Printf("✅ OTP received from %s: %s", sender, otpCode)

	// Broadcast to WebSocket clients
	go broadcastOTP(otp)
}

// Start fetching OTPs from Gmail
func startGmailFetching(token *oauth2.Token) {
	log.Println("🚀 [GMAIL FETCHER] Starting Gmail OTP fetcher with polling every 30 seconds...")
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("🔄 [GMAIL FETCHER] Polling Gmail for new OTPs...")
		fetchGmailOTPs(token)
	}
}

// Fetch OTPs from Gmail
func fetchGmailOTPs(token *oauth2.Token) {
	if googleOAuthConfig == nil {
		return
	}

	log.Println("📧 [GMAIL FETCH] Starting Gmail OTP fetch...")

	// Create Gmail service
	client := googleOAuthConfig.Client(context.Background(), token)
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Printf("❌ [GMAIL FETCH] Error creating Gmail service: %v", err)
		return
	}

	// Query for unread emails with OTP-related keywords
	query := "is:unread subject:(code OR otp OR verification OR passcode OR authenticate)"
	log.Printf("📧 [GMAIL FETCH] Querying: %s", query)
	
	results, err := gmailService.Users.Messages.List("me").
		Q(query).
		MaxResults(10).
		Do()

	if err != nil {
		log.Printf("❌ [GMAIL FETCH] Error fetching Gmail messages: %v", err)
		return
	}

	if results == nil || len(results.Messages) == 0 {
		log.Println("📧 [GMAIL FETCH] No unread OTP emails found")
		return
	}

	log.Printf("📧 [GMAIL FETCH] Found %d potential OTP emails", len(results.Messages))

	for _, msg := range results.Messages {
		message, err := gmailService.Users.Messages.Get("me", msg.Id).Format("full").Do()
		if err != nil {
			log.Printf("❌ [GMAIL FETCH] Error getting message: %v", err)
			continue
		}

		// Extract sender and subject
		headers := message.Payload.Headers
		var sender, subject string
		for _, header := range headers {
			if header.Name == "From" {
				sender = extractSender(header.Value)
			}
			if header.Name == "Subject" {
				subject = header.Value
			}
		}

		log.Printf("📧 [GMAIL FETCH] Processing email from %s, subject: %s", sender, subject)

		// Get email body
		var body string
		if message.Payload.Body != nil && message.Payload.Body.Data != "" {
			decoded, _ := base64.URLEncoding.DecodeString(message.Payload.Body.Data)
			body = string(decoded)
		} else if message.Payload.Parts != nil {
			for _, part := range message.Payload.Parts {
				if part.MimeType == "text/plain" && part.Body != nil {
					decoded, _ := base64.URLEncoding.DecodeString(part.Body.Data)
					body = string(decoded)
					break
				}
			}
		}

		log.Printf("📧 [GMAIL FETCH] Email body (first 200 chars): %.200s", body)

		// Extract OTP
		otp := extractOTP(subject + " " + body)
		if otp != "" {
			log.Printf("✅ [GMAIL FETCH] Extracted OTP from %s: %s", sender, otp)
			saveOTP(sender, otp)
		} else {
			log.Printf("⚠️  [GMAIL FETCH] No OTP pattern matched in email from %s", sender)
		}
	}
}

// Clean up expired OTPs
func cleanupExpiredOTPs() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		now := time.Now()
		for id, otp := range otps {
			if now.After(otp.ExpiresAt) {
				delete(otps, id)
			}
		}
		mu.Unlock()
	}
}

// Generate random invite code
func generateInviteCode() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := ""
	for i := 0; i < 8; i++ {
		code += string(chars[int(time.Now().Unix())%len(chars)])
	}
	return code
}

// Handler: Create workspace
func createWorkspace(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		OwnerID string `json:"owner_id" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	inviteCode := generateInviteCode()
	workspace := &Workspace{
		ID:         fmt.Sprintf("ws_%d", time.Now().Unix()),
		Name:       req.Name,
		OwnerID:    req.OwnerID,
		Members:    []string{req.OwnerID},
		InviteCode: inviteCode,
		CreatedAt:  time.Now(),
	}

	workspaces[workspace.ID] = workspace
	
	// Add owner as admin member
	members[workspace.ID] = []Member{
		{
			UserID:   req.OwnerID,
			Name:     "Admin",
			Email:    "admin@workspace",
			Role:     "admin",
			JoinedAt: time.Now(),
		},
	}

	log.Printf("✅ Workspace created: %s (Code: %s)", workspace.Name, inviteCode)
	c.JSON(200, workspace)
}

// Handler: Add member to workspace
func addMember(c *gin.Context) {
	var req struct {
		WorkspaceID string `json:"workspace_id" binding:"required"`
		UserID      string `json:"user_id" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if ws, exists := workspaces[req.WorkspaceID]; exists {
		ws.Members = append(ws.Members, req.UserID)
		c.JSON(200, ws)
	} else {
		c.JSON(404, gin.H{"error": "workspace not found"})
	}
}

// Handler: Get all OTPs
func getOTPs(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	var otpList []*OTP
	for _, otp := range otps {
		otpList = append(otpList, otp)
	}

	c.JSON(200, otpList)
}

// Handler: Simulate receiving OTP (for testing)
func simulateOTP(c *gin.Context) {
	var req struct {
		Sender string `json:"sender" binding:"required"`
		OTP    string `json:"otp" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	otp := &OTP{
		ID:         fmt.Sprintf("otp_%d", time.Now().Unix()),
		OTPCode:    req.OTP,
		Sender:     req.Sender,
		ReceivedAt: time.Now(),
		ExpiresAt:  time.Now().Add(otpExpirationTime),
		ViewedBy:   []string{},
	}

	otps[otp.ID] = otp

	// Broadcast to all connected clients
	go broadcastOTP(otp)

	c.JSON(200, otp)
}

// Handler: Manually add OTP (for when you receive it but want to add it manually)
func manualAddOTP(c *gin.Context) {
	var req struct {
		Sender string `json:"sender" binding:"required"`
		OTP    string `json:"otp" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	saveOTP(req.Sender, req.OTP)
	c.JSON(200, gin.H{"success": true, "message": "OTP added to dashboard"})
}

// Handler: Mark OTP as viewed
func viewOTP(c *gin.Context) {
	var req struct {
		OTPID  string `json:"otp_id" binding:"required"`
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if otp, exists := otps[req.OTPID]; exists {
		// Add user to viewed list if not already there
		alreadyViewed := false
		for _, uid := range otp.ViewedBy {
			if uid == req.UserID {
				alreadyViewed = true
				break
			}
		}

		if !alreadyViewed {
			otp.ViewedBy = append(otp.ViewedBy, req.UserID)
		}

		c.JSON(200, otp)
	} else {
		c.JSON(404, gin.H{"error": "otp not found"})
	}
}

// WebSocket handler
func wsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	wsMutex.Lock()
	wsClients[conn] = true
	wsMutex.Unlock()

	// Send current OTPs to newly connected client
	mu.RLock()
	var otpList []*OTP
	for _, otp := range otps {
		otpList = append(otpList, otp)
	}
	mu.RUnlock()

	message := map[string]interface{}{
		"type": "initial_otps",
		"data": otpList,
	}
	data, _ := json.Marshal(message)
	conn.WriteMessage(websocket.TextMessage, data)

	// Keep connection open
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wsMutex.Lock()
			delete(wsClients, conn)
			wsMutex.Unlock()
			break
		}
	}
}

// Gmail webhook handler (placeholder for actual Gmail webhook)
func gmailWebhookHandler(c *gin.Context) {
	// In production, this would handle Gmail push notifications
	// For now, we'll use the simulateOTP endpoint for testing

	c.JSON(200, gin.H{"status": "webhook received"})
}

// Register user
func registerUser(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Email       string `json:"email" binding:"required"`
		WorkspaceID string `json:"workspace_id"`
		Role        string `json:"role"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Default to member role
	role := req.Role
	if role == "" {
		role = "member"
	}

	user := &User{
		ID:          fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Email:       req.Email,
		WorkspaceID: req.WorkspaceID,
		Role:        role,
		CreatedAt:   time.Now(),
	}

	users[user.ID] = user

	// Add user to workspace members if workspace exists
	if req.WorkspaceID != "" {
		if _, exists := workspaces[req.WorkspaceID]; exists {
			if workspaceMembers, exists := members[req.WorkspaceID]; exists {
				workspaceMembers = append(workspaceMembers, Member{
					UserID:   user.ID,
					Name:     req.Name,
					Email:    req.Email,
					Role:     role,
					JoinedAt: time.Now(),
				})
				members[req.WorkspaceID] = workspaceMembers
			}
		}
	}

	log.Printf("✅ User registered: %s (%s)", req.Email, role)
	c.JSON(200, user)
}

// Get user
func getUser(c *gin.Context) {
	userID := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	if user, exists := users[userID]; exists {
		c.JSON(200, user)
	} else {
		c.JSON(404, gin.H{"error": "user not found"})
	}
}

// Handler: Get workspace members
func getWorkspaceMembers(c *gin.Context) {
	workspaceID := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	if _, exists := workspaces[workspaceID]; !exists {
		c.JSON(404, gin.H{"error": "workspace not found"})
		return
	}

	if ws, exists := members[workspaceID]; exists {
		c.JSON(200, ws)
	} else {
		c.JSON(200, []Member{})
	}
}

// Handler: Get invite link for workspace
func getInviteLink(c *gin.Context) {
	workspaceID := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	workspace, exists := workspaces[workspaceID]
	if !exists {
		c.JSON(404, gin.H{"error": "workspace not found"})
		return
	}

	inviteLink := fmt.Sprintf("http://localhost:3000/join/%s", workspace.InviteCode)
	c.JSON(200, gin.H{
		"invite_code": workspace.InviteCode,
		"invite_link": inviteLink,
	})
}

// Handler: Join workspace with invite code
func joinWorkspace(c *gin.Context) {
	inviteCode := c.Param("code")

	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Find workspace by invite code
	var workspace *Workspace
	for _, ws := range workspaces {
		if ws.InviteCode == inviteCode {
			workspace = ws
			break
		}
	}

	if workspace == nil {
		c.JSON(404, gin.H{"error": "invalid invite code"})
		return
	}

	// Create user
	user := &User{
		ID:          fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Email:       req.Email,
		WorkspaceID: workspace.ID,
		Role:        "member",
		CreatedAt:   time.Now(),
	}

	users[user.ID] = user

	// Add to workspace
	workspace.Members = append(workspace.Members, user.ID)
	if ws, exists := members[workspace.ID]; exists {
		ws = append(ws, Member{
			UserID:   user.ID,
			Name:     req.Name,
			Email:    req.Email,
			Role:     "member",
			JoinedAt: time.Now(),
		})
		members[workspace.ID] = ws
	}

	log.Printf("✅ Member joined workspace: %s (%s)", req.Email, workspace.Name)
	c.JSON(200, gin.H{
		"user":      user,
		"workspace": workspace,
		"message":   "Successfully joined workspace",
	})
}

// OAuth Login endpoint
func oauthLogin(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		c.JSON(400, gin.H{"error": "Google OAuth not configured. Set GOOGLE_CLIENT_ID in .env"})
		return
	}

	// Use just the origin, no path
	redirectURI := "http://localhost:8080"
	
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent",
		clientID,
		url.QueryEscape(redirectURI),
		url.QueryEscape("https://www.googleapis.com/auth/gmail.readonly"),
	)
	
	c.JSON(200, gin.H{"auth_url": authURL})
}

// OAuth Callback endpoint - Google redirects back to root
func oauthCallback(c *gin.Context) {
	code := c.Query("code")
	errorMsg := c.Query("error")
	
	// If user denied permission
	if errorMsg != "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Gmail Connection Error</title>
				<style>
					body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
					.container { background: white; padding: 2rem; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); }
					h1 { color: #ff4444; margin-top: 0; }
					p { color: #666; }
					a { color: #667eea; text-decoration: none; font-weight: bold; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>❌ Connection Failed</h1>
					<p>Error: %s</p>
					<p><a href="http://localhost:3000">← Go back to dashboard</a></p>
				</div>
			</body>
			</html>
		`, errorMsg)
		return
	}
	
	if code == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Gmail Connection Error</title>
				<style>
					body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
					.container { background: white; padding: 2rem; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); }
					h1 { color: #ff4444; margin-top: 0; }
					p { color: #666; }
					a { color: #667eea; text-decoration: none; font-weight: bold; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>❌ Missing Authorization Code</h1>
					<p>Google did not return an authorization code.</p>
					<p><a href="http://localhost:3000">← Go back to dashboard</a></p>
				</div>
			</body>
			</html>
		`)
		return
	}

	log.Printf("✅ OAuth code received: %s...", code[:20])
	
	// Exchange code for token
	if googleOAuthConfig == nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Gmail Connection Error</title>
				<style>
					body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
					.container { background: white; padding: 2rem; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); }
					h1 { color: #ff4444; margin-top: 0; }
					p { color: #666; }
					a { color: #667eea; text-decoration: none; font-weight: bold; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>❌ OAuth Not Configured</h1>
					<p>Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in .env file</p>
					<p><a href="http://localhost:3000">← Go back to dashboard</a></p>
				</div>
			</body>
			</html>
		`)
		return
	}

	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("❌ Error exchanging token: %v", err)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Gmail Connection Error</title>
				<style>
					body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
					.container { background: white; padding: 2rem; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); }
					h1 { color: #ff4444; margin-top: 0; }
					p { color: #666; }
					a { color: #667eea; text-decoration: none; font-weight: bold; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>❌ Token Exchange Failed</h1>
					<p>Could not get access token from Google</p>
					<p><a href="http://localhost:3000">← Go back to dashboard</a></p>
				</div>
			</body>
			</html>
		`)
		return
	}

	// Store the token
	tokenMutex.Lock()
	gmailToken = token.AccessToken
	tokenMutex.Unlock()

	// Start fetching OTPs from Gmail
	go startGmailFetching(token)

	// Show success page and redirect
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Gmail Connected!</title>
			<style>
				body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
				.container { background: white; padding: 2rem; border-radius: 12px; text-align: center; max-width: 400px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); }
				h1 { color: #4caf50; margin-top: 0; }
				p { color: #666; margin: 1rem 0; }
				.code { background: #f0f0f0; padding: 1rem; border-radius: 6px; font-family: monospace; color: #333; word-break: break-all; margin: 1rem 0; }
				a { color: white; background: #667eea; padding: 0.8rem 1.5rem; border-radius: 6px; text-decoration: none; font-weight: bold; display: inline-block; margin-top: 1rem; }
				a:hover { background: #764ba2; }
			</style>
			<script>
				setTimeout(function() {
					window.location.href = 'http://localhost:3000?gmail_connected=true';
				}, 3000);
			</script>
		</head>
		<body>
			<div class="container">
				<h1>✅ Gmail Connected!</h1>
				<p>Your Gmail account has been successfully connected.</p>
				<p>The dashboard will receive OTP codes from your Gmail inbox.</p>
				<p>Redirecting in 3 seconds...</p>
				<a href="http://localhost:3000?gmail_connected=true">← Go to Dashboard</a>
			</div>
		</body>
		</html>
	`)
}

func main() {
	// Enable CORS for local development
	r := gin.Default()

	// Initialize Gmail OAuth
	initializeGmailOAuth()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Start cleanup goroutine
	go cleanupExpiredOTPs()

	// OAuth routes - MUST be before other routes
	r.GET("/", func(c *gin.Context) {
		code := c.Query("code")
		if code != "" {
			oauthCallback(c)
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/auth/login", oauthLogin)
	r.GET("/auth/callback", oauthCallback)

	// Routes
	r.POST("/api/workspace", createWorkspace)
	r.POST("/api/workspace/add-member", addMember)
	r.GET("/api/workspace/:id/members", getWorkspaceMembers)
	r.GET("/api/workspace/:id/invite-link", getInviteLink)
	r.POST("/api/workspace/join/:code", joinWorkspace)
	r.GET("/api/otps", getOTPs)
	r.POST("/api/otps/simulate", simulateOTP) // For testing
	r.POST("/api/otps/manual", manualAddOTP) // Manual OTP entry
	r.POST("/api/otps/view", viewOTP)
	r.POST("/api/users", registerUser)
	r.GET("/api/users/:id", getUser)
	r.POST("/webhook/gmail", gmailWebhookHandler)
	r.GET("/ws", wsHandler)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := ":8080"
	log.Printf("🚀 Starting server on %s", port)
	log.Printf("📧 Gmail integration available")
	r.Run(port)
}
