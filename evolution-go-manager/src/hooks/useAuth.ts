import useAuthStore from '@/store/authStore';

/**
 * useAuth Hook
 *
 * Convenient hook to access auth store
 * Provides authentication state and methods
 */
function useAuth() {
  const authStore = useAuthStore();

  return {
    // State
    isAuthenticated: authStore.isAuthenticated,
    apiUrl: authStore.apiUrl,
    apiKey: authStore.apiKey,

    // Methods
    login: authStore.login,
    logout: authStore.logout,
    setApiUrl: authStore.setApiUrl,
    setApiKey: authStore.setApiKey,
  };
}

export default useAuth;
