/*
Copyright 2020 The ClusterKit Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pki

import (
	"context"
	"crypto"
	_ "crypto/x509"
	"crypto/x509/pkix"
	_ "fmt"
	"math/big"
	_ "net"
	_ "strings"
	"time"
)

var wellKnownCertificateTypes = map[string]string{
	"ca":           "CA,KeyUsageCRLSign,KeyUsageCertSign",
	"client":       "ExtKeyUsageClientAuth,KeyUsageDigitalSignature",
	"clientServer": "ExtKeyUsageClientAuth,ExtKeyUsageServerAuth,KeyUsageDigitalSignature,KeyUsageKeyEncipherment",
	"server":       "ExtKeyUsageServerAuth,KeyUsageDigitalSignature,KeyUsageKeyEncipherment",
}

type IssueCertRequest struct {
	// Signer is the keypair to use to sign. Ignored if Type is "CA", in which case the cert will be self-signed.
	Signer string
	// Type is the type of certificate i.e. CA, server, client etc.
	Type string
	// Subject is the certificate subject.
	Subject pkix.Name
	// AlternateNames is a list of alternative names for this certificate.
	AlternateNames []string

	// PublicKey is the public key for this certificate. If nil, it will be calculated from PrivateKey.
	PublicKey crypto.PublicKey
	// PrivateKey is the private key for this certificate. If both this and PublicKey are nil, a new private key will be generated.
	PrivateKey *PrivateKey
	// Validity is the certificate validity. The default is 10 years.
	Validity time.Duration

	// Serial is the certificate serial number. If nil, a random number will be generated.
	Serial *big.Int
}

type Keystore interface {
	// FindPrimaryKeypair finds a cert & private key, returning nil where either is not found
	// (if the certificate is found but not keypair, that is not an error: only the cert will be returned).
	// Also note that if the keypair is not found at all, this returns (nil, nil, nil)
	FindPrimaryKeypair(ctx context.Context, name string) (*Certificate, *PrivateKey, error)
}

// IssueCert issues a certificate, either a self-signed CA or from a CA in a keystore.
func IssueCert(ctx context.Context, request *IssueCertRequest, keystore Keystore) (issuedCertificate *Certificate, issuedKey *PrivateKey, caCertificate *Certificate, err error) {
	panic("excised: IssueCert")
}
