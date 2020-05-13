/*******************************************************************************
 * Copyright (c) 2020 IBM Corporation and others.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

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

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// GenerateCertificate : generates a key and certificate
// returns ServerKey ServerCert, error
func GenerateCertificate(dnsName string, certTitle string) (string, string, error) {
	var log = logf.Log.WithName("controller_codewind_tlsutils.go")

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano() / 1000000),
		Subject: pkix.Name{
			Organization: []string{certTitle},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 730),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{dnsName},
	}

	log.Info("Creating " + dnsName + " server Key")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	log.Info("Creating " + dnsName + " server certificate")
	certDerBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	if err != nil {
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
