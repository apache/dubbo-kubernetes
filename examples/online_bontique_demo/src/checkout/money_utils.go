package main

import (
	"errors"
)

type Money struct {
	CurrencyCode string `json:"currencyCode"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

var (
	ErrInvalidMoney       = errors.New("Invalid money value")
	ErrMismatchingCurrency = errors.New("Mismatching currency codes")
)

func IsValid(m Money) bool {
	return signMatches(m) && validNanos(m.Nanos)
}

func signMatches(m Money) bool {
	return m.Nanos == 0 || m.Units == 0 || (m.Nanos < 0) == (m.Units < 0)
}

func validNanos(nanos int32) bool {
	return nanos >= -999999999 && nanos <= 999999999
}

func Reset(m *Money) {
	m.Units = 0
	m.Nanos = 0
}

func Sum(a, b Money) (Money, error) {
	if !IsValid(a) || !IsValid(b) {
		return Money{}, ErrInvalidMoney
	}
	if a.CurrencyCode != b.CurrencyCode {
		return Money{}, ErrMismatchingCurrency
	}

	units := a.Units + b.Units
	nanos := a.Nanos + b.Nanos

	if (units >= 0 && nanos >= 0) || (units < 0 && nanos <= 0) {
		units += int64(nanos / 1000000000)
		nanos %= 1000000000
	} else {
		if units > 0 {
			units--
			nanos += 1000000000
		} else {
			units++
			nanos -= 1000000000
		}
	}

	return Money{
		CurrencyCode: a.CurrencyCode,
		Units:        units,
		Nanos:        nanos,
	}, nil
}

func MultiplySlow(money Money, multiplier int) (Money, error) {
	result := money
	for i := 1; i < multiplier; i++ {
		var err error
		result, err = Sum(result, money)
		if err != nil {
			return Money{}, err
		}
	}
	return result, nil
}
