package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type TLSCertManager struct {
	dataPath string
}

var tlsCertMgr *TLSCertManager

func GetTLSCertManager() *TLSCertManager {
	if tlsCertMgr == nil {
		tlsCertMgr = &TLSCertManager{}
	}
	return tlsCertMgr
}

func (tm *TLSCertManager) SetDataPath(path string) {
	tm.dataPath = path
}

func (tm *TLSCertManager) GenerateCertificate(commonName string) (*TLSKeyPair, error) {
	if commonName == "" {
		commonName = "oiwest.local"
	}

	certsDir := filepath.Join(tm.dataPath, "certs")
	os.MkdirAll(certsDir, 0700)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("密钥生成失败: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Oiwest Core"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("证书生成失败: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyPKCS8, _ := x509.MarshalPKCS8PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyPKCS8})

	safeName := sanitizeFilename(commonName)
	certPath := filepath.Join(certsDir, safeName+".crt")
	keyPath := filepath.Join(certsDir, safeName+".key")

	os.WriteFile(certPath, certPEM, 0600)
	os.WriteFile(keyPath, keyPEM, 0600)

	return &TLSKeyPair{
		CertPEM:  string(certPEM),
		KeyPEM:   string(keyPEM),
		CertFile: certPath,
		KeyFile:  keyPath,
	}, nil
}

func (tm *TLSCertManager) GenerateRealityKeys() (map[string]string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("Reality密钥生成失败: %w", err)
	}

	keyPKCS8, _ := x509.MarshalPKCS8PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyPKCS8})

	pubPKCS8, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKCS8})

	return map[string]string{
		"privateKey": string(privPEM),
		"publicKey":  string(pubPEM),
	}, nil
}

func sanitizeFilename(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result = append(result, byte(c))
		} else {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "cert"
	}
	return string(result)
}
