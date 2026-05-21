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

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$PID" ]; then
        kill $PID 2>/dev/null || true
    fi
    rm -f "$DB" "$DB-shm" "$DB-wal" "$BIN_NAME"
    rm -rf $LIB_DIR $CACHE_DIR
}

trap cleanup EXIT

# Setup dummy library
echo "Setting up dummy library..."
mkdir -p "$LIB_DIR/Author Name/Book Title"
touch "$LIB_DIR/Author Name/Book Title/Book Title.epub"

# Build server
echo "Building KOPDS..."
go build -o $BIN_NAME ../cmd/kopds

# Create user via CLI
echo "Creating test user..."
CLI_OUTPUT=$(KOPDS_DATABASE_PATH=$DB ./$BIN_NAME create-user "$USER" --password-stdin <<EOF
$PASS
EOF
)
if [[ $CLI_OUTPUT != *"User '$USER' created/updated successfully."* ]]; then
    echo "CLI user creation FAILED: $CLI_OUTPUT"
    exit 1
fi

# Start server in background
echo "Starting KOPDS server..."
export KOPDS_PORT=$PORT
export KOPDS_LIBRARY_PATH=$LIB_DIR
export KOPDS_DATABASE_PATH=$DB
export KOPDS_IMAGE_CACHE_PATH=$CACHE_DIR
export KOPDS_BASE_URL=$URL
export KOPDS_LOG_LEVEL=debug

./$BIN_NAME &
PID=$!
sleep 2

# Verify server is up
echo "Verifying Health Check..."
curl -s -f $URL/health > /dev/null

# Test OPDS Catalog with Basic Auth
echo "Testing OPDS Catalog Access..."
RESP=$(curl -s -u "$USER:$PASS" "$URL/opds/v1.2/catalog")

if [[ $RESP == *"KOPDS Root Catalog"* ]]; then
    echo "Integration test PASSED"
else
    echo "Integration test FAILED: $RESP"
    exit 1
fi
