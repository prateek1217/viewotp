import React, { useState, useEffect, useRef } from 'react';
import axios from 'axios';
import './App.css';
import OTPCard from './components/OTPCard';
import TestPanel from './components/TestPanel';

function App() {
  const [otps, setOtps] = useState([]);
  const [workspace, setWorkspace] = useState(null);
  const [user, setUser] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [showTestPanel, setShowTestPanel] = useState(true);
  const [gmailConnected, setGmailConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [members, setMembers] = useState([]);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteLink, setInviteLink] = useState('');
  const wsRef = useRef(null);

  const API_BASE = 'http://localhost:8080';
  const WS_URL = 'ws://localhost:8080/ws';

  useEffect(() => {
    // Initialize app
    initializeApp();
    connectWebSocket();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const initializeApp = async () => {
    try {
      // Check health
      await axios.get(`${API_BASE}/health`);

      // Create a user first
      const userResponse = await axios.post(`${API_BASE}/api/users`, {
        name: 'Current User',
        email: 'admin@workspace.local',
      });
      setUser(userResponse.data);

      // Create or get workspace with the user as admin
      const wsResponse = await axios.post(`${API_BASE}/api/workspace`, {
        name: 'Team Workspace',
        owner_id: userResponse.data.id,
      });
      
      // Update user with workspace and admin role
      const updatedUser = {
        ...userResponse.data,
        workspace_id: wsResponse.data.id,
        role: 'admin',
      };
      setUser(updatedUser);
      setWorkspace(wsResponse.data);

      // Fetch initial OTPs
      fetchOTPs();
    } catch (error) {
      console.error('Initialization error:', error);
    }
  };

  const connectWebSocket = () => {
    try {
      const ws = new WebSocket(WS_URL);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
      };

      ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log('WebSocket message:', message);

        if (message.type === 'initial_otps') {
          setOtps(message.data || []);
        } else if (message.type === 'otp_received') {
          setOtps((prev) => {
            const exists = prev.some((otp) => otp.id === message.data.id);
            if (!exists) {
              return [message.data, ...prev];
            }
            return prev;
          });
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        // Attempt to reconnect after 3 seconds
        setTimeout(() => connectWebSocket(), 3000);
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('WebSocket connection error:', error);
    }
  };

  const fetchOTPs = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/otps`);
      setOtps(response.data || []);
    } catch (error) {
      console.error('Error fetching OTPs:', error);
    }
  };

  const handleOTPView = async (otpId) => {
    if (user) {
      try {
        await axios.post(`${API_BASE}/api/otps/view`, {
          otp_id: otpId,
          user_id: user.id,
        });
      } catch (error) {
        console.error('Error marking OTP as viewed:', error);
      }
    }
  };

  const handleCopyOTP = (otp) => {
    navigator.clipboard.writeText(otp);
    alert('OTP copied to clipboard!');
  };

  const handleConnectGmail = async () => {
    setIsConnecting(true);
    try {
      const response = await axios.get(`${API_BASE}/auth/login`);
      if (response.data.auth_url) {
        // Open Google login in new window
        window.open(response.data.auth_url, 'Gmail Login', 'width=500,height=600');
        setGmailConnected(true);
        alert('Gmail connected! OTPs will now appear automatically from your emails.');
      }
    } catch (error) {
      console.error('Error connecting Gmail:', error);
      alert('Failed to connect Gmail. Make sure you have set GOOGLE_CLIENT_ID in .env');
    } finally {
      setIsConnecting(false);
    }
  };

  const handleGetInviteLink = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/workspace/${workspace.id}/invite-link`);
      setInviteLink(response.data.invite_link);
      setShowInviteModal(true);
    } catch (error) {
      alert('Error getting invite link: ' + error.message);
    }
  };

  const handleCopyInviteLink = () => {
    navigator.clipboard.writeText(inviteLink);
    alert('Invite link copied to clipboard!');
  };

  const fetchWorkspaceMembers = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/workspace/${workspace.id}/members`);
      setMembers(response.data || []);
    } catch (error) {
      console.error('Error fetching members:', error);
    }
  };

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-content">
          <h1>🔐 Shared OTP Dashboard</h1>
          <div className="header-actions">
            <div className="header-status">
              <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}></div>
              <span>{isConnected ? 'Connected' : 'Reconnecting...'}</span>
            </div>
            {user && user.role === 'admin' && (
              <>
                <button 
                  className="invite-btn"
                  onClick={handleGetInviteLink}
                >
                  👥 Invite Members
                </button>
                <button 
                  className={`gmail-btn ${gmailConnected ? 'connected' : ''}`}
                  onClick={handleConnectGmail}
                  disabled={isConnecting}
                >
                  {isConnecting ? '🔄 Connecting...' : gmailConnected ? '✅ Gmail Connected' : '📧 Connect Gmail'}
                </button>
              </>
            )}
            {user && user.role === 'member' && (
              <span className="member-badge">👤 Member</span>
            )}
          </div>
        </div>
      </header>

      <main className="app-main">
        {workspace && (
          <div className="workspace-info">
            <h2>{workspace.name}</h2>
            <p>Workspace ID: {workspace.id}</p>
          </div>
        )}

        <div className="otps-container">
          {otps.length === 0 ? (
            <div className="empty-state">
              <p>No OTPs received yet</p>
              <small>OTPs will appear here when received</small>
            </div>
          ) : (
            <div className="otps-grid">
              {otps.map((otp) => (
                <OTPCard
                  key={otp.id}
                  otp={otp}
                  onView={() => handleOTPView(otp.id)}
                  onCopy={() => handleCopyOTP(otp.otp)}
                />
              ))}
            </div>
          )}
        </div>
      </main>

      {showTestPanel && (
        <TestPanel
          onClose={() => setShowTestPanel(false)}
          workspace={workspace}
        />
      )}

      {showInviteModal && (
        <div className="modal-overlay" onClick={() => setShowInviteModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>👥 Invite Team Members</h3>
              <button className="modal-close" onClick={() => setShowInviteModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <p>Share this link with your team members:</p>
              <div className="invite-link-box">
                <input type="text" value={inviteLink} readOnly />
                <button className="copy-link-btn" onClick={handleCopyInviteLink}>
                  📋 Copy Link
                </button>
              </div>
              <p className="invite-info">
                Team members can use this link to join the workspace and view OTPs without needing a Gmail account.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
