import type { UserListResult, UserSummary } from "../../types/user";
import { apiRequest } from "./client";

export async function fetchUsers(params?: { page?: number; pageSize?: number }) {
  const query = new URLSearchParams();
  query.set("page", String(params?.page ?? 1));
  query.set("page_size", String(params?.pageSize ?? 100));

  return apiRequest<UserListResult>(`/api/users?${query.toString()}`);
}

export async function createUser(input: {
  username: string;
  password: string;
  role: "admin" | "user";
}) {
  return apiRequest<UserSummary>("/api/users", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateUser(
  userID: number,
  input: {
    password?: string;
    role?: "admin" | "user";
  }
) {
  return apiRequest<UserSummary>(`/api/users/${userID}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteUser(userID: number) {
  return apiRequest<{ deleted: boolean; user_id: number }>(`/api/users/${userID}`, {
    method: "DELETE",
  });
}
