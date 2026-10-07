package upload

import "testing"

func TestMatchesMagic(t *testing.T) {
	cases := []struct {
		ext  string
		data []byte
		ok   bool
	}{
		{".pdf", []byte("%PDF-1.4"), true},
		{".pdf", []byte("notpdf"), false},
		{".png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, true},
		{".png", []byte("xxxx"), false},
		{".jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{".jpeg", []byte{0xFF, 0xD8, 0xFF}, true},
		{".jpg", []byte{0x00, 0x01}, false},
		{".webp", []byte("RIFF....WEBP"), true},
		{".webp", []byte("RIFF....XXXX"), false},
		{".exe", []byte("%PDF"), false},
	}
	for _, tc := range cases {
		got := matchesMagic(tc.ext, tc.data)
		if got != tc.ok {
			t.Fatalf("matchesMagic(%q) = %v, want %v", tc.ext, got, tc.ok)
		}
	}
}
