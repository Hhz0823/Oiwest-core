package shadowsocks

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

var (
	ErrInvalidPassword = errors.New("shadowsocks: invalid password")
	ErrInvalidMethod   = errors.New("shadowsocks: unsupported encryption method")
	ErrShortPacket     = errors.New("shadowsocks: packet too short")
)

type CipherMethod string

const (
	MethodAES128GCM        CipherMethod = "aes-128-gcm"
	MethodAES256GCM        CipherMethod = "aes-256-gcm"
	MethodChaCha20Poly1305 CipherMethod = "chacha20-poly1305"
	MethodXChacha20Poly1305 CipherMethod = "xchacha20-poly1305"
	MethodNone             CipherMethod = "none"
)

type Cipher interface {
	KeySize() int
	SaltSize() int
	Encrypt(plaintext []byte, key []byte) ([]byte, error)
	Decrypt(ciphertext []byte, key []byte) ([]byte, error)
	NewEncryptWriter(w io.Writer, key []byte) (io.Writer, error)
	NewDecryptReader(r io.Reader, key []byte) (io.Reader, error)
	PacketEncrypt(plaintext []byte, key []byte) ([]byte, error)
	PacketDecrypt(ciphertext []byte, key []byte) ([]byte, error)
}

type AEADCipher struct {
	method   CipherMethod
	keySize  int
	saltSize int
}

func NewCipher(method CipherMethod) (Cipher, error) {
	switch method {
	case MethodAES128GCM:
		return &AEADCipher{method: method, keySize: 16, saltSize: 16}, nil
	case MethodAES256GCM:
		return &AEADCipher{method: method, keySize: 32, saltSize: 32}, nil
	case MethodChaCha20Poly1305:
		return &AEADCipher{method: method, keySize: 32, saltSize: 32}, nil
	case MethodXChacha20Poly1305:
		return &AEADCipher{method: method, keySize: 32, saltSize: 32}, nil
	case MethodNone:
		return &AEADCipher{method: method, keySize: 0, saltSize: 0}, nil
	default:
		return nil, ErrInvalidMethod
	}
}

func (c *AEADCipher) KeySize() int    { return c.keySize }
func (c *AEADCipher) SaltSize() int   { return c.saltSize }

func (c *AEADCipher) newAEAD(key []byte) (cipher.AEAD, error) {
	switch c.method {
	case MethodAES128GCM, MethodAES256GCM:
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	case MethodChaCha20Poly1305:
		return chacha20poly1305.New(key)
	case MethodXChacha20Poly1305:
		return chacha20poly1305.NewX(key)
	default:
		return nil, ErrInvalidMethod
	}
}

func KDFPassword(password string, keyLen int) []byte {
	key := make([]byte, keyLen)
	md5Hash := md5.Sum([]byte(password))
	copy(key, md5Hash[:])
	for len(key) < keyLen {
		md5Hash = md5.Sum(append(md5Hash[:], []byte(password)...))
		copy(key[len(key)-16:], md5Hash[:])
	}
	return key
}

func KDFSubKey(key, salt []byte, keyLen int) ([]byte, error) {
	h := hkdf.New(sha256.New, key, salt, []byte("ss-subkey"))
	subKey := make([]byte, keyLen)
	if _, err := io.ReadFull(h, subKey); err != nil {
		return nil, err
	}
	return subKey, nil
}

func GenerateSalt(size int) ([]byte, error) {
	salt := make([]byte, size)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func (c *AEADCipher) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if c.method == MethodNone {
		return plaintext, nil
	}
	aead, err := c.newAEAD(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func (c *AEADCipher) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if c.method == MethodNone {
		return ciphertext, nil
	}
	aead, err := c.newAEAD(key)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrShortPacket
	}
	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]
	return aead.Open(nil, nonce, data, nil)
}

type ssEncryptWriter struct {
	writer io.Writer
	cipher Cipher
	key    []byte
	aead   cipher.AEAD
	mu     sync.Mutex
}

type ssDecryptReader struct {
	reader  io.Reader
	cipher  Cipher
	key     []byte
	aead    cipher.AEAD
	buf     []byte
	offset  int
	length  int
	mu      sync.Mutex
}

func (c *AEADCipher) NewEncryptWriter(w io.Writer, key []byte) (io.Writer, error) {
	return &ssEncryptWriter{writer: w, cipher: c, key: key}, nil
}

func (c *AEADCipher) NewDecryptReader(r io.Reader, key []byte) (io.Reader, error) {
	return &ssDecryptReader{reader: r, cipher: c, key: key}, nil
}

func (w *ssEncryptWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	encrypted, err := w.cipher.Encrypt(p, w.key)
	if err != nil {
		return 0, err
	}
	length := len(encrypted)
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(length))
	if _, err := w.writer.Write(header); err != nil {
		return 0, err
	}
	if _, err := w.writer.Write(encrypted); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (r *ssDecryptReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.length > 0 {
		n := copy(p, r.buf[r.offset:r.offset+r.length])
		r.offset += n
		r.length -= n
		if r.length == 0 {
			r.offset = 0
		}
		return n, nil
	}

	header := make([]byte, 2)
	if _, err := io.ReadFull(r.reader, header); err != nil {
		return 0, err
	}
	length := int(binary.BigEndian.Uint16(header))
	if length > 64*1024 {
		return 0, errors.New("shadowsocks: packet too large")
	}

	encrypted := make([]byte, length)
	if _, err := io.ReadFull(r.reader, encrypted); err != nil {
		return 0, err
	}

	plaintext, err := r.cipher.Decrypt(encrypted, r.key)
	if err != nil {
		return 0, err
	}

	n := copy(p, plaintext)
	if n < len(plaintext) {
		r.buf = plaintext
		r.offset = n
		r.length = len(plaintext) - n
	}
	return n, nil
}

func (c *AEADCipher) PacketEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	return c.Encrypt(plaintext, key)
}

func (c *AEADCipher) PacketDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	return c.Decrypt(ciphertext, key)
}

type ShadowsocksUDP struct {
	cipher Cipher
	key    []byte
}

func NewShadowsocksUDP(method CipherMethod, password string) (*ShadowsocksUDP, error) {
	c, err := NewCipher(method)
	if err != nil {
		return nil, err
	}
	key := KDFPassword(password, c.KeySize())
	return &ShadowsocksUDP{cipher: c, key: key}, nil
}

func (su *ShadowsocksUDP) Pack(plaintext []byte) ([]byte, error) {
	return su.cipher.PacketEncrypt(plaintext, su.key)
}

func (su *ShadowsocksUDP) Unpack(ciphertext []byte) ([]byte, error) {
	return su.cipher.PacketDecrypt(ciphertext, su.key)
}
