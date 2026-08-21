package auth

import "testing"

func TestPasswordHashVerify(t *testing.T) {
	h := NewPasswordHasher()
	salt, hash, it, err := h.Hash("alice123")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Verify("alice123", salt, hash, it) {
		t.Fatal("expected verify success")
	}
	if h.Verify("wrong", salt, hash, it) {
		t.Fatal("expected verify fail")
	}
}

func TestPasswordEmpty(t *testing.T) {
	h := NewPasswordHasher()
	if _, _, _, err := h.Hash(""); err == nil {
		t.Fatal("empty password should fail")
	}
}
