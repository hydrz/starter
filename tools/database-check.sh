#!/usr/bin/env sh
set -eu

export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-enterprise-platform-check-$$}"
export POSTGRES_PORT="${POSTGRES_PORT:-55432}"
export DATABASE_URL="${DATABASE_URL:-postgres://enterprise:enterprise@127.0.0.1:${POSTGRES_PORT}/enterprise?sslmode=disable}"
export GOFLAGS="${GOFLAGS:--tags=no_clickhouse,no_libsql,no_mssql,no_mysql,no_sqlite3,no_vertica,no_ydb}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to run the database integration check" >&2
  exit 127
fi

cleanup() {
  docker compose down --volumes
}
trap cleanup EXIT INT TERM

docker compose up --detach --wait postgres
go tool goose -dir db/migrations postgres "$DATABASE_URL" up
go tool goose -dir db/migrations postgres "$DATABASE_URL" down
go tool goose -dir db/migrations postgres "$DATABASE_URL" up
go tool goose -dir db/migrations postgres "$DATABASE_URL" status
