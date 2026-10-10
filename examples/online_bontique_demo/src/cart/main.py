import os
import sqlite3
from pathlib import Path
from typing import List, Optional
from fastapi import FastAPI, HTTPException, Query
from pydantic import BaseModel

app = FastAPI(title="Cart")

DEFAULT_DB = str(Path(__file__).resolve().parent.parent.parent / "data" / "cart.db")
DB_PATH = os.environ.get("DB_PATH", DEFAULT_DB)

def get_db():
    p = Path(DB_PATH)
    p.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(str(p), check_same_thread=False)
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS cart_items (
            user_id TEXT NOT NULL,
            product_id TEXT NOT NULL,
            quantity INTEGER NOT NULL,
            PRIMARY KEY (user_id, product_id)
        )
        """
    )
    conn.commit()
    return conn

db_conn = get_db()

class CartItem(BaseModel):
    productId: str
    quantity: int

class AddItemRequest(BaseModel):
    userId: Optional[str] = "1"
    item: CartItem

class EmptyCartRequest(BaseModel):
    userId: Optional[str] = "1"

class Cart(BaseModel):
    userId: str
    items: List[CartItem]

@app.get("/healthz")
def healthz():
    return {"status": "ok"}

@app.post("/cart/add")
def add_item(req: AddItemRequest):
    user_id = req.userId or "1"
    cur = db_conn.cursor()
    cur.execute(
        """
        INSERT INTO cart_items (user_id, product_id, quantity)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id, product_id) DO UPDATE SET quantity = quantity + excluded.quantity
        """,
        (user_id, req.item.productId, req.item.quantity),
    )
    db_conn.commit()
    return {"status": "ok"}

@app.get("/cart", response_model=Cart)
def get_cart(userId: Optional[str] = Query(None)):
    user_id = userId or "1"
    cur = db_conn.cursor()
    cur.execute("SELECT product_id, quantity FROM cart_items WHERE user_id = ?", (user_id,))
    rows = cur.fetchall()
    items = [CartItem(productId=r[0], quantity=r[1]) for r in rows]
    return Cart(userId=user_id, items=items)

@app.post("/cart/empty")
def empty_cart(req: Optional[EmptyCartRequest] = None):
    user_id = req.userId if req and req.userId else "1"
    cur = db_conn.cursor()
    cur.execute("DELETE FROM cart_items WHERE user_id = ?", (user_id,))
    db_conn.commit()
    return {"status": "ok"}

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", "9002"))
    uvicorn.run(app, host="127.0.0.1", port=port)
