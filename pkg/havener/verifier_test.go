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

package havener_test

import (
	"encoding/json"
	"io/ioutil"
	"os"

	. "github.com/homeport/havener/pkg/havener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func errorCount(certMap map[string]*VerifiedCert) (count int) {
	for _, cert := range certMap {
		if cert.Error != nil {
			count++
		}
	}

	return
}

var _ = Describe("Valid?", func() {
	Context("check certificates", func() {
		It("should check whether the certificate is valid, and return true", func() {
			input := `
-----BEGIN CERTIFICATE-----
MIIGjTCCBXWgAwIBAgIQB1bcrVT4VnHncFBc5Ox0GDANBgkqhkiG9w0BAQsFADBg
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMR8wHQYDVQQDExZHZW9UcnVzdCBUTFMgUlNBIENBIEcx
MB4XDTE4MDMxNjAwMDAwMFoXDTIwMDMxNTEyMDAwMFowezELMAkGA1UEBhMCVVMx
ETAPBgNVBAgTCE5ldyBZb3JrMQ8wDQYDVQQHEwZBcm1vbmsxNDAyBgNVBAoTK0lu
dGVybmF0aW9uYWwgQnVzaW5lc3MgTWFjaGluZXMgQ29ycG9yYXRpb24xEjAQBgNV
BAMMCSouaWJtLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANjv
ihG7I0exBs0XEwTzySVnjI06R1dY5Q9mMhwtxVKPRzZpDs0r4YleaJyh2+AwwB3V
qfuYyiGrbrGb6X6S8+Ik0xmXkp6A8UI9m23ZOF5YopoV0+rda9F8k3mw3SDKbRA4
DJIXZeykHmGDff0kr27AGIkWlgZTWT3d2yzew0fTNg5swQ1TTqcJFobc5eo6XXYR
NSAGVoxs/cr7ONCFipFsRYDXaa0ZVc2MOC0tMlBi/5k2shjTDF8hGoBfUv3s4uuE
fd4ZG/8m2rQSnUYxMmcyV9AhoMo0BCoz1qdq+c1GQ9auvEBNvcAe/NDtxcr36ccf
NPHWbE/oKTDRolC/OPUCAwEAAaOCAyYwggMiMB8GA1UdIwQYMBaAFJRP1F2L5KTi
poD+/dj5AO+jvgJXMB0GA1UdDgQWBBS+ot+5azUF1uKuRUf4xbLBTavNKTAdBgNV
HREEFjAUggkqLmlibS5jb22CB2libS5jb20wDgYDVR0PAQH/BAQDAgWgMB0GA1Ud
JQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjA/BgNVHR8EODA2MDSgMqAwhi5odHRw
Oi8vY2RwLmdlb3RydXN0LmNvbS9HZW9UcnVzdFRMU1JTQUNBRzEuY3JsMEwGA1Ud
IARFMEMwNwYJYIZIAYb9bAEBMCowKAYIKwYBBQUHAgEWHGh0dHBzOi8vd3d3LmRp
Z2ljZXJ0LmNvbS9DUFMwCAYGZ4EMAQICMHYGCCsGAQUFBwEBBGowaDAmBggrBgEF
BQcwAYYaaHR0cDovL3N0YXR1cy5nZW90cnVzdC5jb20wPgYIKwYBBQUHMAKGMmh0
dHA6Ly9jYWNlcnRzLmdlb3RydXN0LmNvbS9HZW9UcnVzdFRMU1JTQUNBRzEuY3J0
MAkGA1UdEwQCMAAwggF+BgorBgEEAdZ5AgQCBIIBbgSCAWoBaAB1AKS5CZC0GFgU
h7sTosxncAo8NZgE+RvfuON3zQ7IDdwQAAABYjB51hEAAAQDAEYwRAIgBjf3pU/s
4ebuDH8njpvZeGFRmDOkzshfALM0iPSb6ncCIGXbcn7ZlCQjjbCAl0MaIGRCg/Ha
p61q/fM3Ldq/aMTHAHcAb1N2rDHwMRnYmQCkURX/dxUcEdkCwQApBo2yCJo32RMA
AAFiMHnXgwAABAMASDBGAiEAyuqf0fCgssHtDK+ZoDmoEulGmR8IlC+oinpCVmI+
NvsCIQDhDhpjp+DQ6So9sAKJmhzyMI5H1iOvrNeX88cc5IgZDgB2ALvZ37wfinG1
k5Qjl6qSe0c4V5UKq1LoGpCWZDaOHtGFAAABYjB51t0AAAQDAEcwRQIhAL2iaiFI
EehgEE9OSSLXiKJsWdGLw6dPcFH78bX/L4lJAiAWyVKBCiyWOEz0GuUYbJKeNcSs
5TrdPwRAkmYfUBQAozANBgkqhkiG9w0BAQsFAAOCAQEAkx45MJBT+BLAasI3392g
N7M0ARaCmIWr1/oDqJpJCc+vRWfzuoKPNn5EPgmtTr6ea1Bd3ttdA3ys4IlRlrc8
TP58OewboGjXOdLczHDT8Kqvco/+71d779HDtf3zRfLOj6j0io0H5tnCkNcpeAuN
seJH0GV7y3bcxcVEIslQsUwHcRSiGIWoS+7tyLldXpouTSQz+XBzclXXXfTqQQG0
a9rbC4BLhJ0QfZlN6yjvIhs0R0aQV3fFYcqFhuAXWlNAlX/74tCR9fq2XzFaDgIR
w58BqAkjgyZmRCI25KRC4wypPo2Vx2Ixzb1R3lLkYi/sXtqntb9KO1fF69zx7rtQ
ag==
-----END CERTIFICATE-----`

			cert, err := GetCert(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(cert).NotTo(BeNil())
		})

		It("should return false", func() {
			input := `
-----BEGIN CERTIFICATE-----
MIIGgTCCBWmgAwIBAgIIP2loSeAn4ucwDQYJKoZIhvcNAQEFBQAwSTELMAkGA1UE
BhMCVVMxEzARBgNVBAoTCkdvb2dsZSBJbmMxJTAjBgNVBAMTHEdvb2dsZSBJbnRl
cm5ldCBBdXRob3JpdHkgRzIwHhcNMTQwNTIyMTEyODU3WhcNMTQwODIwMDAwMDAw
WjBmMQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwN
TW91bnRhaW4gVmlldzETMBEGA1UECgwKR29vZ2xlIEluYzEVMBMGA1UEAwwMKi5n
b29nbGUuY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEQ80mW9KOdkTavOvJ
T8KdnZW/ClBvM2DNSYlXEjlHxLfN23DIgwfk7xnThlwyH4RTk4bhhtWtBTyR9Gh4
3BIE5aOCBBkwggQVMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjCCAuIG
A1UdEQSCAtkwggLVggwqLmdvb2dsZS5jb22CDSouYW5kcm9pZC5jb22CFiouYXBw
ZW5naW5lLmdvb2dsZS5jb22CEiouY2xvdWQuZ29vZ2xlLmNvbYIWKi5nb29nbGUt
YW5hbHl0aWNzLmNvbYILKi5nb29nbGUuY2GCCyouZ29vZ2xlLmNsgg4qLmdvb2ds
ZS5jby5pboIOKi5nb29nbGUuY28uanCCDiouZ29vZ2xlLmNvLnVrgg8qLmdvb2ds
ZS5jb20uYXKCDyouZ29vZ2xlLmNvbS5hdYIPKi5nb29nbGUuY29tLmJygg8qLmdv
b2dsZS5jb20uY2+CDyouZ29vZ2xlLmNvbS5teIIPKi5nb29nbGUuY29tLnRygg8q
Lmdvb2dsZS5jb20udm6CCyouZ29vZ2xlLmRlggsqLmdvb2dsZS5lc4ILKi5nb29n
bGUuZnKCCyouZ29vZ2xlLmh1ggsqLmdvb2dsZS5pdIILKi5nb29nbGUubmyCCyou
Z29vZ2xlLnBsggsqLmdvb2dsZS5wdIIPKi5nb29nbGVhcGlzLmNughQqLmdvb2ds
ZWNvbW1lcmNlLmNvbYIRKi5nb29nbGV2aWRlby5jb22CDSouZ3N0YXRpYy5jb22C
CiouZ3Z0MS5jb22CDCoudXJjaGluLmNvbYIQKi51cmwuZ29vZ2xlLmNvbYIWKi55
b3V0dWJlLW5vY29va2llLmNvbYINKi55b3V0dWJlLmNvbYIWKi55b3V0dWJlZWR1
Y2F0aW9uLmNvbYILKi55dGltZy5jb22CC2FuZHJvaWQuY29tggRnLmNvggZnb28u
Z2yCFGdvb2dsZS1hbmFseXRpY3MuY29tggpnb29nbGUuY29tghJnb29nbGVjb21t
ZXJjZS5jb22CCnVyY2hpbi5jb22CCHlvdXR1LmJlggt5b3V0dWJlLmNvbYIUeW91
dHViZWVkdWNhdGlvbi5jb20wCwYDVR0PBAQDAgeAMGgGCCsGAQUFBwEBBFwwWjAr
BggrBgEFBQcwAoYfaHR0cDovL3BraS5nb29nbGUuY29tL0dJQUcyLmNydDArBggr
BgEFBQcwAYYfaHR0cDovL2NsaWVudHMxLmdvb2dsZS5jb20vb2NzcDAdBgNVHQ4E
FgQUZ+wFAJG6n8knT4i1EhyqBhTlMxgwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAW
gBRK3QYWG7z2aLV29YG2u2IaulqBLzAXBgNVHSAEEDAOMAwGCisGAQQB1nkCBQEw
MAYDVR0fBCkwJzAloCOgIYYfaHR0cDovL3BraS5nb29nbGUuY29tL0dJQUcyLmNy
bDANBgkqhkiG9w0BAQUFAAOCAQEAFId/P3amOfPZtGwUDvIZlfp4kUJ/Qr/y9KMc
syO7YdcO+mSwOarZtZ1UdB3zBJ3d7vn2Ld1G0TiqFW8vIZk1OtWtdMC6hFQuC21P
Papck9jRhLZO1Jx4uFbGQdWM25z+a1TzxaoULmhAN9FF38OFKcrZlb/Gf4uETYV7
mMFQ10GT6UBESCkvEsT4hgEONQ/wXiOxDgMrbXBBm67IfXJzxmpncPDG6o49Dqw4
F6Jkkotp7ca6OvBnTvi0hcd4qS/64c/+0SjjLsWFq04W/zRJAUvF7mt8yiZHmv8f
E+FdDynG49hiV4MhWpmLdY5xzOWqb7+xmPdo3947SoHe9ZO2Mg==
-----END CERTIFICATE-----`

			_, err := GetCert(input)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error if, for some reason, input is not a certificate", func() {
			input := "hello"
			_, err := GetCert(input)
			Expect(err.Error()).To(BeEquivalentTo("failed to parse root certificate"))
		})

		pwDir, _ := os.Getwd()

		It("should return no error if the certificate's valid -- from file", func() {
			fileContent, err := ioutil.ReadFile(pwDir + "/../../test/valid_cert.json")
			Expect(err).NotTo(HaveOccurred())

			var datamap map[string][]uint8
			err = json.Unmarshal(fileContent, &datamap)
			Expect(err).NotTo(HaveOccurred())

			certMap := GetCertificateFromSecret(datamap, "namespace", "secret")
			Expect(errorCount(certMap)).To(BeEquivalentTo(0))
		})

		It("should return an error if the certificate's invalid -- from file", func() {
			fileContent, err := ioutil.ReadFile(pwDir + "/../../test/invalid_cert.json")
			Expect(err).NotTo(HaveOccurred())

			var datamap map[string][]uint8
			err = json.Unmarshal(fileContent, &datamap)
			Expect(err).NotTo(HaveOccurred())

			certMap := GetCertificateFromSecret(datamap, "namespace", "secret")
			Expect(errorCount(certMap)).To(BeEquivalentTo(1))
		})

		It("should return an error if it's not a certificate -- from file", func() {
			fileContent, err := ioutil.ReadFile(pwDir + "/../../test/not_a_cert.json")
			Expect(err).NotTo(HaveOccurred())

			var datamap map[string][]uint8
			err = json.Unmarshal(fileContent, &datamap)
			Expect(err.Error()).To(BeEquivalentTo("illegal base64 data at input byte 0"))
		})

		It("should return no errors when empty certificates  -- from file", func() {
			fileContent, err := ioutil.ReadFile(pwDir + "/../../test/valid_cert_with_empty_keys.json")
			Expect(err).NotTo(HaveOccurred())

			var datamap map[string][]uint8
			err = json.Unmarshal(fileContent, &datamap)
			Expect(err).NotTo(HaveOccurred())

			certMap := GetCertificateFromSecret(datamap, "namespace", "secret")
			Expect(errorCount(certMap)).To(BeEquivalentTo(0))
		})
	})
})
