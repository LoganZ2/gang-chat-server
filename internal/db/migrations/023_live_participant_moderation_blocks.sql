ALTER TABLE live_participants ADD COLUMN mic_blocked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE live_participants ADD COLUMN headphones_blocked INTEGER NOT NULL DEFAULT 0;

UPDATE live_participants
SET mic_blocked = 1,
    headphones_blocked = 1
WHERE voice_blocked = 1;
