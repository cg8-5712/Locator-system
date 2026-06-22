import { authStore } from "../stores/auth-store";

export function useAuth() {
  return authStore();
}
