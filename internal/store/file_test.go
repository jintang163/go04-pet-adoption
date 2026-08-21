package store

import (
	"os"
	"path/filepath"
	"testing"

	"go04-pet-adoption/internal/model"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	st := fs.Store()
	u, err := st.CreateUser(nil, model.User{
		Username:    "alice",
		DisplayName: "Alice",
		Role:        model.RoleAdopter,
		CreditScore: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs2.Store().GetUserByUsername(nil, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("id mismatch %s %s", got.ID, u.ID)
	}
}
