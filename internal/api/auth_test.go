package api

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "my-secret-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == password {
		t.Fatal("hash should not be equal to password")
	}

	if !CheckPassword(hash, password) {
		t.Fatal("password check failed for correct password")
	}

	if CheckPassword(hash, "wrong-password") {
		t.Fatal("password check succeeded for wrong password")
	}
}
