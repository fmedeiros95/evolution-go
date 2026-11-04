/**
 * Authentication types
 */

export interface AuthState {
  apiUrl: string;
  apiKey: string;
  isAuthenticated: boolean;
}

export interface AuthStore extends AuthState {
  login: (apiUrl: string, apiKey: string) => Promise<void>;
  logout: () => void;
  setApiUrl: (apiUrl: string) => void;
  setApiKey: (apiKey: string) => void;
}

export interface LoginCredentials {
  apiUrl: string;
  apiKey: string;
}

export interface LoginResponse {
  success: boolean;
  message?: string;
}
