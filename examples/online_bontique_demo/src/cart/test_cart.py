import json
from pathlib import Path
import pytest
from fastapi.testclient import TestClient
from main import app, db_conn
from money_utils import Money, is_valid, sum_money, multiply_slow, reset

client = TestClient(app)

def test_healthz():
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}

def test_cart_operations():
    # Empty cart first
    client.post("/cart/empty", json={"userId": "test-user"})
    
    # Get cart (should be empty)
    resp = client.get("/cart?userId=test-user")
    assert resp.status_code == 200
    assert resp.json()["items"] == []

    # Add item
    resp = client.post("/cart/add", json={"userId": "test-user", "item": {"productId": "PROD1", "quantity": 2}})
    assert resp.status_code == 200

    # Add same item (should accumulate)
    resp = client.post("/cart/add", json={"userId": "test-user", "item": {"productId": "PROD1", "quantity": 3}})
    assert resp.status_code == 200

    # Get cart
    resp = client.get("/cart?userId=test-user")
    assert resp.status_code == 200
    items = resp.json()["items"]
    assert len(items) == 1
    assert items[0]["productId"] == "PROD1"
    assert items[0]["quantity"] == 5

    # Empty cart
    client.post("/cart/empty", json={"userId": "test-user"})
    resp = client.get("/cart?userId=test-user")
    assert resp.json()["items"] == []

def load_vectors():
    path = Path("../../contracts/money_vectors.json")
    if not path.exists():
        path = Path("contracts/money_vectors.json")
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)

def test_money_validity():
    data = load_vectors()
    for v in data["validity_vectors"]:
        m = Money(**v["input"])
        assert is_valid(m) == v["expected_valid"], f"Branch: {v['branch']}"

def test_money_sum():
    data = load_vectors()
    for v in data["sum_vectors"]:
        a = Money(**v["a"])
        b = Money(**v["b"])
        if "expect_error" in v:
            with pytest.raises(ValueError):
                sum_money(a, b)
        else:
            res = sum_money(a, b)
            exp = Money(**v["expected"])
            assert res == exp, f"Branch: {v['branch']}"

def test_money_multiply():
    data = load_vectors()
    for v in data["multiply_slow_vectors"]:
        m = Money(**v["input"])
        res = multiply_slow(m, v["multiplier"])
        exp = Money(**v["expected"])
        assert res == exp, f"Branch: {v['branch']}"

def test_money_reset():
    data = load_vectors()
    for v in data["reset_vectors"]:
        m = Money(**v["input"])
        reset(m)
        exp = Money(**v["expected"])
        assert m == exp, f"Branch: {v['branch']}"
