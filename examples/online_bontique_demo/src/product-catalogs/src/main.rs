use axum::{
    extract::{Path as AxPath, State},
    http::StatusCode,
    response::IntoResponse,
    routing::get,
    Json, Router,
};
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use std::sync::{Arc, Mutex};

#[derive(Clone)]
struct AppState {
    db: Arc<Mutex<Connection>>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Money {
    #[serde(rename = "currencyCode")]
    pub currency_code: String,
    pub units: i64,
    pub nanos: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Product {
    pub id: String,
    pub name: String,
    pub description: String,
    pub picture: String,
    #[serde(rename = "priceUsd")]
    pub price_usd: Money,
    pub categories: Vec<String>,
}

#[derive(Serialize, Deserialize)]
struct ProductsWrapper {
    products: Vec<Product>,
}

#[derive(Serialize)]
struct ListProductsResponse {
    products: Vec<Product>,
}

#[derive(Serialize)]
struct HealthResponse {
    status: &'static str,
}

fn init_db(db_path: &str) -> Connection {
    if let Some(parent) = Path::new(db_path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    let conn = Connection::open(db_path).expect("failed to open product catalogs sqlite db");
    conn.execute(
        "CREATE TABLE IF NOT EXISTS products (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            description TEXT NOT NULL,
            picture TEXT NOT NULL,
            price_currency TEXT NOT NULL,
            price_units INTEGER NOT NULL,
            price_nanos INTEGER NOT NULL,
            categories TEXT NOT NULL
        )",
        [],
    )
    .expect("failed to create products table");

    let count: i64 = conn
        .query_row("SELECT count(*) FROM products", [], |row| row.get(0))
        .unwrap_or(0);

    if count == 0 {
        let raw_json = include_str!("../products.json");
        let parsed: ProductsWrapper = serde_json::from_str(raw_json).expect("parse products.json");
        for p in parsed.products {
            let cat_json = serde_json::to_string(&p.categories).unwrap();
            conn.execute(
                "INSERT INTO products (id, name, description, picture, price_currency, price_units, price_nanos, categories)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
                 ON CONFLICT(id) DO NOTHING",
                params![
                    p.id,
                    p.name,
                    p.description,
                    p.picture,
                    p.price_usd.currency_code,
                    p.price_usd.units,
                    p.price_usd.nanos,
                    cat_json
                ],
            )
            .expect("insert product");
        }
    }

    conn
}

async fn healthz() -> impl IntoResponse {
    (StatusCode::OK, Json(HealthResponse { status: "ok" }))
}

async fn list_products(State(state): State<AppState>) -> impl IntoResponse {
    let conn = state.db.lock().unwrap();
    let mut stmt = conn
        .prepare("SELECT id, name, description, picture, price_currency, price_units, price_nanos, categories FROM products ORDER BY id")
        .unwrap();

    let prods: Vec<Product> = stmt
        .query_map([], |row| {
            let cat_str: String = row.get(7)?;
            let cats: Vec<String> = serde_json::from_str(&cat_str).unwrap_or_default();
            Ok(Product {
                id: row.get(0)?,
                name: row.get(1)?,
                description: row.get(2)?,
                picture: row.get(3)?,
                price_usd: Money {
                    currency_code: row.get(4)?,
                    units: row.get(5)?,
                    nanos: row.get(6)?,
                },
                categories: cats,
            })
        })
        .unwrap()
        .filter_map(|r| r.ok())
        .collect();

    (StatusCode::OK, Json(ListProductsResponse { products: prods }))
}

async fn get_product(
    State(state): State<AppState>,
    AxPath(id): AxPath<String>,
) -> impl IntoResponse {
    let conn = state.db.lock().unwrap();
    let mut stmt = conn
        .prepare("SELECT id, name, description, picture, price_currency, price_units, price_nanos, categories FROM products WHERE id = ?1")
        .unwrap();

    let res = stmt.query_row(params![id], |row| {
        let cat_str: String = row.get(7)?;
        let cats: Vec<String> = serde_json::from_str(&cat_str).unwrap_or_default();
        Ok(Product {
            id: row.get(0)?,
            name: row.get(1)?,
            description: row.get(2)?,
            picture: row.get(3)?,
            price_usd: Money {
                currency_code: row.get(4)?,
                units: row.get(5)?,
                nanos: row.get(6)?,
            },
            categories: cats,
        })
    });

    match res {
        Ok(p) => (StatusCode::OK, Json(serde_json::to_value(p).unwrap())),
        Err(_) => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({ "error": "Product not found" })),
        ),
    }
}

#[tokio::main]
async fn main() {
    let port = std::env::var("PORT").unwrap_or_else(|_| "9007".to_string());
    let db_path = std::env::var("DB_PATH").unwrap_or_else(|_| "data/product-catalogs.db".to_string());

    let conn = init_db(&db_path);
    let state = AppState {
        db: Arc::new(Mutex::new(conn)),
    };

    let app = Router::new()
        .route("/healthz", get(healthz))
        .route("/products", get(list_products))
        .route("/products/:id", get(get_product))
        .with_state(state);

    let addr = format!("127.0.0.1:{}", port);
    println!("product-catalogs listening on {}", addr);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_products_json_parse() {
        let raw_json = include_str!("../products.json");
        let parsed: ProductsWrapper = serde_json::from_str(raw_json).expect("parse products.json");
        assert!(!parsed.products.is_empty());
        assert_eq!(parsed.products[0].id, "OLJCESPC7Z");
    }
}
