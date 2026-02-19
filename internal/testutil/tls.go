package testutil

import (
	"os"
	"testing"
)

// testCertPEM is a pre-generated self-signed certificate for testing.
// Valid for localhost and 127.0.0.1 until year 2125.
// Generated with:
//   openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem \
//     -days 36500 -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
const testCertPEM = `-----BEGIN CERTIFICATE-----
MIIDJzCCAg+gAwIBAgIUGQUaZliZYJFuT/jVCbQjKXjInUAwDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI1MTIxMjAwMjY0NVoYDzIxMjUx
MTE4MDAyNjQ1WjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQCjPGNViHwCF6MrkxXViWh+Oni+nwxJD3K2rRbYieN4
ar4QHvs+WFK0srKVXKKBveFw1qh42/1PPMjUP5UUy2D26axWXMUXBH9RpeYZCaJZ
1tPxcYNWjmNNhk+r/4xU6vWZlUlG9iy+eh440Bk1Iey6FoXZhDTZzE8n8pMS4mRl
rABMnF6gSSJTwqLVjq+rgrx9hW3qfM2x88fhFnPaHDJCnBSZ9GwMgsSHByF+WYrG
d3zQ3lgVyZd2mmWjpBa3yUWh5hBqt18avz3x7H+QAwfSbfzIRRjBWfCbfMIxqpUj
ejAx285z/q5tqvkgDsVoYGJN3tcs7ss45DCyfdtHbNNNAgMBAAGjbzBtMB0GA1Ud
DgQWBBQfpnxH8ydYCKxd7ZmnRnUBd6kXuDAfBgNVHSMEGDAWgBQfpnxH8ydYCKxd
7ZmnRnUBd6kXuDAPBgNVHRMBAf8EBTADAQH/MBoGA1UdEQQTMBGCCWxvY2FsaG9z
dIcEfwAAATANBgkqhkiG9w0BAQsFAAOCAQEAO5WQvxD8ySPGVdDVX6YnKQvmdmlj
s4qjAbVrHQrkr+WU/x1V6YNAou3j3hYfMFOsni9YqPW5aXqCpX4lWKICK/jTWW06
B3Y26K8HjVUqaSqiVbqbk+cU2ZUPRv7z6V1l/zUulfRnbyfjbuidEBzdDBbYcsy9
yIGu0Fkm9y4ur6LMJ6l2jM12eU7+bsGfWjEigZJ81mVouxITYtzw/v1ZxbS5d6PZ
sbHKF7ajuIEwrP1dfiKZlWVbhj7eqh/DDN+gcP7xZc65MHZx4schwgqElE2n9zlA
HtZwZdgWUzUVKTGUyPs397PCNk7b/i0p7I52pAIPUTgKwNZaFqRkhNwbmw==
-----END CERTIFICATE-----`

