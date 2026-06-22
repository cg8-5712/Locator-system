export interface AuthUser {
  id: number;
  username: string;
  role: string;
}

export interface LoginResponse {
  token: string;
  expires_at: string;
  user: AuthUser;
}
