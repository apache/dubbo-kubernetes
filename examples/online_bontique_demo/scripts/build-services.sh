#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "=== Building Web UI ==="
(cd "$ROOT_DIR/web" && npm ci && npm run build)

echo "=== Building Rust Services ==="
cargo build --release

echo "=== Building Go Services ==="
mkdir -p "$ROOT_DIR/bin"
go build -o "$ROOT_DIR/bin/ad" ./src/ad
go build -o "$ROOT_DIR/bin/checkout" ./src/checkout
go build -o "$ROOT_DIR/bin/frontend" ./src/frontend
go build -o "$ROOT_DIR/bin/recommendation" ./src/recommendation

echo "=== Setting up Python venv ==="
if [ ! -d "$ROOT_DIR/.venv" ]; then
    python3 -m venv "$ROOT_DIR/.venv"
fi
"$ROOT_DIR/.venv/bin/pip" install -q -r "$ROOT_DIR/requirements.txt"

echo "=== Preparing Data and Run directories ==="
mkdir -p "$ROOT_DIR/data"
mkdir -p "$ROOT_DIR/.run"

# Stop existing processes if any
if [ -d "$ROOT_DIR/.run" ] && [ "$(ls -A "$ROOT_DIR/.run")" ]; then
    "$ROOT_DIR/scripts/stop.sh" || true
fi

echo "=== Starting Services ==="

# Ad (Go, 9001)
nohup env PORT=9001 "$ROOT_DIR/bin/ad" > "$ROOT_DIR/.run/ad.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/ad.pid"

# Cart (Python, 9002)
nohup env PORT=9002 DB_PATH="$ROOT_DIR/data/cart.db" "$ROOT_DIR/.venv/bin/python" "$ROOT_DIR/src/cart/main.py" > "$ROOT_DIR/.run/cart.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/cart.pid"

# Checkout (Go, 9003)
nohup env PORT=9003 \
CART_URL="http://127.0.0.1:9002" \
PRODUCT_CATALOGS_URL="http://127.0.0.1:9007" \
CURRENCY_URL="http://127.0.0.1:9004" \
SHIPPING_URL="http://127.0.0.1:9009" \
PAYMENT_URL="http://127.0.0.1:9006" \
EMAIL_URL="http://127.0.0.1:9005" \
"$ROOT_DIR/bin/checkout" > "$ROOT_DIR/.run/checkout.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/checkout.pid"

# Currency (Rust, 9004)
nohup env PORT=9004 DB_PATH="$ROOT_DIR/data/currency.db" "$ROOT_DIR/target/release/currency" > "$ROOT_DIR/.run/currency.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/currency.pid"

# Email (PHP, 9005)
nohup env PHP_CLI_SERVER_WORKERS=4 php -S 127.0.0.1:9005 -t "$ROOT_DIR/src/email" > "$ROOT_DIR/.run/email.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/email.pid"

# Payment (PHP, 9006)
nohup env PHP_CLI_SERVER_WORKERS=4 php -S 127.0.0.1:9006 -t "$ROOT_DIR/src/payment" > "$ROOT_DIR/.run/payment.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/payment.pid"

# Product Catalogs (Rust, 9007)
nohup env PORT=9007 DB_PATH="$ROOT_DIR/data/product-catalogs.db" "$ROOT_DIR/target/release/product-catalogs" > "$ROOT_DIR/.run/product-catalogs.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/product-catalogs.pid"

# Recommendation (Go, 9008)
nohup env PORT=9008 PRODUCT_CATALOGS_URL="http://127.0.0.1:9007" "$ROOT_DIR/bin/recommendation" > "$ROOT_DIR/.run/recommendation.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/recommendation.pid"

# Shipping (PHP, 9009)
nohup env PHP_CLI_SERVER_WORKERS=4 php -S 127.0.0.1:9009 -t "$ROOT_DIR/src/shipping" > "$ROOT_DIR/.run/shipping.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/shipping.pid"

# Frontend Gateway (Go, 8080)
nohup env PORT=8080 \
STATIC_DIR="$ROOT_DIR/web/dist" \
AD_URL="http://127.0.0.1:9001" \
CART_URL="http://127.0.0.1:9002" \
CHECKOUT_URL="http://127.0.0.1:9003" \
CURRENCY_URL="http://127.0.0.1:9004" \
EMAIL_URL="http://127.0.0.1:9005" \
PAYMENT_URL="http://127.0.0.1:9006" \
PRODUCT_CATALOGS_URL="http://127.0.0.1:9007" \
RECOMMENDATION_URL="http://127.0.0.1:9008" \
SHIPPING_URL="http://127.0.0.1:9009" \
"$ROOT_DIR/bin/frontend" > "$ROOT_DIR/.run/frontend.log" 2>&1 &
echo $! > "$ROOT_DIR/.run/frontend.pid"

echo "=== Waiting for services to be ready ==="
PORTS=(8080 9001 9002 9003 9004 9005 9006 9007 9008 9009)
for p in "${PORTS[@]}"; do
    READY=0
    for _ in {1..30}; do
        if curl -sSf "http://127.0.0.1:${p}/healthz" >/dev/null 2>&1; then
            READY=1
            break
        fi
        sleep 0.5
    done
    if [ "$READY" -ne 1 ]; then
        echo "Service on port $p failed to start! Check logs in .run/"
        exit 1
    fi
done

echo "All services healthy!"
echo "http://127.0.0.1:8080"
