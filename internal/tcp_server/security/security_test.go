package security

import "testing"

func BenchmarkHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash("password")
	}
}

func BenchmarkComparePwHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ComparePwHash("password", "$2a$10$sJXc15p4plrW8Sds.o5d0uUzSKBFAha35f3e79Mgl6oCS1eeEWq6W")
	}
}
