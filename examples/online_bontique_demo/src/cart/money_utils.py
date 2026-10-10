from dataclasses import dataclass

@dataclass
class Money:
    currencyCode: str
    units: int
    nanos: int

def is_valid(m: Money) -> bool:
    return _sign_matches(m) and _valid_nanos(m.nanos)

def _sign_matches(m: Money) -> bool:
    return m.nanos == 0 or m.units == 0 or (m.nanos < 0) == (m.units < 0)

def _valid_nanos(nanos: int) -> bool:
    return -999_999_999 <= nanos <= 999_999_999

def reset(m: Money):
    m.units = 0
    m.nanos = 0

def _trunc_div(a: int, b: int) -> int:
    return int(a / b)

def _trunc_mod(a: int, b: int) -> int:
    rem = abs(a) % abs(b)
    return -rem if a < 0 else rem

def sum_money(a: Money, b: Money) -> Money:
    if not is_valid(a) or not is_valid(b):
        raise ValueError("Invalid money value")
    if a.currencyCode != b.currencyCode:
        raise ValueError("Mismatching currency codes")

    units = a.units + b.units
    nanos = a.nanos + b.nanos

    if (units >= 0 and nanos >= 0) or (units < 0 and nanos <= 0):
        units += _trunc_div(nanos, 1_000_000_000)
        nanos = _trunc_mod(nanos, 1_000_000_000)
    else:
        if units > 0:
            units -= 1
            nanos += 1_000_000_000
        else:
            units += 1
            nanos -= 1_000_000_000

    return Money(currencyCode=a.currencyCode, units=units, nanos=nanos)

def multiply_slow(money: Money, multiplier: int) -> Money:
    result = Money(money.currencyCode, money.units, money.nanos)
    for _ in range(1, multiplier):
        result = sum_money(result, money)
    return result
