package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CertificateConfig struct {
	CommonName   string   `json:"commonName"`
	Organization []string `json:"organization"`
	DNSNames     []string `json:"dnsNames"`
	IPAddresses  []string `json:"ipAddresses"`
	KeyType      string   `json:"keyType"`
	KeySize      int      `json:"keySize"`
	ValidFrom    string   `json:"validFrom"`
	ValidFor     int      `json:"validFor"`
	IsCA         bool     `json:"isCA"`
	CAKeyFile    string   `json:"caKeyFile"`
	CACertFile   string   `json:"caCertFile"`
	OutputDir    string   `json:"outputDir"`
	CertFile     string   `json:"certFile"`
	KeyFile      string   `json:"keyFile"`
}

type KeyPair struct {
	Certificate tls.Certificate
	CertPEM     []byte
	KeyPEM      []byte
	Expiry      time.Time
}

type CertificateManager struct {
	config      *CertificateConfig
	certs       map[string]*KeyPair
	mu          sync.RWMutex
	cacheDir    string
	autoRenew   bool
	renewBefore time.Duration
}

var (
	ErrInvalidKeyType = errors.New("invalid key type")
	ErrCertNotFound   = errors.New("certificate not found")
	ErrCertExpired    = errors.New("certificate expired")
)

var defaultManager *CertificateManager
var managerMu sync.Mutex

func DefaultCertificateConfig() *CertificateConfig {
	return &CertificateConfig{
		CommonName:   "localhost",
		Organization: []string{"Oiwest Core"},
		DNSNames:     []string{"localhost"},
		KeyType:      "ecdsa",
		KeySize:      256,
		ValidFor:     365,
		OutputDir:    "",
		CertFile:     "certificate.crt",
		KeyFile:      "private.key",
	}
}

func GetCertificateManager() *CertificateManager {
	managerMu.Lock()
	defer managerMu.Unlock()
	if defaultManager == nil {
		defaultManager = NewCertificateManager(DefaultCertificateConfig())
	}
	return defaultManager
}

func NewCertificateManager(config *CertificateConfig) *CertificateManager {
	if config == nil {
		config = DefaultCertificateConfig()
	}
	return &CertificateManager{
		config:      config,
		certs:       make(map[string]*KeyPair),
		cacheDir:    config.OutputDir,
		autoRenew:   true,
		renewBefore: 30 * 24 * time.Hour,
	}
}

func (m *CertificateManager) SetCacheDir(dir string) {
	m.cacheDir = dir
}

func (m *CertificateManager) GenerateCertificate(config *CertificateConfig) (*KeyPair, error) {
	if config == nil {
		config = m.config
	}

	var privateKey crypto.PrivateKey
	var publicKey crypto.PublicKey
	var err error

	switch strings.ToLower(config.KeyType) {
	case "rsa":
		if config.KeySize == 0 {
			config.KeySize = 2048
		}
		privateKey, err = rsa.GenerateKey(rand.Reader, config.KeySize)
		if err != nil {
			return nil, fmt.Errorf("generate RSA key: %w", err)
		}
		publicKey = &privateKey.(*rsa.PrivateKey).PublicKey
	case "ecdsa", "ec":
		if config.KeySize == 0 {
			config.KeySize = 256
		}
		var curve elliptic.Curve
		switch config.KeySize {
		case 224:
			curve = elliptic.P224()
		case 256:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			curve = elliptic.P256()
		}
		privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate ECDSA key: %w", err)
		}
		publicKey = &privateKey.(*ecdsa.PrivateKey).PublicKey
	case "ed25519":
		_, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate Ed25519 key: %w", err)
		}
		publicKey = privateKey.(ed25519.PrivateKey).Public()
	default:
		return nil, ErrInvalidKeyType
	}

	notBefore := time.Now()
	if config.ValidFrom != "" {
		if parsed, err := time.Parse("2006-01-02", config.ValidFrom); err == nil {
			notBefore = parsed
		}
	}
	validFor := config.ValidFor
	if validFor <= 0 {
		validFor = 365
	}
	notAfter := notBefore.Add(time.Duration(validFor) * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: config.Organization,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	for _, dnsName := range config.DNSNames {
		template.DNSNames = append(template.DNSNames, dnsName)
	}
	if len(config.DNSNames) == 0 {
		template.DNSNames = []string{config.CommonName}
	}
	for _, ipStr := range config.IPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	if config.IsCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	var parent *x509.Certificate
	var parentKey crypto.PrivateKey
	parent = &template
	parentKey = privateKey

	if config.CACertFile != "" && config.CAKeyFile != "" {
		caCert, caKey, err := m.loadCA(config.CACertFile, config.CAKeyFile)
		if err == nil {
			parent = caCert
			parentKey = caKey
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, parent, publicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM, err := m.marshalPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("create TLS key pair: %w", err)
	}

	keyPair := &KeyPair{
		Certificate: tlsCert,
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		Expiry:      notAfter,
	}

	certKey := config.CommonName
	if len(config.DNSNames) > 0 {
		certKey = config.DNSNames[0]
	}
	m.mu.Lock()
	m.certs[certKey] = keyPair
	m.mu.Unlock()

	if config.OutputDir != "" {
		m.saveToFile(config, certPEM, keyPEM)
	}

	return keyPair, nil
}

func (m *CertificateManager) loadCA(certFile, keyFile string) (*x509.Certificate, crypto.PrivateKey, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("failed to parse CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("failed to parse CA key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return nil, nil, errors.New("failed to parse private key")
		}
		return cert, key2, nil
	}
	return cert, key, nil
}

