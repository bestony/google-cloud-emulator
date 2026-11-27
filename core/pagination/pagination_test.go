package pagination

import "testing"

func TestRoundTrip(t *testing.T) {
	if off, err := Decode(""); err != nil || off != 0 {
		t.Fatalf("empty token: off=%d err=%v", off, err)
	}
	tok := Encode(42)
	off, err := Decode(tok)
	if err != nil || off != 42 {
		t.Fatalf("round trip failed: off=%d err=%v", off, err)
	}
	if _, err := Decode("!!!not-base64!!!"); err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestPageToken(t *testing.T) {
	if PageToken(0, 2, 4) == "" {
		t.Fatal("expected next token when more pages remain")
	}
	if PageToken(2, 2, 4) != "" {
		t.Fatal("expected no next token on last page")
	}
}

func TestEffectivePageSize(t *testing.T) {
	if EffectivePageSize(0, 50) != 50 {
		t.Fatal("default not applied")
	}
	if EffectivePageSize(10, 50) != 10 {
		t.Fatal("explicit size ignored")
	}
}
