#!/bin/bash
# Integration test for KOPDS
set -e

# Config
PORT=8082
URL="http://localhost:$PORT"
DB="kopds_test.db"
USER="testuser"
PASS="testpass"
LIB_DIR="test_library"
CACHE_DIR="test_cache"
BIN_NAME="kopds_bin"
TMP_BODY=$(mktemp)

# Assertion helpers
assert_status() {
    local expected="$1"
    local actual="$2"
    local label="$3"
    if [ "$actual" = "$expected" ]; then
        echo "PASS [$label]: HTTP $actual"
    else
        echo "FAIL [$label]: expected HTTP $expected, got HTTP $actual"
        echo "--- Server Logs ---"
        cat server.log
        exit 1
    fi
}

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$PID" ]; then
        kill $PID 2>/dev/null || true
    fi
    rm -f "$DB" "$DB-shm" "$DB-wal" "$BIN_NAME" "$TMP_BODY" server.log
    rm -rf $LIB_DIR $CACHE_DIR
}

trap cleanup EXIT

# Setup dummy library
echo "Setting up dummy library..."
mkdir -p "$LIB_DIR/Author Name/Book Title"
touch "$LIB_DIR/Author Name/Book Title/Book Title.epub"

# Build server
echo "Building KOPDS..."
GOCACHE=/tmp/kopds-gocache go build -o $BIN_NAME ../cmd/kopds

# Create user via CLI
echo "Creating test user..."
CLI_OUTPUT=$(KOPDS_DATABASE_PATH=$DB ./$BIN_NAME create-user "$USER" --password-stdin <<EOF
$PASS
EOF
)
if [[ $CLI_OUTPUT != *"User '$USER' created successfully."* ]]; then
    echo "CLI user creation FAILED: $CLI_OUTPUT"
    exit 1
fi
echo "PASS [CLI create-user]"

# Start server in background
echo "Starting KOPDS server..."
export KOPDS_PORT=$PORT
export KOPDS_LIBRARY_PATH=$LIB_DIR
export KOPDS_DATABASE_PATH=$DB
export KOPDS_IMAGE_CACHE_PATH=$CACHE_DIR
export KOPDS_BASE_URL=$URL
export KOPDS_LOG_LEVEL=debug
export KOPDS_RATE_LIMIT_ENABLED=false

./$BIN_NAME > server.log 2>&1 &
PID=$!
sleep 2

# Health check
echo "Testing Health Check..."
code=$(curl -s -o "$TMP_BODY" -w "%{http_code}" "$URL/health")
assert_status 200 "$code" "GET /health"

# Auth success: valid credentials
echo "Testing Auth Success (valid credentials)..."
code=$(curl -s -o "$TMP_BODY" -w "%{http_code}" -u "$USER:$PASS" "$URL/opds/v1.2/catalog")
assert_status 200 "$code" "GET /opds/v1.2/catalog valid creds"
if [[ $(cat "$TMP_BODY") != *"KOPDS Root Catalog"* ]]; then
    echo "FAIL [catalog body]: expected 'KOPDS Root Catalog' in response"
    echo "--- Body ---"
    cat "$TMP_BODY"
    echo "--- Server Logs ---"
    cat server.log
    exit 1
fi
echo "PASS [catalog body]: found 'KOPDS Root Catalog'"

# Auth failure: wrong password
echo "Testing Auth Failure (wrong password)..."
code=$(curl -s -o "$TMP_BODY" -w "%{http_code}" -u "$USER:wrongpass" "$URL/opds/v1.2/catalog")
assert_status 401 "$code" "GET /opds/v1.2/catalog wrong password"

# Auth failure: missing credentials
echo "Testing Auth Failure (missing credentials)..."
code=$(curl -s -o "$TMP_BODY" -w "%{http_code}" "$URL/opds/v1.2/catalog")
assert_status 401 "$code" "GET /opds/v1.2/catalog no credentials"

# 404 on unknown path (authenticated)
echo "Testing 404 on unknown path..."
code=$(curl -s -o "$TMP_BODY" -w "%{http_code}" -u "$USER:$PASS" "$URL/notfound")
assert_status 404 "$code" "GET /notfound"

echo ""
echo "Integration test PASSED"
echo "--- Server Logs ---"
cat server.log
