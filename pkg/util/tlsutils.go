package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	logr "github.com/sirupsen/logrus"
)

// GenerateCertificate : generates a key and certificate
// returns ServerKey ServerCert, error
func GenerateCertificate(dnsName string, certTitle string) (string, string, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{certTitle},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 180),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{dnsName},
	}

	logr.Println("Creating " + dnsName + " server Key")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logr.Errorln("Unable to create server key")
		return "", "", err
	}

	logr.Println("Creating " + dnsName + " server certificate")
	certDerBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	if err != nil {
		logr.Errorf("Failed to create certificate: %s\n", err)
		return "", "", err
	}

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})
	pemPublicCert := out.String()
	out.Reset()
	pem.Encode(out, pemBlockForKey(privateKey))
	pemPrivateKey := out.String()
	out.Reset()

	return pemPrivateKey, pemPublicCert, nil
}

func publicKey(privateKey interface{}) interface{} {
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(privateKey interface{}) *pem.Block {
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	default:
		return nil
	}
}
