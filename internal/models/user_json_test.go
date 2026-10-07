package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPasswordHashOmittedFromJSON(t *testing.T) {
	u := User{
		FirstName:    "Jean",
		LastName:     "Dupont",
		Email:        "jean@example.com",
		PasswordHash: "$2a$10$secretshouldneverleak",
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "PasswordHash") || strings.Contains(s, "secretshouldneverleak") || strings.Contains(s, "$2a$") {
		t.Fatalf("password hash leaked in JSON: %s", s)
	}
}
