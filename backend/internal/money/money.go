// Package money calculates amounts in integer kobo. Decimal strings are only
// used at database/provider boundaries; financial arithmetic never uses floats.
package money

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var decimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if !decimal.MatchString(s) {
		return 0, errors.New("enter an amount with at most two decimal places")
	}
	parts := strings.Split(s, ".")
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	digits := strings.TrimLeft(parts[0]+fraction, "0")
	if digits == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, errors.New("amount exceeds the supported limit")
	}
	return n, nil
}
func Positive(s string) (int64, error) {
	n, e := Parse(s)
	if e == nil && n <= 0 {
		e = errors.New("enter a positive amount")
	}
	return n, e
}
func Format(n int64) string {
	sign := ""
	value := uint64(n)
	if n < 0 {
		sign = "-"
		value = uint64(-(n + 1)) + 1
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}
func Add(values ...int64) (int64, error) {
	var n int64
	for _, v := range values {
		if v < 0 || n > math.MaxInt64-v {
			return 0, errors.New("amount exceeds the supported limit")
		}
		n += v
	}
	return n, nil
}
func BasisPoints(n, bps int64) (int64, error) {
	if n < 0 || bps < 0 || bps > 10000 {
		return 0, errors.New("invalid fee rate")
	}
	return n/10000*bps + (n%10000*bps+5000)/10000, nil
}

const FundingFee int64 = 5000

func WithdrawalFee(n int64) (int64, error) {
	if n < 100 || n%100 != 0 {
		return 0, errors.New("withdrawals must be in whole naira, starting at ₦1")
	}
	if n <= 500000 {
		return 5000, nil
	}
	if n < 1000000 {
		return 20000, nil
	}
	return 30000, nil
}
