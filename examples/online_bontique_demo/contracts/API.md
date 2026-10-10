# Microservices HTTP & JSON API Specification

This document defines the HTTP & JSON contracts for all services in this project.
Note on Migration: This system has been migrated from the original Apache Dubbo RPC protocol (using Dubbo triple protocol and registry) to standard HTTP + JSON RESTful communication without a service registry. Downstream endpoints are configured via environment variables.

All services listen on `127.0.0.1`.
All services provide `GET /healthz` returning HTTP 200 `{"status":"ok"}`.
Inter-service call timeout is 5 seconds. If downstream fails or times out, the service returns HTTP 502 with `{"error":"<message>"}`.

---

## 1. Gateway (`frontend`, Go, Port 8080)

The Go gateway hosts the built Vue 3 Single Page Application static assets from `web/dist` and proxies `/api/*` requests to backend services.

- `GET /healthz` -> 200 `{"status":"ok"}`
- `GET /api/products` -> forwards to Product Catalog Service `GET /products`
- `GET /api/products/:id` -> forwards to Product Catalog Service `GET /products/:id`
- `GET /api/cart?userId=:userId` -> forwards to Cart Service `GET /cart?userId=:userId`
- `POST /api/cart/add` -> forwards to Cart Service `POST /cart/add`
- `POST /api/cart/empty` -> forwards to Cart Service `POST /cart/empty`
- `POST /api/checkout` -> forwards to Checkout Service `POST /checkout`
- `GET /api/currencies` -> forwards to Currency Service `GET /currencies`
- `POST /api/currencies/convert` -> forwards to Currency Service `POST /convert`
- `POST /api/shipping/quote` -> forwards to Shipping Service `POST /quote`
- `POST /api/shipping/ship` -> forwards to Shipping Service `POST /ship`
- `POST /api/recommendations` -> forwards to Recommendation Service `POST /recommendations`
- `POST /api/ads` -> forwards to Ad Service `POST /ads`
- `GET /*` -> static files from `web/dist`, falling back to `web/dist/index.html` for SPA client-side routing.

---

## 2. Ad (`ad`, Go, Port 9001)

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /ads`
  - Request:
    ```json
    {
      "contextKeys": ["clothing", "accessories"]
    }
    ```
  - Response: 200
    ```json
    {
      "ads": [
        {
          "redirectUrl": "/product/66VCHSJNUP",
          "text": "Tank top for sale. 20% off."
        }
      ]
    }
    ```

---

## 3. Cart (`cart`, Python FastAPI, Port 9002)

Data store: SQLite (`data/cart.db`).

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /cart/add`
  - Request:
    ```json
    {
      "userId": "1",
      "item": {
        "productId": "OLJCESPC7Z",
        "quantity": 1
      }
    }
    ```
  - Response: 200
    ```json
    {
      "status": "ok"
    }
    ```

- `GET /cart?userId=:userId`
  - Response: 200
    ```json
    {
      "userId": "1",
      "items": [
        {
          "productId": "OLJCESPC7Z",
          "quantity": 1
        }
      ]
    }
    ```

- `POST /cart/empty`
  - Request:
    ```json
    {
      "userId": "1"
    }
    ```
  - Response: 200
    ```json
    {
      "status": "ok"
    }
    ```

---

## 4. Checkout (`checkout`, Go, Port 9003)

