ALTER TYPE exam_status ADD VALUE IF NOT EXISTS 'published';
ALTER TYPE exam_status ADD VALUE IF NOT EXISTS 'ended';
ALTER TABLE exams ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';
ALTER TABLE exams ADD COLUMN IF NOT EXISTS passing_score numeric(6,3) NOT NULL DEFAULT 60 CHECK(passing_score BETWEEN 0 AND 100);
ALTER TABLE exams ADD COLUMN IF NOT EXISTS total_marks numeric(10,3) NOT NULL DEFAULT 0 CHECK(total_marks >= 0);
ALTER TABLE exams ADD COLUMN IF NOT EXISTS allow_review boolean NOT NULL DEFAULT true;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS result_visibility varchar(30) NOT NULL DEFAULT 'not_published' CHECK(result_visibility IN ('not_published','published'));
ALTER TABLE exams ADD COLUMN IF NOT EXISTS published_at timestamptz;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS server_deadline timestamptz;
ALTER TABLE exam_attempts ADD COLUMN IF NOT EXISTS expires_at timestamptz;
ALTER TABLE student_answers ADD COLUMN IF NOT EXISTS text_answer text NOT NULL DEFAULT '';
ALTER TABLE student_answers ADD COLUMN IF NOT EXISTS marked_for_review boolean NOT NULL DEFAULT false;
CREATE TABLE IF NOT EXISTS attempt_questions (
  attempt_id uuid NOT NULL REFERENCES exam_attempts(id) ON DELETE CASCADE,
  question_id uuid NOT NULL REFERENCES questions(id),
  position integer NOT NULL CHECK(position > 0),
  points numeric(8,3) NOT NULL CHECK(points > 0),
  option_order jsonb NOT NULL DEFAULT '[]',
  PRIMARY KEY(attempt_id, question_id),
  UNIQUE(attempt_id, position)
);
CREATE INDEX IF NOT EXISTS idx_attempts_expiry ON exam_attempts(status, expires_at);
CREATE INDEX IF NOT EXISTS idx_attempt_questions_position ON attempt_questions(attempt_id, position);
CREATE INDEX IF NOT EXISTS idx_student_answers_attempt_saved ON student_answers(attempt_id, saved_at DESC);
UPDATE exams SET total_marks = COALESCE((SELECT SUM(eq.points) FROM exam_questions eq WHERE eq.exam_id=exams.id),0) WHERE total_marks=0;