func (m *CertificateManager) marshalPrivateKey(key crypto.PrivateKey) ([]byte, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}), nil
	case *ecdsa.PrivateKey:
		keyDER, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), nil
	case ed25519.PrivateKey:
		keyDER, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
	default:
		keyDER, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
	}
}

func (m *CertificateManager) saveToFile(config *CertificateConfig, certPEM, keyPEM []byte) error {
	if config.OutputDir == "" {
		return nil
	}
	os.MkdirAll(config.OutputDir, 0700)

	certPath := filepath.Join(config.OutputDir, config.CertFile)
	keyPath := filepath.Join(config.OutputDir, config.KeyFile)

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}
	return nil
}

func (m *CertificateManager) GetCertificate(serverName string) (*tls.Certificate, error) {
	m.mu.RLock()
	keyPair, ok := m.certs[serverName]
	m.mu.RUnlock()

	if ok {
		if time.Now().After(keyPair.Expiry) {
			if m.autoRenew {
				return m.renewCertificate(serverName)
			}
			return nil, ErrCertExpired
		}
		return &keyPair.Certificate, nil
	}

	cfg := *m.config
	cfg.CommonName = serverName
	cfg.DNSNames = []string{serverName}
	keyPair, err := m.GenerateCertificate(&cfg)
	if err != nil {
		return nil, err
	}
	return &keyPair.Certificate, nil
}

func (m *CertificateManager) renewCertificate(serverName string) (*tls.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keyPair, ok := m.certs[serverName]
	if !ok {
		return nil, ErrCertNotFound
	}

	cfg := *m.config
	cfg.CommonName = serverName
	cfg.DNSNames = []string{serverName}

	newKeyPair, err := m.GenerateCertificate(&cfg)
	if err != nil {
		return nil, err
	}

	m.certs[serverName] = newKeyPair
	_ = keyPair
	return &newKeyPair.Certificate, nil
}

func (m *CertificateManager) LoadFromFile(certFile, keyFile string) (*tls.Certificate, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		Certificate: tlsCert,
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		Expiry:      cert.NotAfter,
	}

	name := ""
	if len(cert.DNSNames) > 0 {
		name = cert.DNSNames[0]
	} else if len(cert.Subject.CommonName) > 0 {
		name = cert.Subject.CommonName
	}
	if name != "" {
		m.mu.Lock()
		m.certs[name] = keyPair
		m.mu.Unlock()
	}

	return &tlsCert, nil
}

func (m *CertificateManager) ListCertificates() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.certs))
	for name := range m.certs {
		names = append(names, name)
	}
	return names
}

func (m *CertificateManager) GetTLSConfig(serverName string) (*tls.Config, error) {
	cert, err := m.GetCertificate(serverName)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (m *CertificateManager) EnableAutoRenew(enable bool) {
	m.autoRenew = enable
}

func (m *CertificateManager) SetRenewBefore(d time.Duration) {
	m.renewBefore = d
}

func AutoGenerateCert(commonName string, dnsNames []string) (*tls.Certificate, error) {
	cfg := DefaultCertificateConfig()
	cfg.CommonName = commonName
	cfg.DNSNames = dnsNames
	cfg.ValidFor = 365
	manager := NewCertificateManager(cfg)
	keyPair, err := manager.GenerateCertificate(cfg)
	if err != nil {
		return nil, err
	}
	return &keyPair.Certificate, nil
}

func GenerateSelfSignedCertKey(commonName string, organization []string, dnsNames []string) (certPEM []byte, keyPEM []byte, err error) {
	cfg := DefaultCertificateConfig()
	cfg.CommonName = commonName
	cfg.Organization = organization
	cfg.DNSNames = dnsNames
	cfg.ValidFor = 36500
	cfg.KeyType = "ecdsa"
	cfg.KeySize = 256

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(cfg.ValidFor) * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: organization,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	template.DNSNames = dnsNames
	if len(dnsNames) == 0 {
		template.DNSNames = []string{commonName}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}

func CertificateFingerprint(cert *x509.Certificate) []byte {
	h := sha256.Sum256(cert.Raw)
	return h[:]
}

func IsCertificateExpired(cert *tls.Certificate) bool {
	if cert == nil || len(cert.Certificate) == 0 {
		return true
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return true
	}
	return time.Now().After(x509Cert.NotAfter)
}

func CertificateExpiresIn(cert *tls.Certificate, d time.Duration) bool {
	if cert == nil || len(cert.Certificate) == 0 {
		return true
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return true
	}
	return time.Now().Add(d).After(x509Cert.NotAfter)
}
