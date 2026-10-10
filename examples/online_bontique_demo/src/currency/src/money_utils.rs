use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Money {
    #[serde(rename = "currencyCode")]
    pub currency_code: String,
    pub units: i64,
    pub nanos: i32,
}

pub fn is_valid(m: &Money) -> bool {
    sign_matches(m) && valid_nanos(m.nanos)
}

fn sign_matches(m: &Money) -> bool {
    m.nanos == 0 || m.units == 0 || (m.nanos < 0) == (m.units < 0)
}

fn valid_nanos(nanos: i32) -> bool {
    nanos >= -999_999_999 && nanos <= 999_999_999
}

pub fn reset(m: &mut Money) {
    m.units = 0;
    m.nanos = 0;
}

pub fn sum(a: &Money, b: &Money) -> Result<Money, &'static str> {
    if !is_valid(a) || !is_valid(b) {
        return Err("Invalid money value");
    }
    if a.currency_code != b.currency_code {
        return Err("Mismatching currency codes");
    }

    let mut units = a.units + b.units;
    let mut nanos = a.nanos + b.nanos;

    if (units >= 0 && nanos >= 0) || (units < 0 && nanos <= 0) {
        units += (nanos / 1_000_000_000) as i64;
        nanos %= 1_000_000_000;
    } else {
        if units > 0 {
            units -= 1;
            nanos += 1_000_000_000;
        } else {
            units += 1;
            nanos -= 1_000_000_000;
        }
    }

    Ok(Money {
        currency_code: a.currency_code.clone(),
        units,
        nanos,
    })
}

pub fn multiply_slow(money: &Money, multiplier: usize) -> Result<Money, &'static str> {
    let mut result = money.clone();
    for _ in 1..multiplier {
        result = sum(&result, money)?;
    }
    Ok(result)
}
