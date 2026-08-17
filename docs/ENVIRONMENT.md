# Environment

Copy `.env.example` to `.env` for local development. `JWT_SECRET` must contain a random value of at least 32 characters in any non-development deployment. `DATABASE_URL` points to PostgreSQL. `CORS_ORIGINS` must list only trusted browser origins. `PROCTORING_URL` is the internal URL of the Python service, while `VITE_API_URL` is the public browser API base URL.

Never commit `.env`, production keys, real student data, passwords, refresh tokens or webcam evidence. Use a secret manager and distinct credentials per environment.
