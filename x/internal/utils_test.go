package internal

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestParseEther(t *testing.T) {
	tests := []struct {
		in   json.Number
		want string
	}{
		{"0.585", "585000000000000000"},
		{"1", "1000000000000000000"},
		{"0", "0"},
		{".5", "500000000000000000"},
		{"10.000000000000000001", "10000000000000000001"},
		{"-0.5", "-500000000000000000"},
		{"  0.585  ", "585000000000000000"},
		{"1.0000000000000000009", "1000000000000000000"}, // 19th frac digit truncated
		{"1e-3", "1000000000000000"},                     // scientific-notation fallback
		{"", "0"},
	}
	for _, tc := range tests {
		got := ParseEther(tc.in)
		if got.String() != tc.want {
			t.Fatalf("ParseEther(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestParseUnits(t *testing.T) {
	got := parseUnits("123.45", 6)
	if got.String() != "123450000" {
		t.Fatalf("ParseUnits(123.45, 6) = %s, want 123450000", got)
	}
}

func TestRetainSignificantDigits(t *testing.T) {
	tests := []struct {
		num  int64
		sig  int
		want int64
	}{
		{123_456, 3, 123_000},
		{12, 3, 12},
		{-123_456, 3, -123_000},
		{0, 3, 0},
	}
	for _, tc := range tests {
		got := RetainSignificantDigits(big.NewInt(tc.num), tc.sig)
		if got.Int64() != tc.want {
			t.Fatalf("RetainSignificantDigits(%d, %d) = %d, want %d", tc.num, tc.sig, got.Int64(), tc.want)
		}
	}
}
