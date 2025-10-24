import { useState, type FormEvent } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';

interface LoginModalProps {
  onClose: () => void;
  onSwitchToRegister: () => void;
  onSwitchToPasswordReset: () => void;
}

export const LoginModal = ({ onClose, onSwitchToRegister, onSwitchToPasswordReset }: LoginModalProps) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const { login } = useAuthStore();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const data = await apiClient.login(username, password);
      login(data.user, data.sessionToken);
      showToast('Login successful!', 'success');
      onClose();
      window.location.reload(); // Refresh to update UI
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Login failed';
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
        <h2>Welcome Back!</h2>
        <form id="loginForm" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="loginUsername">Username</label>
            <input
              type="text"
              id="loginUsername"
              name="username"
              required
              minLength={3}
              maxLength={20}
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={isLoading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="loginPassword">Password</label>
            <input
              type="password"
              id="loginPassword"
              name="password"
              required
              minLength={6}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isLoading}
            />
          </div>
          <button type="submit" className="auth-submit-btn" disabled={isLoading}>
            {isLoading ? 'Logging in...' : 'Login'}
          </button>
        </form>
        <p className="auth-switch">
          Don't have an account? <a href="#" onClick={(e) => { e.preventDefault(); onSwitchToRegister(); }}>Sign up</a><br />
          <a href="#" onClick={(e) => { e.preventDefault(); onSwitchToPasswordReset(); }}>Forgot Password?</a>
        </p>
      </div>
    </div>
  );
};
