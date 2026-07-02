import React, { useState } from 'react';
import axios from 'axios';
import './TestPanel.css';

function TestPanel({ onClose, workspace }) {
  const [sender, setSender] = useState('Amazon');
  const [otp, setOtp] = useState('527341');
  const [isLoading, setIsLoading] = useState(false);
  const [tabMode, setTabMode] = useState('simulate'); // 'simulate' or 'manual'

  const API_BASE = 'http://localhost:8080';

  const handleSimulateOTP = async () => {
    if (!sender || !otp) {
      alert('Please fill in both fields');
      return;
    }

    setIsLoading(true);
    try {
      const endpoint = tabMode === 'simulate' ? '/api/otps/simulate' : '/api/otps/manual';
      await axios.post(`${API_BASE}${endpoint}`, {
        sender,
        otp,
      });
      // Clear form after successful submission
      setSender('Amazon');
      setOtp('527341');
    } catch (error) {
      alert('Error: ' + error.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="test-panel">
      <div className="panel-header">
        <h3>🧪 Test OTP Simulation</h3>
        <button className="close-btn" onClick={onClose}>✕</button>
      </div>

      <div className="panel-content">
        <p className="panel-description">
          Simulate receiving an OTP to test the dashboard in real-time
        </p>

        <div className="form-group">
          <label htmlFor="sender">Sender / Service</label>
          <input
            id="sender"
            type="text"
            value={sender}
            onChange={(e) => setSender(e.target.value)}
            placeholder="e.g., Amazon, Google, GitHub"
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="otp">OTP Code</label>
          <input
            id="otp"
            type="text"
            value={otp}
            onChange={(e) => setOtp(e.target.value)}
            placeholder="e.g., 527341"
            disabled={isLoading}
          />
        </div>

        <button
          className="submit-btn"
          onClick={handleSimulateOTP}
          disabled={isLoading}
        >
          {isLoading ? 'Sending...' : 'Send OTP'}
        </button>

        <div className="panel-info">
          <p>💡 <strong>Tip:</strong> Use the test panel to simulate OTPs arriving from different services. Watch the dashboard update in real-time!</p>
        </div>
      </div>
    </div>
  );
}

export default TestPanel;
