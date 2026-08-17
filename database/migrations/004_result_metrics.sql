ALTER TABLE results ADD COLUMN IF NOT EXISTS correct_count integer NOT NULL DEFAULT 0 CHECK(correct_count >= 0);
ALTER TABLE results ADD COLUMN IF NOT EXISTS incorrect_count integer NOT NULL DEFAULT 0 CHECK(incorrect_count >= 0);
ALTER TABLE results ADD COLUMN IF NOT EXISTS unanswered_count integer NOT NULL DEFAULT 0 CHECK(unanswered_count >= 0);
ALTER TABLE results ADD COLUMN IF NOT EXISTS duration_seconds integer NOT NULL DEFAULT 0 CHECK(duration_seconds >= 0);
