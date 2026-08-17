# Testing

The Go service includes unit tests for password hashing, JWT claim preservation, minimum password length and objective grading, including multiple-select and negative marking.

The repository now includes a repeatable database-backed acceptance script at `scripts/e2e_acceptance.sh`. It creates a draft exam, attaches a question, publishes and schedules it, logs in as a student, starts an attempt, saves an answer, submits, generates a result, publishes that result as an instructor, and verifies that the student can read the published result.

## Local checks

```bash
cd backend-go
go test ./...
go vet ./...

cd ../frontend
npm run typecheck
npm run build

cd ..
python3 -m compileall -q ai-proctoring-python/app
sudo docker compose config
sudo docker compose up --build -d
./scripts/e2e_acceptance.sh
```

The acceptance path verifies server-authoritative deadlines, attempt limits, transactional answer persistence, deterministic question/option randomization, objective grading, negative marking, result metrics and result publication. The backend returns explicit errors for invalid or expired attempts instead of accepting client-controlled time.

## Remaining QA scope

Production QA must additionally cover refresh-token rotation, password reset and email verification, the full browser webcam/device matrix, offline conflict resolution under concurrent tabs, human review workflows, CSV/JSON/PDF exports, accessibility, mobile layouts, rate limits, backup restoration, concurrent sessions and load. The Python service deliberately records privacy-preserving signals and does not claim that a signal alone proves misconduct.
