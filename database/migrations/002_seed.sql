-- Development-only seed data. Replace credentials and remove this migration before production.
INSERT INTO users(id,email,display_name,password_hash) VALUES
('00000000-0000-0000-0000-000000000001','admin@hnu-test.local','Demo System Administrator','$2a$10$XUuP3Lls6OuVTG8rDbYw6.4mqxNe.FIFvbHNNhEB3Be3fqPYgmxty'),
('00000000-0000-0000-0000-000000000002','instructor@hnu-test.local','Demo Instructor','$2a$10$XUuP3Lls6OuVTG8rDbYw6.4mqxNe.FIFvbHNNhEB3Be3fqPYgmxty'),
('00000000-0000-0000-0000-000000000003','proctor@hnu-test.local','Demo Proctor','$2a$10$XUuP3Lls6OuVTG8rDbYw6.4mqxNe.FIFvbHNNhEB3Be3fqPYgmxty'),
('00000000-0000-0000-0000-000000000004','student@hnu-test.local','Demo Student','$2a$10$XUuP3Lls6OuVTG8rDbYw6.4mqxNe.FIFvbHNNhEB3Be3fqPYgmxty')
ON CONFLICT (id) DO NOTHING;
INSERT INTO user_roles(user_id,role_id)
SELECT '00000000-0000-0000-0000-000000000001',id FROM roles WHERE slug='super_admin' ON CONFLICT DO NOTHING;
INSERT INTO user_roles(user_id,role_id)
SELECT '00000000-0000-0000-0000-000000000002',id FROM roles WHERE slug='instructor' ON CONFLICT DO NOTHING;
INSERT INTO user_roles(user_id,role_id)
SELECT '00000000-0000-0000-0000-000000000003',id FROM roles WHERE slug='proctor' ON CONFLICT DO NOTHING;
INSERT INTO user_roles(user_id,role_id)
SELECT '00000000-0000-0000-0000-000000000004',id FROM roles WHERE slug='student' ON CONFLICT DO NOTHING;
INSERT INTO universities(id,name,code) VALUES ('10000000-0000-0000-0000-000000000001','HNU Test University','HNU-TEST') ON CONFLICT DO NOTHING;
INSERT INTO faculties(id,university_id,name,code) VALUES ('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','Faculty of Computing','FC') ON CONFLICT DO NOTHING;
INSERT INTO departments(id,faculty_id,name,code) VALUES ('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','Computer Science','CS') ON CONFLICT DO NOTHING;
INSERT INTO courses(id,department_id,code,title,credits) VALUES ('40000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','SWE-201','Software Engineering Fundamentals',3) ON CONFLICT DO NOTHING;
INSERT INTO students(user_id,student_number,department_id,year_level) VALUES ('00000000-0000-0000-0000-000000000004','DEMO-2026-001','30000000-0000-0000-0000-000000000001',2) ON CONFLICT DO NOTHING;
INSERT INTO instructors(user_id,employee_number,department_id) VALUES ('00000000-0000-0000-0000-000000000002','DEMO-EMP-001','30000000-0000-0000-0000-000000000001') ON CONFLICT DO NOTHING;
INSERT INTO question_bank(id,owner_id,course_id,name,description) VALUES ('50000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','40000000-0000-0000-0000-000000000001','SWE-201 Core Bank','Development seed question bank') ON CONFLICT DO NOTHING;
INSERT INTO questions(id,bank_id,author_id,type,prompt,points,difficulty) VALUES ('60000000-0000-0000-0000-000000000001','50000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','multiple_choice','Which practice most directly improves maintainability?',2,'medium') ON CONFLICT DO NOTHING;
INSERT INTO question_options(question_id,option_key,option_text,is_correct) VALUES ('60000000-0000-0000-0000-000000000001','A','Shared naming conventions',true),('60000000-0000-0000-0000-000000000001','B','Undocumented duplication',false),('60000000-0000-0000-0000-000000000001','C','Disabled tests',false) ON CONFLICT DO NOTHING;
INSERT INTO exams(id,created_by,course_id,course_code,title,instructions,status,duration_minutes,attempt_limit,randomize_questions,randomize_options) VALUES ('70000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','40000000-0000-0000-0000-000000000001','SWE-201','Development Seed Examination','This is a labelled development exam.','scheduled',90,1,true,true) ON CONFLICT DO NOTHING;
INSERT INTO exam_questions(exam_id,question_id,position,points) VALUES ('70000000-0000-0000-0000-000000000001','60000000-0000-0000-0000-000000000001',1,2) ON CONFLICT DO NOTHING;
