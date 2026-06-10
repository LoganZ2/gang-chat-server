DROP TABLE IF EXISTS room_invites_rebuild;

CREATE TABLE room_invites_rebuild (
    id TEXT PRIMARY KEY NOT NULL,
    room_id TEXT NOT NULL,
    inviter_user_id TEXT NOT NULL,
    target_user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    room_rid TEXT NOT NULL DEFAULT '',
    room_name TEXT NOT NULL DEFAULT '',
    room_avatar_url TEXT,
    room_default_avatar_key TEXT NOT NULL DEFAULT 'room-1',
    room_visibility TEXT NOT NULL DEFAULT 'private',
    room_join_policy TEXT NOT NULL DEFAULT 'closed',
    UNIQUE(room_id, target_user_id, inviter_user_id),
    FOREIGN KEY(inviter_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(target_user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO room_invites_rebuild (
    id,
    room_id,
    inviter_user_id,
    target_user_id,
    status,
    created_at,
    updated_at,
    room_rid,
    room_name,
    room_avatar_url,
    room_default_avatar_key,
    room_visibility,
    room_join_policy
)
SELECT
    id,
    room_id,
    inviter_user_id,
    target_user_id,
    status,
    created_at,
    updated_at,
    room_rid,
    room_name,
    room_avatar_url,
    room_default_avatar_key,
    room_visibility,
    room_join_policy
FROM room_invites;

DROP TABLE room_invites;
ALTER TABLE room_invites_rebuild RENAME TO room_invites;

CREATE INDEX IF NOT EXISTS idx_room_invites_target
ON room_invites(target_user_id, status, created_at);

CREATE INDEX IF NOT EXISTS idx_room_invites_room_target
ON room_invites(room_id, target_user_id, status);
