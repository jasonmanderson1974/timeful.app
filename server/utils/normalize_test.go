package utils

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"  Foo@Bar.COM ":    "foo@bar.com",
		"USER@EXAMPLE.com":  "user@example.com",
		"already@lower.com": "already@lower.com",
		"":                  "",
		"   ":               "",
		"\tMixed@Case.io\n": "mixed@case.io",
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
