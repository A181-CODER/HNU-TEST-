#!/usr/bin/env python3
import os
import requests
import websocket

API=os.environ.get('API','http://localhost:8080/api/v1')
PASSWORD='ChangeMe-Development-Only!'

def login(email):
    r=requests.post(f'{API}/auth/login',json={'email':email,'password':PASSWORD},timeout=10);r.raise_for_status();return r.json()['accessToken']
def req(method,path,token,expected=None,**kwargs):
    r=requests.request(method,f'{API}{path}',headers={'Authorization':f'Bearer {token}'},timeout=15,**kwargs)
    if expected is not None:
        assert r.status_code==expected,(method,path,r.status_code,r.text)
    elif r.status_code>=400:
        r.raise_for_status()
    return r

def main():
    admin=login('admin@hnu-test.local'); ia=login('instructor@hnu-test.local'); ib=login('instructor-b@hnu-test.local'); pa=login('proctor@hnu-test.local'); pb=login('proctor-b@hnu-test.local'); sa=login('student@hnu-test.local'); sb=login('student-b@hnu-test.local')
    tree_a=req('GET','/organization/tree',ia).json();tree_b=req('GET','/organization/tree',ib).json();tree_admin=req('GET','/organization/tree',admin).json()
    assert {u['code'] for u in tree_a['universities']}=={'HNU-TEST'},tree_a
    assert {u['code'] for u in tree_b['universities']}=={'HNU-B'},tree_b
    assert {u['code'] for u in tree_admin['universities']}=={'HNU-TEST','HNU-B'},tree_admin

    exams_a=req('GET','/exams',ia).json();exams_b=req('GET','/exams',ib).json();exams_admin=req('GET','/exams',admin).json()
    assert all(e['courseCode']!='B-201' for e in exams_a),exams_a
    assert any(e['courseCode']=='B-201' for e in exams_b),exams_b
    assert any(e['courseCode']=='B-201' for e in exams_admin),exams_admin
    req('GET','/exams/70000000-0000-0000-0000-000000000002',ia,403)
    req('GET','/exams/70000000-0000-0000-0000-000000000002',ib,200)
    req('POST','/exams',ia,403,json={'courseId':'40000000-0000-0000-0000-000000000002','courseCode':'B-201','title':'Cross scope','durationMinutes':30})

    student_a_exams=req('GET','/student/exams',sa).json();student_b_exams=req('GET','/student/exams',sb).json()
    assert all(e['courseCode']!='B-201' for e in student_a_exams),student_a_exams
    assert any(e['courseCode']=='B-201' for e in student_b_exams),student_b_exams
    req('POST','/exams/70000000-0000-0000-0000-000000000002/start',sa,403)
    attempt_b=req('POST','/exams/70000000-0000-0000-0000-000000000002/start',sb).json()['id']
    req('GET',f'/attempts/{attempt_b}',sa,403)
    req('GET',f'/attempts/{attempt_b}',pa,403)
    req('GET',f'/proctoring/attempts/{attempt_b}/events',pa,403)
    try:
        websocket.create_connection(f'ws://localhost:8080/api/v1/proctoring/ws?token={pa}&attemptId={attempt_b}', origin='http://localhost:5173', timeout=5)
        raise AssertionError('proctor A unexpectedly opened University B WebSocket')
    except Exception:
        pass

    req('GET','/proctoring/active-attempts',pa,200)
    active_a=req('GET','/proctoring/active-attempts',pa).json();active_b=req('GET','/proctoring/active-attempts',pb).json()
    assert all(x['examTitle']!='University B Isolation Exam' for x in active_a),active_a
    assert any(x['examTitle']=='University B Isolation Exam' for x in active_b),active_b
    print({'status':'PHASE4_ISOLATION_PASS','universityAVisible':sorted(u['code'] for u in tree_a['universities']),'universityBVisible':sorted(u['code'] for u in tree_b['universities']),'studentBAttempt':attempt_b,'proctorAActive':len(active_a),'proctorBActive':len(active_b)})

if __name__=='__main__': main()
