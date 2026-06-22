import type { LoginResponse } from "../../types/auth";
import { apiRequest } from "./client";

export async function login(username: string, password: string) {
  return apiRequest<LoginResponse>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}
