pub mod money_utils;

use axum::{
    extract::State,
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use money_utils::Money;
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::Path;
use std::sync::{Arc, Mutex};

#[derive(Clone)]
struct AppState {
    db: Arc<Mutex<Connection>>,
}

#[derive(Serialize)]
struct HealthResponse {
    status: &'static str,
}

#[derive(Serialize)]
struct GetSupportedCurrenciesResponse {
    #[serde(rename = "currencyCodes")]
    currency_codes: Vec<String>,
}

#[derive(Deserialize)]
struct CurrencyConversionRequest {
    from: Money,
    #[serde(rename = "toCode")]
    to_code: String,
}

#[derive(Serialize)]
struct ErrorResponse {
    error: String,
}

fn init_db(db_path: &str) -> Connection {
    if let Some(parent) = Path::new(db_path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    let conn = Connection::open(db_path).expect("failed to open currency sqlite db");
    conn.execute(
        "CREATE TABLE IF NOT EXISTS currencies (
            code TEXT PRIMARY KEY,
            rate REAL NOT NULL
        )",
        [],
    )
    .expect("failed to create currencies table");

    let count: i64 = conn
        .query_row("SELECT count(*) FROM currencies", [], |row| row.get(0))
        .unwrap_or(0);

    if count == 0 {
        let default_rates: [(&str, f64); 33] = [
            ("EUR", 1.0),
            ("USD", 1.1305),
            ("JPY", 126.40),
            ("BGN", 1.9558),
            ("CZK", 25.592),
            ("DKK", 7.4609),
            ("GBP", 0.85970),
            ("HUF", 315.51),
            ("PLN", 4.2996),
            ("RON", 4.7463),
            ("SEK", 10.5375),
            ("CHF", 1.1360),
            ("ISK", 136.80),
            ("NOK", 9.8040),
            ("HRK", 7.4210),
            ("RUB", 74.4208),
            ("TRY", 6.1247),
            ("AUD", 1.6072),
            ("BRL", 4.2682),
            ("CAD", 1.5128),
            ("CNY", 7.5857),
            ("HKD", 8.8743),
            ("IDR", 15999.40),
            ("ILS", 4.0875),
            ("INR", 79.4320),
            ("KRW", 1275.05),
            ("MXN", 21.7999),
            ("MYR", 4.6289),
            ("NZD", 1.6679),
            ("PHP", 59.083),
            ("SGD", 1.5349),
            ("THB", 36.012),
            ("ZAR", 16.0583),
        ];
        for (code, rate) in default_rates {
            conn.execute(
                "INSERT INTO currencies (code, rate) VALUES (?1, ?2) ON CONFLICT(code) DO NOTHING",
                params![code, rate],
            )
            .expect("failed to insert seed currency");
        }
    }
    conn
}

async fn healthz() -> impl IntoResponse {
    (StatusCode::OK, Json(HealthResponse { status: "ok" }))
}

async fn get_currencies(State(state): State<AppState>) -> impl IntoResponse {
    let conn = state.db.lock().unwrap();
    let mut stmt = conn
        .prepare("SELECT code FROM currencies ORDER BY code")
        .unwrap();
    let codes: Vec<String> = stmt
        .query_map([], |row| row.get(0))
        .unwrap()
        .filter_map(|r| r.ok())
        .collect();

    (
        StatusCode::OK,
        Json(GetSupportedCurrenciesResponse {
            currency_codes: codes,
        }),
    )
}

async fn convert(
    State(state): State<AppState>,
    Json(req): Json<CurrencyConversionRequest>,
) -> impl IntoResponse {
    let conn = state.db.lock().unwrap();
    let mut stmt = conn.prepare("SELECT code, rate FROM currencies").unwrap();
    let rates: HashMap<String, f64> = stmt
        .query_map([], |row| Ok((row.get(0)?, row.get(1)?)))
        .unwrap()
        .filter_map(|r| r.ok())
        .collect();

    let from_rate = match rates.get(&req.from.currency_code) {
        Some(r) => *r,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::to_value(ErrorResponse {
                    error: format!("Unsupported currency: {}", req.from.currency_code),
                })
                .unwrap()),
            );
        }
    };

    let to_rate = match rates.get(&req.to_code) {
        Some(r) => *r,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::to_value(ErrorResponse {
                    error: format!("Unsupported currency: {}", req.to_code),
                })
                .unwrap()),
            );
        }
    };

    let total_nanos = req.from.units * 1_000_000_000 + req.from.nanos as i64;
    let converted_nanos = (total_nanos as f64) * to_rate / from_rate;

    let units = (converted_nanos / 1_000_000_000.0) as i64;
    let nanos = (converted_nanos % 1_000_000_000.0) as i32;

    let resp = Money {
        currency_code: req.to_code,
        units,
        nanos,
    };
    (StatusCode::OK, Json(serde_json::to_value(resp).unwrap()))
}

