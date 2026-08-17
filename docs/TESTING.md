# Testing

The Go service includes unit tests for password hashing, JWT claim preservation, minimum password length and objective grading, including multiple-select and negative marking.

## Automated checks

The repository includes two repeatable monitoring checks and one full exam acceptance path:

| Script | Coverage |
|---|---|
| `scripts/proctoring_signal_matrix.py` | Python analyzer classification for face missing, multiple faces, camera interruption, fullscreen exit, tab visibility, network offline and degraded network, including expected severity |
| `scripts/proctoring_e2e.py` | Proctor login, authenticated WebSocket handshake, published/scheduled exam, student attempt, Python signal, persisted event, WebSocket broadcast, active dashboard, human review, answer save and Exam Engine submit/result |
| `scripts/e2e_acceptance.sh` | Existing Exam Engine lifecycle from draft exam through publish, schedule, student start, autosave, submit, grading, result publication and student result visibility |

## Local checks

```bash
cd backend-go
go test ./...
go vet ./...

auto=$(pwd)
cd ../frontend
npm run typecheck
npm run build

cd ..
python3 -m compileall -q ai-proctoring-python/app
sudo docker compose config
sudo docker compose up --build -d
python3 scripts/proctoring_signal_matrix.py
python3 scripts/proctoring_e2e.py
./scripts/e2e_acceptance.sh
```

The Phase 3 acceptance path verifies that a proctor can connect to the WebSocket feed, a multiple-face signal is classified as `high`, the risk is persisted, the active-session dashboard exposes the open event, a human reviewer can confirm it, and the same attempt can still save, submit, grade and publish a result. The matrix verifies the remaining signal classes directly against the Python service.

## Latest verified local result

The latest Docker run completed with PostgreSQL healthy, Go tests and vet passing, frontend typecheck and production build passing, Python compilation passing, `PROCTORING_SIGNAL_MATRIX_PASS cases=7`, `PROCTORING_E2E_PASS`, and `E2E_PASS`. The Vite build still emits a non-blocking configuration warning about native config loading; this does not fail typecheck or build.

## Remaining QA scope

Production QA must additionally cover refresh-token rotation, password reset and email verification, the full browser webcam/device matrix, self-hosted and pinned MediaPipe model assets, offline conflict resolution under concurrent tabs, access scope for multi-institution instructor review, accessibility, mobile layouts, rate limits, backup restoration, concurrent sessions, WebSocket load and reconnect storms. The Python service deliberately records privacy-preserving signals and neither it nor the Go risk engine claims that a signal alone proves misconduct.
