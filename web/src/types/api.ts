export interface ApiEnvelope<T> {
  success: boolean;
  data: T;
  error?: string;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
  has_next: boolean;
}
