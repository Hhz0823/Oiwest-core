package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

var (
	ErrInvalidKey   = errors.New("invalid key length")
	ErrDecryptFail  = errors.New("decryption failed")
	ErrEncryptFail  = errors.New("encryption failed")
)

type CipherType int

const (
	CipherNone       CipherType = 0
	CipherAES128GCM  CipherType = 1
	CipherAES256GCM  CipherType = 2
	CipherChacha20Poly1305 CipherType = 3
	CipherXChacha20Poly1305 CipherType = 4
)

type AEADCipher struct {
	aead cipher.AEAD
	typ  CipherType
}

func NewAEADCipher(typ CipherType, key []byte) (*AEADCipher, error) {
	var aead cipher.AEAD
	var err error
	switch typ {
	case CipherAES128GCM:
		if len(key) != 16 {
			return nil, ErrInvalidKey
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err = cipher.NewGCM(block)
	case CipherAES256GCM:
		if len(key) != 32 {
			return nil, ErrInvalidKey
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err = cipher.NewGCM(block)
	case CipherChacha20Poly1305:
		if len(key) != 32 {
			return nil, ErrInvalidKey
		}
		aead, err = chacha20poly1305.New(key)
	case CipherXChacha20Poly1305:
		if len(key) != 32 {
			return nil, ErrInvalidKey
		}
		aead, err = chacha20poly1305.NewX(key)
	default:
		return nil, errors.New("unsupported cipher type")
	}
	if err != nil {
		return nil, err
	}
	return &AEADCipher{aead: aead, typ: typ}, nil
}

func (c *AEADCipher) NonceSize() int {
	return c.aead.NonceSize()
}

func (c *AEADCipher) Overhead() int {
	return c.aead.Overhead()
}

func (c *AEADCipher) Seal(dst, nonce, plaintext, additional []byte) []byte {
	return c.aead.Seal(dst, nonce, plaintext, additional)
}

func (c *AEADCipher) Open(dst, nonce, ciphertext, additional []byte) ([]byte, error) {
	return c.aead.Open(dst, nonce, ciphertext, additional)
}

func (c *AEADCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (c *AEADCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptFail
	}
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]
	return c.aead.Open(nil, nonce, encrypted, nil)
}

func GenerateKey(length int) ([]byte, error) {
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func HKDFSecrets(secret, salt, info []byte, length int) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}

func MD5Sum(data []byte) [16]byte {
	return md5.Sum(data)
}

func SHA256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

type SessionCounter struct {
	count [8]byte
}

func NewSessionCounter(initial uint64) *SessionCounter {
	sc := &SessionCounter{}
	binary.BigEndian.PutUint64(sc.count[:], initial)
	return sc
}

func (sc *SessionCounter) Next() []byte {
	val := binary.BigEndian.Uint64(sc.count[:])
	binary.BigEndian.PutUint64(sc.count[:], val+1)
	return sc.count[:]
}

func XORKeyStream(dst, src []byte) {
	for i := range src {
		dst[i] = src[i] ^ 0xFF
	}
}
