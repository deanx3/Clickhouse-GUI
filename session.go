package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
)

// Everything secret in chgui is protected by one master key from CHGUI_SECRET.
// Each use derives its own key from it, so a stolen session cookie can never
// be used to read the connection store, and vice versa.
var (
	masterKey  = loadMasterKey()
	cookieAEAD = aeadFor("cookie")
	storeAEAD  = aeadFor("connection-store")
)

func loadMasterKey() []byte {
	key := make([]byte, 32)
	if s := os.Getenv("CHGUI_SECRET"); s != "" {
		b, err := hex.DecodeString(s)
		if err != nil || len(b) != len(key) {
			log.Fatalf("CHGUI_SECRET must be %d hex characters", len(key)*2)
		}
		copy(key, b)
		return key
	}

	if _, err := rand.Read(key); err != nil {
		log.Fatal(err)
	}
	log.Println("CHGUI_SECRET is unset: using an ephemeral key, so every restart signs " +
		"everyone out and makes saved connections unreadable")
	return key
}

func aeadFor(purpose string) cipher.AEAD {
	sum := sha256.Sum256(append([]byte(purpose+"\x00"), masterKey...))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		log.Fatal(err)
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatal(err)
	}
	return g
}

// sealBytes encrypts and authenticates plain. The nonce is prepended.
func sealBytes(a cipher.AEAD, plain []byte) ([]byte, error) {
	nonce := make([]byte, a.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return a.Seal(nonce, nonce, plain, nil), nil
}

func openBytes(a cipher.AEAD, raw []byte) ([]byte, error) {
	if len(raw) < a.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	return a.Open(nil, raw[:a.NonceSize()], raw[a.NonceSize():], nil)
}

/* --- the session cookie -------------------------------------------------- */

// The session cookie carries ClickHouse passwords, so it is encrypted and
// authenticated, not merely signed.
func seal(v any) (string, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	out, err := sealBytes(cookieAEAD, plain)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(out), nil
}

func unseal(tok string, v any) error {
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return err
	}
	plain, err := openBytes(cookieAEAD, raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, v)
}
