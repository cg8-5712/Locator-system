import type { Pagination } from "./api";

export interface UserSummary {
  id: number;
  username: string;
  role: "admin" | "user";
}

export interface UserListResult {
  users: UserSummary[];
  pagination: Pagination;
}
