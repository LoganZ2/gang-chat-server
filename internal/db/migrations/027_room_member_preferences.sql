ALTER TABLE room_memberships ADD COLUMN is_pinned INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_room_memberships_user_pinned
ON room_memberships(user_id, is_pinned);
