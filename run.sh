#!/bin/bash
set -e

# SongMartyn - Build and Run Script
# Usage: ./run.sh [options]
#
# Options (passed to backend server):
#   -port 8443       HTTPS port (default: 8443)
#   -http-port 8080  HTTP redirect port (default: 8080)
#   -pin 1234        Admin PIN (random if not set)
#   -dev             Development mode (CORS enabled)
#
# Examples:
#   ./run.sh -pin 1234                    # HTTPS on 8443, HTTP redirect on 8080
#   ./run.sh -port 443 -http-port 80      # Standard ports (requires sudo)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_DIR="$SCRIPT_DIR/backend"

echo "=== SongMartyn Build & Run ==="
echo ""

# Build frontend
echo "[1/3] Building frontend..."
cd "$FRONTEND_DIR"
if [ ! -d "node_modules" ]; then
    echo "  Installing npm dependencies..."
    npm install
fi
npm run build
echo "  Frontend built to: $FRONTEND_DIR/dist"
echo ""

# Build backend
echo "[2/3] Building backend..."
cd "$BACKEND_DIR"
go build -o songmartyn ./cmd/songmartyn
echo "  Backend built: $BACKEND_DIR/songmartyn"
echo ""

# Check for TLS certificates
if [ ! -f "$BACKEND_DIR/certs/cert.pem" ] || [ ! -f "$BACKEND_DIR/certs/key.pem" ]; then
    echo "[!] TLS certificates not found. Generating self-signed certificates..."
    mkdir -p "$BACKEND_DIR/certs"
    openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
        -keyout "$BACKEND_DIR/certs/key.pem" \
        -out "$BACKEND_DIR/certs/cert.pem" \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
    echo "  Certificates generated in: $BACKEND_DIR/certs/"
    echo ""
fi

# Run server
echo "[3/3] Starting SongMartyn server..."
echo ""
cd "$BACKEND_DIR"
./songmartyn "$@"
