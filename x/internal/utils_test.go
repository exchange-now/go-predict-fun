package internal

import (
	"math/big"
	"testing"
)

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
