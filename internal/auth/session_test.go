package auth

import (
	"testing"
	"time"

	"go04-pet-adoption/internal/model"
)

func TestSessionCreateGetInvalidate(t *testing.T) {
	sm := NewSessionManager(time.Hour)
	u := model.User{ID: "u_1", Username: "alice", Role: model.RoleAdopter}
	tok, err := sm.Create(u)
	if err != nil || tok == "" {
		t.Fatalf("create: %v %s", err, tok)
	}
	sess, err := sm.Get(tok)
	if err != nil {
		t.Fatal(err)
	}
	if sess.UserID != u.ID {
		t.Fatalf("got %+v", sess)
	}
	sm.Invalidate(tok)
	if _, err := sm.Get(tok); err == nil {
		t.Fatal("expected invalid after logout")
	}
}

func TestSessionExpired(t *testing.T) {
	sm := NewSessionManager(time.Millisecond)
	u := model.User{ID: "u_1", Username: "a", Role: model.RoleAdopter}
	tok, _ := sm.Create(u)
	time.Sleep(5 * time.Millisecond)
	if _, err := sm.Get(tok); err == nil {
		t.Fatal("expected expired")
	}
}
