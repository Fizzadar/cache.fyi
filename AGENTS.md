- this project, cache,fyi, is a go http service using sqlite for persistence
- you must use stdlib go where possible

## Project structure

- @internal/routes/ - route handlers
- @internal/routes/templates/ - html templates for relevant routes
- @internal/database/ - sqlite database
- @internal/database/migrations/ - database migrations
- @internal/workers/ - background workers

## Rules

- do NOT put inline CSS into templates, ignore styling unless explicitly requested