Orchestrates order placement by calling Cart, Product Catalog, Currency, Shipping, Payment, and Email services.

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /checkout`
  - Request:
    ```json
    {
      "userId": "1",
      "userCurrency": "USD",
      "address": {
        "streetAddress": "1600 Amphitheatre Parkway",
        "city": "Mountain View",
        "state": "CA",
        "country": "United States",
        "zipCode": 94043
      },
      "email": "someone@example.com",
      "creditCard": {
        "creditCardNumber": "4432-8015-6152-0454",
        "creditCardCvv": 123,
        "creditCardExpirationYear": 2026,
        "creditCardExpirationMonth": 12
      }
    }
    ```
  - Response: 200
    ```json
    {
      "order": {
        "orderId": "6b9b39e6-0567-4a0b-98f5-cf6d1cb89b88",
        "shippingTrackingId": "AB-23123-111234567",
        "shippingCost": {
          "currencyCode": "USD",
          "units": 8,
          "nanos": 990000000
        },
        "shippingAddress": {
          "streetAddress": "1600 Amphitheatre Parkway",
          "city": "Mountain View",
          "state": "CA",
          "country": "United States",
          "zipCode": 94043
        },
        "items": [
          {
            "item": {
              "productId": "OLJCESPC7Z",
              "quantity": 1
            },
            "cost": {
              "currencyCode": "USD",
              "units": 19,
              "nanos": 990000000
            }
          }
        ]
      }
    }
    ```

---

## 5. Currency (`currency`, Rust Axum, Port 9004)

Data store: SQLite (`data/currency.db`).

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `GET /currencies`
  - Response: 200
    ```json
    {
      "currencyCodes": ["USD", "EUR", "JPY", "GBP", "CNY"]
    }
    ```

- `POST /convert`
  - Request:
    ```json
    {
      "from": {
        "currencyCode": "USD",
        "units": 10,
        "nanos": 0
      },
      "toCode": "EUR"
    }
    ```
  - Response: 200
    ```json
    {
      "currencyCode": "EUR",
      "units": 8,
      "nanos": 845643520
    }
    ```

---

## 6. Email (`email`, PHP, Port 9005)

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /send-order-confirmation`
  - Request:
    ```json
    {
      "email": "someone@example.com",
      "order": "6b9b39e6-0567-4a0b-98f5-cf6d1cb89b88"
    }
    ```
  - Response: 200
    ```json
    {
      "message": "Order confirmation sent successfully!"
    }
    ```

---

## 7. Payment (`payment`, PHP, Port 9006)

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /charge`
  - Request:
    ```json
    {
      "amount": {
        "currencyCode": "USD",
        "units": 19,
        "nanos": 990000000
      },
      "creditCard": {
        "creditCardNumber": "4432-8015-6152-0454",
        "creditCardCvv": 123,
        "creditCardExpirationYear": 2026,
        "creditCardExpirationMonth": 12
      }
    }
    ```
  - Response: 200
    ```json
    {
      "transactionId": "f7d75efb-914b-4a57-8b9a-4c25f4625d80"
    }
    ```

---

## 8. Product Catalog (`product-catalogs`, Rust Axum, Port 9007)

Data store: SQLite (`data/product-catalogs.db`).

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `GET /products`
  - Response: 200
    ```json
    {
      "products": [
        {
          "id": "OLJCESPC7Z",
          "name": "Sunglasses",
          "description": "Add a modern touch to your outfits with these sleek aviator sunglasses.",
          "picture": "/img/products/sunglasses.jpg",
          "priceUsd": {
            "currencyCode": "USD",
            "units": 19,
            "nanos": 990000000
          },
          "categories": ["accessories"]
        }
      ]
    }
    ```

- `GET /products/:id`
  - Response: 200
    ```json
    {
      "id": "OLJCESPC7Z",
      "name": "Sunglasses",
      "description": "Add a modern touch to your outfits with these sleek aviator sunglasses.",
      "picture": "/img/products/sunglasses.jpg",
      "priceUsd": {
        "currencyCode": "USD",
        "units": 19,
        "nanos": 990000000
      },
      "categories": ["accessories"]
    }
    ```

---

## 9. Recommendation (`recommendation`, Go, Port 9008)

Downstream: Product Catalog Service.

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /recommendations`
  - Request:
    ```json
    {
      "userId": "1",
      "productIds": ["OLJCESPC7Z"]
    }
    ```
  - Response: 200
    ```json
    {
      "productIds": ["66VCHSJNUP", "1YMWWN1N4O", "L9ECAV7KIM", "2ZYFJ3GM2N"]
    }
    ```

---

## 10. Shipping (`shipping`, PHP, Port 9009)

- `GET /healthz`
  - Response: 200 `{"status":"ok"}`

- `POST /quote`
  - Request:
    ```json
    {
      "address": {
        "streetAddress": "1600 Amphitheatre Parkway",
        "city": "Mountain View",
        "state": "CA",
        "country": "United States",
        "zipCode": 94043
      },
      "items": [
        {
          "productId": "OLJCESPC7Z",
          "quantity": 1
        }
      ]
    }
    ```
  - Response: 200
    ```json
    {
      "costUsd": {
        "currencyCode": "USD",
        "units": 8,
        "nanos": 99000000
      }
    }
    ```

- `POST /ship`
  - Request:
    ```json
    {
      "address": {
        "streetAddress": "1600 Amphitheatre Parkway",
        "city": "Mountain View",
        "state": "CA",
        "country": "United States",
        "zipCode": 94043
      },
      "items": [
        {
          "productId": "OLJCESPC7Z",
          "quantity": 1
        }
      ]
    }
    ```
  - Response: 200
    ```json
    {
      "trackingId": "AB-23123-111234567"
    }
    ```
