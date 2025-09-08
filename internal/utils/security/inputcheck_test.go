package security

import "testing"

func TestValidateString_Basics(t *testing.T) {
	lim := DefaultLimits()
	if err := ValidateString("ok", "hello", lim); err != nil {
		t.Fatal(err)
	}
	if err := ValidateString("nul", "a\x00b", lim); err == nil {
		t.Fatal("expected NUL reject")
	}
	if err := ValidateString("nonprint", "a\u0007b", lim); err == nil {
		t.Fatal("expected control char reject")
	}
	if err := ValidateString("badutf8", string([]byte{0xff, 0xfe, 0xfd}), lim); err == nil {
		t.Fatal("expected invalid UTF-8 reject")
	}
}
