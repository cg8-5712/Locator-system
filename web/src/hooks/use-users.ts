import { useQuery } from "@tanstack/react-query";
import { fetchUsers } from "../services/http/users";

export function useUsers() {
  return useQuery({
    queryKey: ["users"],
    queryFn: () => fetchUsers({ page: 1, pageSize: 100 }),
    staleTime: 15_000,
  });
}
