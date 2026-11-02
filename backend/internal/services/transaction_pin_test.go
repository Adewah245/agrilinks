package services

import (
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestTransactionPINValidationAndHashing(t *testing.T) {
	for _, test := range [][2]string{{"", ""}, {"12345", "12345"}, {"1234567", "1234567"}, {"abcdef", "abcdef"}, {"123456", "123455"}, {"１２３４５６", "１２３４５６"}} {
		if _, err := hashTransactionPIN(test[0], test[1]); err == nil {
			t.Fatal("invalid PIN accepted")
		}
	}
	hash, err := hashTransactionPIN("082491", "082491")
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte("082491")) != nil {
		t.Fatalf("PIN including leading zero not supported: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("82491")) == nil {
		t.Fatal("PIN was converted to a number")
	}
}
