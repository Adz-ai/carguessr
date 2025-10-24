import { useEffect, useState } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';
import { LoginModal } from './LoginModal';
import { RegisterModal } from './RegisterModal';
import { PasswordResetModal } from './PasswordResetModal';
import { ProfileModal } from './ProfileModal';

type AuthModalType = 'login' | 'register' | 'passwordReset' | 'profile' | null;

export const AuthModals = () => {
  const [activeModal, setActiveModal] = useState<AuthModalType>(null);
  const { logout: storeLogout } = useAuthStore();

  useEffect(() => {
    const handleShowLogin = () => setActiveModal('login');
    const handleShowRegister = () => setActiveModal('register');
    const handleShowPasswordReset = () => setActiveModal('passwordReset');
    const handleShowProfile = () => setActiveModal('profile');

    const handleLogout = async () => {
      try {
        await apiClient.logout();
        storeLogout();
        showToast('Logged out successfully', 'info');
        window.location.reload();
      } catch (error) {
        console.error('Logout error:', error);
        storeLogout(); // Clear local state anyway
        window.location.reload();
      }
    };

    window.addEventListener('showLoginModal', handleShowLogin);
    window.addEventListener('showRegisterModal', handleShowRegister);
    window.addEventListener('showPasswordReset', handleShowPasswordReset);
    window.addEventListener('showProfile', handleShowProfile);
    window.addEventListener('logout', handleLogout);

    return () => {
      window.removeEventListener('showLoginModal', handleShowLogin);
      window.removeEventListener('showRegisterModal', handleShowRegister);
      window.removeEventListener('showPasswordReset', handleShowPasswordReset);
      window.removeEventListener('showProfile', handleShowProfile);
      window.removeEventListener('logout', handleLogout);
    };
  }, [storeLogout]);

  const closeModal = () => setActiveModal(null);

  return (
    <>
      {activeModal === 'login' && (
        <LoginModal
          onClose={closeModal}
          onSwitchToRegister={() => setActiveModal('register')}
          onSwitchToPasswordReset={() => setActiveModal('passwordReset')}
        />
      )}

      {activeModal === 'register' && (
        <RegisterModal
          onClose={closeModal}
          onSwitchToLogin={() => setActiveModal('login')}
        />
      )}

      {activeModal === 'passwordReset' && (
        <PasswordResetModal
          onClose={closeModal}
          onSwitchToLogin={() => setActiveModal('login')}
        />
      )}

      {activeModal === 'profile' && (
        <ProfileModal onClose={closeModal} />
      )}
    </>
  );
};
