#!/usr/bin/env bash

make_valid_env() {
  local path="$1"
  cat > "$path" <<'EOF'
POSTGRES_DB=mf_test
POSTGRES_USER=mf_user
POSTGRES_PASSWORD=synthetic-postgres-password-123456
POSTGRES_HOST=db
POSTGRES_PORT=15432
DB_BIND_ADDR=127.0.0.1
AUTH_PORT=19999
AUTH_BIND_ADDR=127.0.0.1
ALLOW_LAN_BINDING=no
GOTRUE_DB_DATABASE_URL=postgres://mf_user:synthetic-postgres-password-123456@db:5432/mf_test?sslmode=disable&search_path=auth
GOTRUE_SITE_URL=http://127.0.0.1:3000
GOTRUE_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3000
GOTRUE_URI_ALLOW_LIST=http://127.0.0.1:3000
GOTRUE_API_EXTERNAL_URL=http://127.0.0.1:19999
GOTRUE_JWT_SECRET=synthetic-jwt-secret-with-at-least-32-bytes
GOTRUE_OPERATOR_TOKEN=synthetic-operator-token-with-at-least-32-bytes
GOTRUE_DB_MAX_POOL_SIZE=10
TZ=UTC
EOF
}

replace_env_value() {
  local path="$1"
  local key="$2"
  local value="$3"
  local replacement="${path}.replacement"
  awk -v key="$key" -v value="$value" '
    index($0, key "=") == 1 { print key "=" value; next }
    { print }
  ' "$path" > "$replacement"
  mv "$replacement" "$path"
}

remove_env_key() {
  local path="$1"
  local key="$2"
  local replacement="${path}.replacement"
  awk -v key="$key" 'index($0, key "=") != 1 { print }' "$path" > "$replacement"
  mv "$replacement" "$path"
}
