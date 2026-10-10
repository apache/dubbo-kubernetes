package main

import (
	"encoding/json"
	"os"
	"testing"
)

type MoneyVectorFile struct {
	ValidityVectors []struct {
		Branch        string `json:"branch"`
		Input         Money  `json:"input"`
		ExpectedValid bool   `json:"expected_valid"`
	} `json:"validity_vectors"`
	SumVectors []struct {
		Branch      string `json:"branch"`
		A           Money  `json:"a"`
		B           Money  `json:"b"`
		Expected    Money  `json:"expected"`
		ExpectError string `json:"expect_error"`
	} `json:"sum_vectors"`
	MultiplySlowVectors []struct {
		Branch     string `json:"branch"`
		Input      Money  `json:"input"`
		Multiplier int    `json:"multiplier"`
		Expected   Money  `json:"expected"`
	} `json:"multiply_slow_vectors"`
	ResetVectors []struct {
		Branch   string `json:"branch"`
		Input    Money  `json:"input"`
		Expected Money  `json:"expected"`
	} `json:"reset_vectors"`
}

func loadVectors(t *testing.T) MoneyVectorFile {
	data, err := os.ReadFile("../../contracts/money_vectors.json")
	if err != nil {
		t.Fatalf("failed to read money_vectors.json: %v", err)
	}
	var f MoneyVectorFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("failed to parse money_vectors.json: %v", err)
	}
	return f
}

func TestValidity(t *testing.T) {
	vectors := loadVectors(t)
	for _, v := range vectors.ValidityVectors {
		t.Run(v.Branch, func(t *testing.T) {
			got := IsValid(v.Input)
			if got != v.ExpectedValid {
				t.Errorf("IsValid(%+v) = %v, expected %v", v.Input, got, v.ExpectedValid)
			}
		})
	}
}

func TestSum(t *testing.T) {
	vectors := loadVectors(t)
	for _, v := range vectors.SumVectors {
		t.Run(v.Branch, func(t *testing.T) {
			got, err := Sum(v.A, v.B)
			if v.ExpectError != "" {
				if err == nil {
					t.Fatalf("Sum(%+v, %+v) expected error %q, got nil", v.A, v.B, v.ExpectError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Sum(%+v, %+v) unexpected error: %v", v.A, v.B, err)
			}
			if got != v.Expected {
				t.Errorf("Sum(%+v, %+v) = %+v, expected %+v", v.A, v.B, got, v.Expected)
			}
		})
	}
}

func TestMultiplySlow(t *testing.T) {
	vectors := loadVectors(t)
	for _, v := range vectors.MultiplySlowVectors {
		t.Run(v.Branch, func(t *testing.T) {
			got, err := MultiplySlow(v.Input, v.Multiplier)
			if err != nil {
				t.Fatalf("MultiplySlow(%+v, %d) unexpected error: %v", v.Input, v.Multiplier, err)
			}
			if got != v.Expected {
				t.Errorf("MultiplySlow(%+v, %d) = %+v, expected %+v", v.Input, v.Multiplier, got, v.Expected)
			}
		})
	}
}

func TestReset(t *testing.T) {
	vectors := loadVectors(t)
	for _, v := range vectors.ResetVectors {
		t.Run(v.Branch, func(t *testing.T) {
			m := v.Input
			Reset(&m)
			if m != v.Expected {
				t.Errorf("Reset(%+v) = %+v, expected %+v", v.Input, m, v.Expected)
			}
		})
	}
}
