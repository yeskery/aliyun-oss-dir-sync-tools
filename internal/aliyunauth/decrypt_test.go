package aliyunauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"
)

func TestDecryptCredentials(t *testing.T) {
	privKey, err := getPrivateKey()
	if err != nil {
		t.Fatalf("getPrivateKey failed: %v", err)
	}

	pubKey := &privKey.PublicKey

	testAK := "STS.LTAI5tTestAccessKey123"
	testSK := "TestSecretKey4567890abcdef"
	testSTS := "CAIS...SecurityTokenMockContent..."

	encAKBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(testAK))
	if err != nil {
		t.Fatalf("Encrypt AK failed: %v", err)
	}
	encSKBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(testSK))
	if err != nil {
		t.Fatalf("Encrypt SK failed: %v", err)
	}
	encSTSBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(testSTS))
	if err != nil {
		t.Fatalf("Encrypt STS failed: %v", err)
	}

	ak, sk, sts, err := DecryptCredentials(
		base64.StdEncoding.EncodeToString(encAKBytes),
		base64.StdEncoding.EncodeToString(encSKBytes),
		base64.StdEncoding.EncodeToString(encSTSBytes),
	)
	if err != nil {
		t.Fatalf("DecryptCredentials failed: %v", err)
	}

	if ak != testAK {
		t.Errorf("expected ak %s, got %s", testAK, ak)
	}
	if sk != testSK {
		t.Errorf("expected sk %s, got %s", testSK, sk)
	}
	if sts != testSTS {
		t.Errorf("expected sts %s, got %s", testSTS, sts)
	}
}
