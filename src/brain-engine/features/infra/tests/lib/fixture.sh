#!/usr/bin/env bash

make_valid_env() {
  local path="$1"
  cat > "$path" <<'EOF'
POSTGRES_ADMIN_USER=mf_cluster_admin
POSTGRES_ADMIN_PASSWORD=synthetic-cluster-password-123456
POSTGRES_HOST=db
POSTGRES_CLIENT_HOST=127.0.0.1
POSTGRES_PORT=15432
DB_BIND_ADDR=127.0.0.1
DEVELOP_DB_NAME=mindforge_develop
DEVELOP_DB_USER=mf_develop
DEVELOP_DB_PASSWORD=synthetic-develop-password-123456
MAIN_DB_NAME=mindforge
MAIN_DB_USER=mf_main
MAIN_DB_PASSWORD=synthetic-main-password-123456
DEVELOP_AUTH_DB_URL=postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop?sslmode=disable&search_path=auth
MAIN_AUTH_DB_URL=postgres://mf_main:synthetic-main-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth
DEVELOP_AUTH_PORT=19999
MAIN_AUTH_PORT=19998
DEVELOP_AUTH_BIND_ADDR=127.0.0.1
MAIN_AUTH_BIND_ADDR=127.0.0.1
ALLOW_LAN_BINDING=no
DEVELOP_AUTH_SITE_URL=http://127.0.0.1:3000
MAIN_AUTH_SITE_URL=http://127.0.0.1:3001
DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3000
MAIN_AUTH_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3001
DEVELOP_AUTH_URI_ALLOW_LIST=http://127.0.0.1:3000
MAIN_AUTH_URI_ALLOW_LIST=http://127.0.0.1:3001
DEVELOP_AUTH_API_EXTERNAL_URL=http://127.0.0.1:19999
MAIN_AUTH_API_EXTERNAL_URL=http://127.0.0.1:19998
DEVELOP_AUTH_JWT_SECRET=synthetic-develop-jwt-secret-with-32-bytes
MAIN_AUTH_JWT_SECRET=synthetic-main-jwt-secret-with-32-bytes
DEVELOP_AUTH_OPERATOR_TOKEN=synthetic-develop-operator-token-32-bytes
MAIN_AUTH_OPERATOR_TOKEN=synthetic-main-operator-token-32-bytes
GOTRUE_DB_MAX_POOL_SIZE=10
GOTRUE_MAILER_AUTOCONFIRM=false
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
