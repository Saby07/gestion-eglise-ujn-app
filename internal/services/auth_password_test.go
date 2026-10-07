package services

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
	if err := ValidatePassword("onlyletters"); err == nil {
		t.Fatal("expected error for letters-only password")
	}
	if err := ValidatePassword("12345678"); err == nil {
		t.Fatal("expected error for digits-only password")
	}
	if err := ValidatePassword("ChangeMe123!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
