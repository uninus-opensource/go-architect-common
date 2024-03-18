package lib

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

type algorithm string
type size string

const (
	// ===== [START ALGORITHM] =====

	// ECDSA ..
	ECDSA = algorithm("ECDSA")
	// RSA ..
	RSA = algorithm("RSA")

	// ===== [END ALGORITHM] =====

	// ===== [START SIZE] =====

	// RSA512 ..
	RSA512 = size("512")
	// RSA1024 ..
	RSA1024 = size("1024")
	// RSA2048 ..
	RSA2048 = size("2048")
	// RSA3072 ..
	RSA3072 = size("3072")

	// ES224 ..
	ES224 = size("ES224")
	// ES256 ..
	ES256 = size("ES256")
	// ES384 ..
	ES384 = size("ES384")
	// ES521 ..
	ES521 = size("ES521")

	// ===== [END SIZE] =====

	// ===== [START TYPE KEY] =====

	// Certificate ..
	Certificate string = "CERTIFICATE"
	// PublicKey ..
	PublicKey string = "PUBLIC KEY"
	// PrivateKeyEC ..
	PrivateKeyEC string = "EC PRIVATE KEY"
	// PrivateKeyRSA ..
	PrivateKeyRSA string = "RSA PRIVATE KEY"

	// ===== [END TYPE KEY] =====
)

func (s size) convertToInt() (int, error) {
	var err error
	var rsa int64

	switch s {
	case RSA512, RSA1024, RSA2048, RSA3072:
		rsa, err = strconv.ParseInt(string(s), 10, 64)
		if err != nil {
			return 0, err
		}
		return int(rsa), nil
	default:
		return 0, nil
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

// GenerateKey ..
func GenerateKey(algo algorithm, size size) (cert []byte, prv []byte, pub []byte, err error) {
	var priv interface{}

	switch algo {
	case RSA:
		sizeInt, err := size.convertToInt()
		if err == nil {
			priv, err = rsa.GenerateKey(rand.Reader, sizeInt)
		}
	case ECDSA:
		switch size {
		case ES224:
			priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
		case ES256:
			priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		case ES384:
			priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		case ES521:
			priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		default:
			return nil, nil, nil, errors.New(fmt.Sprint("Unrecognized elliptic curve :", size))
		}
	default:
		return nil, nil, nil, errors.New(fmt.Sprint("Unrecognized algorithm :", algo))
	}

	if err != nil {
		return nil, nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(365) * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Uninus"}, // to be hardcode
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	var derBytes, publicBytes, privBytes []byte
	switch algo {
	case RSA:
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
		if err != nil {
			return nil, nil, nil, err
		}
		publicBytes = x509.MarshalPKCS1PublicKey(publicKey(priv).(*rsa.PublicKey))
		privBytes = x509.MarshalPKCS1PrivateKey(priv.(*rsa.PrivateKey))
	case ECDSA:
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
		if err != nil {
			return nil, nil, nil, err
		}
		publicBytes, err = x509.MarshalPKIXPublicKey(publicKey(priv).(*ecdsa.PublicKey))
		if err != nil {
			return nil, nil, nil, err
		}
		privBytes, err = x509.MarshalECPrivateKey(priv.(*ecdsa.PrivateKey))
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var derPem, publicPem, privPem pem.Block
	var derByte, publicByte, privByte bytes.Buffer
	derPem = pem.Block{
		Type:  Certificate,
		Bytes: derBytes,
	}
	err = pem.Encode(&derByte, &derPem)
	if err != nil {
		return nil, nil, nil, err
	}
	publicPem = pem.Block{
		Type:  PublicKey,
		Bytes: publicBytes,
	}
	err = pem.Encode(&publicByte, &publicPem)
	if err != nil {
		return nil, nil, nil, err
	}

	switch algo {
	case RSA:
		privPem = pem.Block{
			Type:  PrivateKeyRSA,
			Bytes: privBytes,
		}
		err = pem.Encode(&privByte, &privPem)
		if err != nil {
			return nil, nil, nil, err
		}
	case ECDSA:
		privPem = pem.Block{
			Type:  PrivateKeyEC,
			Bytes: privBytes,
		}
		err = pem.Encode(&privByte, &privPem)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return derByte.Bytes(), privByte.Bytes(), publicByte.Bytes(), nil
}
