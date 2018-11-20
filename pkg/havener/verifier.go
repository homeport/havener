// Copyright © 2018 The Havener
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package havener

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerifiedCert is a struct with a parsed X.509 and/or an error if it cannot be parsed and verification failed
type VerifiedCert struct {
	Cert  *x509.Certificate
	Error error
}

// VerifyCertExpirations checks all certificates in all secrets in all namespaces
func VerifyCertExpirations() (err error) {
	InfoMessage("Going to check certificates...")

	var count int
	var countEmpty int

	VerboseMessage("Accessing cluster...")

	client, _, err := OutOfClusterAuthentication()
	if err != nil {
		ExitWithError("unable to get access to cluster", err)
	}

	VerboseMessage("Getting namespaces...")

	list, err := ListNamespaces(client)
	if err != nil {
		ExitWithError("unable to get a list of namespaces", err)
	}

	for _, namespace := range list {
		VerboseMessage(fmt.Sprintf("Getting secrets of namespace %s...", namespace))

		secretList, err := ListSecretsInNamespace(client, namespace)
		if err != nil {
			ExitWithError("unable to get a list of secrets", err)
		}

		if len(list) == 0 {
			VerboseMessage(fmt.Sprintf("No secrets in namespace %s", namespace))
		}

		for _, secret := range secretList {
			VerboseMessage(fmt.Sprintf("Accessing secret %s of namespace %s...", secret, namespace))

			nodeList, err := client.CoreV1().Secrets(namespace).Get(secret, v1.GetOptions{})
			if err != nil {
				ExitWithError("unable to access secrets", err)
			}

			results := GetCertificateFromSecret(nodeList.Data, namespace, secret)
			count = 0
			countEmpty = 0
			for key, cert := range results {
				var message string
				if cert.Error != nil {
					message = cert.Error.Error()
					count++
				} else if cert.Cert == nil {
					message = "empty certificate"
					countEmpty++
				} else {
					message = "valid"
				}

				line := fmt.Sprintf("%-18s %-26s %-39s %s", namespace, secret, key, message)

				if len(line) > GetTerminalWidth() {
					line = line[:GetTerminalWidth()-5] + "[...]"
				}

				fmt.Println(line)
			}
			if len(results) == 0 {
				VerboseMessage(fmt.Sprintf("No certificates in secret %s\n", secret))
			} else {
				VerboseMessage(fmt.Sprintf("Total nr. of certs in secret %s in namespace %s: %v; valid: %v; invalid: %v; empty: %v\n", secret, namespace, len(results), len(results)-count-countEmpty, count, countEmpty))
			}

		}
	}

	if count > 0 {
		ExitWithError("unable to verify certificates", fmt.Errorf("number of failed certs: %d", count))
	}

	return nil
}

// GetCertificateFromSecret looks for certificates inside the secrets and checks if they're valid
func GetCertificateFromSecret(datamap map[string][]uint8, namespace string, secret string) map[string]*VerifiedCert {
	result := map[string]*VerifiedCert{}

	shellRegexp := regexp.MustCompile(`.*(-cert|-crt)$`)
	for key, value := range datamap {
		if matches := shellRegexp.FindAllStringSubmatch(key, -1); len(matches) > 0 {
			if len(string(value)) == 0 {
				result[key] = &VerifiedCert{
					Cert:  nil,
					Error: nil,
				}
			} else {
				cert, err := GetCert(string(value))
				result[key] = &VerifiedCert{
					Cert:  cert,
					Error: err,
				}
			}
		}
	}

	return result
}

// GetCert gets a certificate and checks if it's valid
func GetCert(certificate string) (*x509.Certificate, error) {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(certificate))
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}

	block, _ := pem.Decode([]byte(certificate))
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate")
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err := cert.Verify(opts); err != nil {
		return cert, err
	}

	return cert, nil
}
