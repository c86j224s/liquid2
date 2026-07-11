CREATE INDEX documents_recent_order_idx
  ON documents(updated_at DESC, created_at DESC, id DESC);

CREATE INDEX documents_created_order_idx
  ON documents(created_at DESC, id DESC);

CREATE INDEX documents_rating_order_idx
  ON documents(COALESCE(rating, 0) DESC, updated_at DESC, created_at DESC, id DESC);