#[tokio::main]
async fn main() {
    let port = std::env::var("PORT").unwrap_or_else(|_| "9004".to_string());
    let db_path = std::env::var("DB_PATH").unwrap_or_else(|_| "data/currency.db".to_string());

    let conn = init_db(&db_path);
    let state = AppState {
        db: Arc::new(Mutex::new(conn)),
    };

    let app = Router::new()
        .route("/healthz", get(healthz))
        .route("/currencies", get(get_currencies))
        .route("/convert", post(convert))
        .with_state(state);

    let addr = format!("127.0.0.1:{}", port);
    println!("currency listening on {}", addr);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

#[cfg(test)]
mod tests {
    use super::money_utils::*;
    use serde::Deserialize;
    use std::fs;

    #[derive(Deserialize)]
    struct VectorFile {
        validity_vectors: Vec<ValVec>,
        sum_vectors: Vec<SumVec>,
        multiply_slow_vectors: Vec<MulVec>,
        reset_vectors: Vec<ResVec>,
    }

    #[derive(Deserialize)]
    struct ValVec {
        branch: String,
        input: Money,
        expected_valid: bool,
    }

    #[derive(Deserialize)]
    struct SumVec {
        branch: String,
        a: Money,
        b: Money,
        expected: Option<Money>,
        expect_error: Option<String>,
    }

    #[derive(Deserialize)]
    struct MulVec {
        branch: String,
        input: Money,
        multiplier: usize,
        expected: Money,
    }

    #[derive(Deserialize)]
    struct ResVec {
        branch: String,
        input: Money,
        expected: Money,
    }

    fn load_vectors() -> VectorFile {
        let content = fs::read_to_string("../../contracts/money_vectors.json")
            .or_else(|_| fs::read_to_string("contracts/money_vectors.json"))
            .expect("read money_vectors.json");
        serde_json::from_str(&content).expect("parse money_vectors.json")
    }

    #[test]
    fn test_validity() {
        let vf = load_vectors();
        for v in vf.validity_vectors {
            let got = is_valid(&v.input);
            assert_eq!(got, v.expected_valid, "branch: {}", v.branch);
        }
    }

    #[test]
    fn test_sum() {
        let vf = load_vectors();
        for v in vf.sum_vectors {
            let res = sum(&v.a, &v.b);
            if v.expect_error.is_some() {
                assert!(res.is_err(), "branch: {} expected error", v.branch);
            } else {
                assert_eq!(res.unwrap(), v.expected.unwrap(), "branch: {}", v.branch);
            }
        }
    }

    #[test]
    fn test_multiply_slow() {
        let vf = load_vectors();
        for v in vf.multiply_slow_vectors {
            let res = multiply_slow(&v.input, v.multiplier).unwrap();
            assert_eq!(res, v.expected, "branch: {}", v.branch);
        }
    }

    #[test]
    fn test_reset() {
        let vf = load_vectors();
        for v in vf.reset_vectors {
            let mut m = v.input;
            reset(&mut m);
            assert_eq!(m, v.expected, "branch: {}", v.branch);
        }
    }
}
