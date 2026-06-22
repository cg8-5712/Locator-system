import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { AuthUser } from "../types/auth";

interface AuthState {
  token: string | null;
  expiresAt: string | null;
  user: AuthUser | null;
  setSession: (input: {
    token: string;
    expiresAt: string;
    user: AuthUser;
  }) => void;
  clearSession: () => void;
}

export const authStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      expiresAt: null,
      user: null,
      setSession: ({ token, expiresAt, user }) =>
        set({
          token,
          expiresAt,
          user,
        }),
      clearSession: () =>
        set({
          token: null,
          expiresAt: null,
          user: null,
        }),
    }),
    {
      name: "locator-hub-auth",
    }
  )
);
