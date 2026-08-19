CREATE TABLE IF NOT EXISTS resource_memberships (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_slug varchar(64) NOT NULL REFERENCES roles(slug),
  university_id uuid REFERENCES universities(id) ON DELETE CASCADE,
  faculty_id uuid REFERENCES faculties(id) ON DELETE CASCADE,
  department_id uuid REFERENCES departments(id) ON DELETE CASCADE,
  course_id uuid REFERENCES courses(id) ON DELETE CASCADE,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (university_id IS NOT NULL OR faculty_id IS NOT NULL OR department_id IS NOT NULL OR course_id IS NOT NULL),
  CHECK (role_slug IN ('university_admin','instructor','proctor'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_resource_membership_scope ON resource_memberships(user_id,role_slug,COALESCE(university_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(faculty_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(department_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(course_id,'00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX IF NOT EXISTS idx_resource_memberships_user ON resource_memberships(user_id,role_slug);
CREATE INDEX IF NOT EXISTS idx_resource_memberships_course ON resource_memberships(course_id,role_slug);
CREATE INDEX IF NOT EXISTS idx_resource_memberships_department ON resource_memberships(department_id,role_slug);

CREATE TABLE IF NOT EXISTS course_students (
  course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  student_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  assigned_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(course_id,student_id)
);
CREATE INDEX IF NOT EXISTS idx_course_students_student ON course_students(student_id,course_id);

CREATE TABLE IF NOT EXISTS course_instructors (
  course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  instructor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  assigned_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(course_id,instructor_id)
);
CREATE INDEX IF NOT EXISTS idx_course_instructors_instructor ON course_instructors(instructor_id,course_id);

CREATE TABLE IF NOT EXISTS exam_proctors (
  exam_id uuid NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  proctor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  assigned_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(exam_id,proctor_id)
);
CREATE INDEX IF NOT EXISTS idx_exam_proctors_proctor ON exam_proctors(proctor_id,exam_id);

INSERT INTO course_students(course_id,student_id,assigned_by)
VALUES ('40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
INSERT INTO course_instructors(course_id,instructor_id,assigned_by)
VALUES ('40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
INSERT INTO resource_memberships(user_id,role_slug,course_id,created_by)
VALUES ('00000000-0000-0000-0000-000000000002','instructor','40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
INSERT INTO resource_memberships(user_id,role_slug,university_id,created_by)
VALUES ('00000000-0000-0000-0000-000000000002','instructor','10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
INSERT INTO resource_memberships(user_id,role_slug,university_id,created_by)
VALUES ('00000000-0000-0000-0000-000000000003','proctor','10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
INSERT INTO exam_proctors(exam_id,proctor_id,assigned_by)
VALUES ('70000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
