import { useState, type FormEvent } from 'react';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';

interface PasswordResetModalProps {
  onClose: () => void;
  onSwitchToLogin: () => void;
}

export const PasswordResetModal = ({ onClose, onSwitchToLogin }: PasswordResetModalProps) => {
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [securityQuestion, setSecurityQuestion] = useState('');
  const [securityAnswer, setSecurityAnswer] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [showSecuritySection, setShowSecuritySection] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleVerifyUser = async () => {
    if (!username || !displayName) {
      showToast('Please enter both username and display name', 'error');
      return;
    }

    setIsLoading(true);

    try {
      const data = await apiClient.getSecurityQuestion(username, displayName);
      setSecurityQuestion(data.securityQuestion);
      setShowSecuritySection(true);
      showToast('Security question retrieved', 'info');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'User not found with those credentials';
      showToast(message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      await apiClient.resetPassword({
        username,
        displayName,
        securityAnswer,
        newPassword
      });
      showToast('Password reset successfully! You can now login.', 'success');
      onSwitchToLogin();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Password reset failed';
      showToast(message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content auth-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>Reset Password</h2>
        <p>To reset your password, please provide your username, display name, and answer to your security question.</p>
        <form id="passwordResetForm" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="resetUsername">Username</label>
            <input
              type="text"
              id="resetUsername"
              name="username"
              required
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isLoading || showSecuritySection}
            />
          </div>
          <div className="form-group">
            <label htmlFor="resetDisplayName">Display Name</label>
            <input
              type="text"
              id="resetDisplayName"
              name="displayName"
              required
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              disabled={isLoading || showSecuritySection}
            />
          </div>

          {showSecuritySection && (
            <div id="securityQuestionSection">
              <div className="form-group">
                <label id="securityQuestionLabel">Security Question</label>
                <p id="securityQuestionText" className="security-question-text">{securityQuestion}</p>
              </div>
              <div className="form-group">
                <label htmlFor="resetSecurityAnswer">Your Answer</label>
                <input
                  type="text"
                  id="resetSecurityAnswer"
                  name="securityAnswer"
                  required
                  value={securityAnswer}
                  onChange={(e) => setSecurityAnswer(e.target.value)}
                  disabled={isLoading}
                />
                <small>Case insensitive</small>
              </div>
              <div className="form-group">
                <label htmlFor="resetNewPassword">New Password</label>
                <input
                  type="password"
                  id="resetNewPassword"
                  name="newPassword"
                  required
                  minLength={6}
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={isLoading}
                />
              </div>
            </div>
          )}

          {!showSecuritySection ? (
            <button
              type="button"
              id="verifyUserBtn"
              className="auth-submit-btn secondary"
              onClick={handleVerifyUser}
              disabled={isLoading}
            >
              {isLoading ? 'Verifying...' : 'Verify Identity'}
            </button>
          ) : (
            <button type="submit" id="resetPasswordBtn" className="auth-submit-btn" disabled={isLoading}>
              {isLoading ? 'Resetting...' : 'Reset Password'}
            </button>
          )}
        </form>
        <p className="auth-switch">
          Remember your password? <a href="#" onClick={(e) => { e.preventDefault(); onSwitchToLogin(); }}>Login</a>
        </p>
      </div>
    </div>
  );
};
