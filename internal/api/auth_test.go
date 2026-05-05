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
	
	if !CheckPasswordHash(password, hash) {
		t.Fatal("password check failed for correct password")
	}
	
	if CheckPasswordHash("wrong-password", hash) {
		t.Fatal("password check succeeded for wrong password")
	}
}