// testKeyPEM is the private key for testCertPEM.
const testKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCjPGNViHwCF6Mr
kxXViWh+Oni+nwxJD3K2rRbYieN4ar4QHvs+WFK0srKVXKKBveFw1qh42/1PPMjU
P5UUy2D26axWXMUXBH9RpeYZCaJZ1tPxcYNWjmNNhk+r/4xU6vWZlUlG9iy+eh44
0Bk1Iey6FoXZhDTZzE8n8pMS4mRlrABMnF6gSSJTwqLVjq+rgrx9hW3qfM2x88fh
FnPaHDJCnBSZ9GwMgsSHByF+WYrGd3zQ3lgVyZd2mmWjpBa3yUWh5hBqt18avz3x
7H+QAwfSbfzIRRjBWfCbfMIxqpUjejAx285z/q5tqvkgDsVoYGJN3tcs7ss45DCy
fdtHbNNNAgMBAAECggEAR7iRwJQOQHEYTV372u3bzpJsWPzSzgsOkPruLcgv/MmR
ps2mAFeYRzK4ym2ALVc7hXObZDbsTnNbnf4tx4wwl2xGP0/VfO6J7lrhBaE5BxYf
21bPqSk4smhP+zd19h2iOY/hOV+Se0J2oum0GadFNuIMf2zaN9PP97JaMpLsH8tG
PyDRNvRsTocq+59DcdQstB+jA9zHWMybLpM7cTE6RviwJTsqIQZ38nSw6xGX+8vJ
3h24HVqILRQrhmABCeNo9OTTeTFslTa0ISBMxvDLGkMyPM3pMaGL/KqQREB0ESn2
uk8+cyi+59HksMgyeTLmhEJr2X7px13apyPdu7FaVQKBgQDTLEb9QdwdBPZxaeCv
2ruR3gbYHVcvHQMOkGH7UtfX6ACIwzAKIWhOSWWVK0hh0CYqgyKcUAMIB4Fkxfgn
enjW/JqSxHV13o/CJoalbdGnrFgBuUNZlqsD6PduJGAQ+3TM8ckxuOpZgBsE9n+1
Km1ED1dQDvuaNNVtIZmdjq6RZwKBgQDF4xPvUMX8Jq4F9QjgbnZLXSRj906lni2L
IwEkSasKyqv7zkYk3/5wgdCjLxU4892tgKcLHDGa81b7a4ygrPugtHtPJv3o/48f
sO+lUDjzKRRLxBX49We+B5IYeEo4Uwhd8LVDy9PHGk9FdZc/z6kZU5CRgrmh4uQH
BtGLr48BKwKBgQDFLP2n8bveGMknwq26ZaloHQEU/+htJFC7Y5MpgQPrJESDboVH
oEPtfVjtfLmcIyfP4BwMCiWadK+b8cGw0wlL7BdEXU4z3bkrlp94jd8KCcEu8tZx
K17JStjlre9JTBGWX1j2JWwkX5pa+vLprRDTUOO87BB/vz9dI9d0p0pWlwKBgBEC
/nm6VerhYGB+ui6hEGZWMYSuhAJU4NFvRu/ZrWyWE8bn6rvMzdDaOBdvOsHUpR//
SVz5JYKOnNGsY0CE3nToTxl03qsjHSi6Sz/I77xnsaj5yHHIlwyNFhAodyj0amm5
Abw8T450QpBUFZaUwZK9zlXUCSVTngrEmUsK4p5VAoGAVUkL8MDwQMt3OPY1AcZB
/lX5Fx2LWALQa9Zt/JhdY9MSWXH6frkZqkRxCE89woUU5XF6cqd/1JA/JiKkhyKO
fM3D4pEVzZ9e02ZfJQdYIM9kgj6UkhzEE/fTqbdLejA/8Wqb2wtqB19yql8QOgqc
Le+uIaK4kWPVa/0EeyzfJ40=
-----END PRIVATE KEY-----`

// GenerateTestTLSCertificate writes the test certificate and key to temporary files.
// Returns the paths to the cert and key files.
// Files are automatically cleaned up via t.TempDir().
//
// This uses a pre-generated certificate valid until 2125, avoiding the complexity
// and overhead of dynamic certificate generation in tests.
func GenerateTestTLSCertificate(t *testing.T) (certPath, keyPath string) {
	t.Helper()

	// Write certificate to temporary file
	certFile, err := os.CreateTemp(t.TempDir(), "test-cert-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp cert file: %v", err)
	}
	certPath = certFile.Name()

	if _, err := certFile.WriteString(testCertPEM); err != nil {
		t.Fatalf("Failed to write certificate: %v", err)
	}
	if err := certFile.Close(); err != nil {
		t.Fatalf("Failed to close cert file: %v", err)
	}

	// Write private key to temporary file
	keyFile, err := os.CreateTemp(t.TempDir(), "test-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp key file: %v", err)
	}
	keyPath = keyFile.Name()

	if _, err := keyFile.WriteString(testKeyPEM); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}
	if err := keyFile.Close(); err != nil {
		t.Fatalf("Failed to close key file: %v", err)
	}

	return certPath, keyPath
}
