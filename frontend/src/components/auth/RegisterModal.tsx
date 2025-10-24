import { useState, type FormEvent } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';

interface RegisterModalProps {
  onClose: () => void;
  onSwitchToLogin: () => void;
}

export const RegisterModal = ({ onClose, onSwitchToLogin }: RegisterModalProps) => {
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [securityQuestion, setSecurityQuestion] = useState('');
  const [securityAnswer, setSecurityAnswer] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const { login } = useAuthStore();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const data = await apiClient.register({
        username,
        displayName,
        password,
        securityQuestion,
        securityAnswer
      });
      login(data.user, data.sessionToken);
      showToast('Account created successfully!', 'success');
      onClose();
      window.location.reload(); // Refresh to update UI
    } catch (error: any) {
      showToast(error.message || 'Registration failed', 'error');
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
        <h2>Create Account</h2>
        <form id="registerForm" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="registerUsername">Username</label>
            <input
              type="text"
              id="registerUsername"
              name="username"
              required
              minLength={3}
              maxLength={20}
              pattern="[a-zA-Z0-9_-]+"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isLoading}
            />
            <small>Letters, numbers, underscores, and hyphens only</small>
          </div>
          <div className="form-group">
            <label htmlFor="registerDisplayName">Display Name</label>
            <input
              type="text"
              id="registerDisplayName"
              name="displayName"
              required
              minLength={1}
              maxLength={30}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              disabled={isLoading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="registerPassword">Password</label>
            <input
              type="password"
              id="registerPassword"
              name="password"
              required
              minLength={6}
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isLoading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="registerSecurityQuestion">Security Question</label>
            <input
              type="text"
              id="registerSecurityQuestion"
              name="securityQuestion"
              required
              minLength={5}
              maxLength={200}
              placeholder="e.g., What was your first pet's name?"
              value={securityQuestion}
              onChange={(e) => setSecurityQuestion(e.target.value)}
              disabled={isLoading}
            />
            <small>This will be used to recover your password if forgotten</small>
          </div>
          <div className="form-group">
            <label htmlFor="registerSecurityAnswer">Security Answer</label>
            <input
              type="text"
              id="registerSecurityAnswer"
              name="securityAnswer"
              required
              minLength={2}
              maxLength={100}
              placeholder="Your answer to the security question"
              value={securityAnswer}
              onChange={(e) => setSecurityAnswer(e.target.value)}
              disabled={isLoading}
            />
            <small>Case insensitive - remember this answer exactly</small>
          </div>
          <button type="submit" className="auth-submit-btn" disabled={isLoading}>
            {isLoading ? 'Creating Account...' : 'Create Account'}
          </button>
        </form>
        <p className="auth-switch">
          Already have an account? <a href="#" onClick={(e) => { e.preventDefault(); onSwitchToLogin(); }}>Login</a>
        </p>
      </div>
    </div>
  );
};
