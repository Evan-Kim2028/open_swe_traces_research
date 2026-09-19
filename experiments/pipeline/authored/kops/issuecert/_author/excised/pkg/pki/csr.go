/*
Copyright 2019 The ClusterKit Authors.

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
	crypto_rand "crypto/rand"
	"crypto/x509"
	_ "fmt"
	"math/big"
	"time"

	_ "k8s.io/klog/v2"
)

// BuildPKISerial produces a serial number for certs that is vanishingly unlikely to collide
// The timestamp should be provided as an input (time.Now().UnixNano()), and then we combine
// that with a 32 bit random crypto-rand integer.
// We also know that a bigger value was created later (modulo clock skew)
func BuildPKISerial(timestamp int64) *big.Int {
	randomLimit := new(big.Int).Lsh(big.NewInt(1), 32)
	randomComponent, err := crypto_rand.Int(crypto_rand.Reader, randomLimit)
	if err != nil {
		klog.Fatalf("error generating random number: %v", err)
	}

	serial := big.NewInt(timestamp)
	serial.Lsh(serial, 32)
	serial.Or(serial, randomComponent)

	return serial
}

func signNewCertificate(privateKey *PrivateKey, template *x509.Certificate, signer *x509.Certificate, signerPrivateKey *PrivateKey) (*Certificate, error) {
	panic("excised: signNewCertificate")
}
