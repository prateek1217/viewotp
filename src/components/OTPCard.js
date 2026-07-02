import React, { useState, useEffect } from 'react';
import './OTPCard.css';

function OTPCard({ otp, onView, onCopy }) {
  const [timeLeft, setTimeLeft] = useState(null);
  const [isExpired, setIsExpired] = useState(false);

  useEffect(() => {
    onView();

    const interval = setInterval(() => {
      const now = new Date();
      const expiry = new Date(otp.expires_at);
      const diff = expiry - now;

      if (diff <= 0) {
        setIsExpired(true);
        clearInterval(interval);
      } else {
        const minutes = Math.floor(diff / 60000);
        const seconds = Math.floor((diff % 60000) / 1000);
        setTimeLeft(`${minutes}m ${seconds}s`);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [otp, onView]);

  const receivedTime = new Date(otp.received_at).toLocaleTimeString();

  return (
    <div className={`otp-card ${isExpired ? 'expired' : ''}`}>
      <div className="otp-card-header">
        <h3>{otp.sender}</h3>
        <span className={`expiry-badge ${isExpired ? 'expired' : 'active'}`}>
          {isExpired ? 'Expired' : timeLeft || 'Loading...'}
        </span>
      </div>

      <div className="otp-code">
        <div className="otp-display">{otp.otp}</div>
        <button
          className="copy-btn"
          onClick={onCopy}
          disabled={isExpired}
          title="Copy OTP"
        >
          📋 Copy
        </button>
      </div>

      <div className="otp-card-footer">
        <div className="otp-meta">
          <span className="meta-label">Received:</span>
          <span className="meta-value">{receivedTime}</span>
        </div>
        {otp.viewed_by && otp.viewed_by.length > 0 && (
          <div className="otp-meta">
            <span className="meta-label">Viewed by:</span>
            <span className="meta-value">{otp.viewed_by.join(', ')}</span>
          </div>
        )}
      </div>
    </div>
  );
}

export default OTPCard;
