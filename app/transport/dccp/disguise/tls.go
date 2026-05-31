package disguise

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// TLSDisguise wraps DCCP traffic in TLS, making it appear as HTTPS.
type TLSDisguise struct {
	BaseDisguise
	tlsConfig *tls.Config
}

func NewTLSDisguise(config *Config) *TLSDisguise {
	d := &TLSDisguise{
		BaseDisguise: BaseDisguise{method: MethodTLS, config: config},
	}
	d.tlsConfig = &tls.Config{
		ServerName:         config.ServerName,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}
	if len(config.ALPN) > 0 {
		d.tlsConfig.NextProtos = config.ALPN
	} else {
		d.tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}
	return d
}

func (d *TLSDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(conn, d.tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func (d *TLSDisguise) Listen(addr net.Addr) (net.Listener, error) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{*cert}}
	return tls.Listen("tcp", addr.String(), tlsCfg)
}

func (d *TLSDisguise) WrapConn(conn net.Conn) net.Conn {
	return tls.Client(conn, d.tlsConfig)
}

func (d *TLSDisguise) Close() error { return nil }

func generateSelfSignedCert() (*tls.Certificate, error) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return &cert, err
}
