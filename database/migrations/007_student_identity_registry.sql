-- Controlled student-code registry. Real codes are imported at deployment time, never committed.
CREATE TABLE IF NOT EXISTS student_identity_registry (
  student_number varchar(80) PRIMARY KEY,
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  status varchar(24) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','linked','disabled')),
  source varchar(80) NOT NULL DEFAULT 'controlled-import',
  verified_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_student_identity_user ON student_identity_registry(user_id);
CREATE INDEX IF NOT EXISTS idx_student_identity_status ON student_identity_registry(status);

-- Development fixture only; production codes are loaded through scripts/import_student_codes.sh.
INSERT INTO student_identity_registry(student_number,user_id,status,source,verified_at)
VALUES ('DEMO-2026-001','00000000-0000-0000-0000-000000000004','linked','development-seed',now())
ON CONFLICT (student_number) DO NOTHING;

-- Optional Phase 4 fixture; this is inserted only when the isolated test account exists.
INSERT INTO student_identity_registry(student_number,user_id,status,source,verified_at)
SELECT 'B-2026-001', id, 'linked', 'development-fixture', now()
FROM users WHERE email='student-b@hnu-test.local'
ON CONFLICT (student_number) DO NOTHING;
