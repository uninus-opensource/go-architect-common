package lib

import (
	"testing"
)

func TestGenerateKey(t *testing.T) {
	cert, private, public, err := GenerateKey(ECDSA, ES256)
	t.Logf("Certificate : %v\nPrivate key : %v\nPublic key : %v\nError : %v\n", cert, private, public, err)
	cert, private, public, err = GenerateKey(ECDSA, ES384)
	t.Logf("Certificate : %v\nPrivate key : %v\nPublic key : %v\nError : %v\n", cert, private, public, err)

	cert, private, public, err = GenerateKey(RSA, RSA2048)
	t.Logf("Certificate : %v\nPrivate key : %v\nPublic key : %v\nError : %v\n", cert, private, public, err)
	cert, private, public, err = GenerateKey(RSA, RSA3072)
	t.Logf("Certificate : %v\nPrivate key : %v\nPublic key : %v\nError : %v\n", cert, private, public, err)
}

func BenchmarkGenerateKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateKey(ECDSA, ES256)
		GenerateKey(ECDSA, ES384)
		GenerateKey(RSA, RSA2048)
		GenerateKey(RSA, RSA3072)
	}
}
