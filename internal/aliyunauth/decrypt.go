package aliyunauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// PrivateKeyPEM 阿里云 oss-browser 原生内置的用于解密 encrypt.json 凭据的 2048 位 RSA PKCS8 私钥
const PrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDOs8xrbXtRKmtI
8yKok6qXs1LPMy7a1sY/+Bf2YbWd1x06hVFvd0FiBkuMN6+VD6pi86PpETNXrYBu
Cee+jp90i5wyIO+DIl1DkqDGkzEJ/0j3N81L2xcxLBzmEbIM/iVCm920I/bEsbqf
zDpZKLGrGQDYH6zNdlo4qRE0QAengYnSDeLORBTbnIMLvY2S/LjvMAe6tLEDgEP1
e+UorAXz29UQC+kTixDKq0JbZbK4xjZ3XZLyuS44dlgcytlG2/QKrj4Oitc7p8Gu
UXoXhkksjtOAUicZmW+oEsU5cxmEXnaObpuN8Tu0HKi2OPMOuikXSXTNwFIGpxrV
JLaXCj3TAgMBAAECggEBAL3ipF7oA4orqsUoPmWYyfV0UhVwUpHNDfCEDBdV6Y7n
p70z7fy1FEZlvA9CtLT94Cvo4GHiYEUiaUItLUx8HvXkrwfKA912uzCxSatCWqFx
eGlLRPhze1m7yYx9IxEWg8ot/7xV8YL+NpBIAC4Z2SVx4bmpV4ikOfulLN3QGHgC
O1NaTW/H684ZhbK6J6kOw+crReI+4C5ZMb7a8oOXZb4TuBz5Fnm2HHiUaHriUTVK
TjLgf6j3/xi7c89XBcKKchTZ53bEIPIiJVCndDgC7Ck5i0R4sCAFmOu17ucO4FXm
izEx6OgtykQQloq5SJl9mjcfEt5oGxE62Xytwo0Y0HECgYEA+NivIvl60u2aHfCv
CHNItS4k+9HMtLyFWwZzf2QVKjchFogk2YeB0MS2dcgHf4QBi6rlD/rHI0l2OJss
c/vJOnE8HA6/XsSo8+u7X7CYHXWjA86jiiiF44FxJTkUlXJKQLbr/9H/OnCQBS/q
Gqo2h9Kx99skj5eFfkLkZnes3OsCgYEA1KT3fK9xPolTaWupL4cC5oeHKXbVyNRm
gFkQwFyYWXQ1EJGVduvun1csL+4MyK/+Nqrz0kKrdhApG2zW2L2ctEmG5h86dPSo
jBYwSUV7gKrZvwI0fB7k8tJc80rYqlhWaVYau39MYdsmx8xd8XkDTCMVa2FLxCnO
BjajSf3VyLkCgYB5/7GNUgZXVsmSGCeX2v0wZlkXZDoZ+brfb4A8WlLOCY0dQM3g
H8ETGv3hLJoQeCVR1gzLEsWClUdyZREcI7r5/f3cO5ay3XN/cnjAZxGMj9nFyg7/
cvKBITrZZuyigA/LPmLhXpDkk77n7B+3xu9JE5DQ1Q1NoJl/10emvNFw1QKBgHVe
XI/AkmPvDlAYsXhfL9WoJo1XXyaLPcxH3LwgH6L4RjXP9Ux/auUeC3ClFv9KTKlF
8HA0hkkTIG1XxLo65hyInOG1iOF3DPTNZnd613EsXCVWhTuhHcN9IvrSbAaR3Zgkg
3rMB4/Xk28iORQINC65CqCLQnOSk3nQfI5hgbdZAoGBAO/J1I3k/GbaAYjZf/Z1
HCuhu5dal+Xs13/I9t7jwqoKe18RTr8Tp8AGXCgpx/jzjCKQ0MZboyaaCkimUpnv
w136JrIdsKoPeDD6FGQd/rZAI/Fjj51kLYauK4YmknH0JmC17Dpf3ZtbRJuPUsWz
8g541AnLiZaeL/HgmVFpXJ8C
-----END PRIVATE KEY-----`

var (
	parsedPrivKey *rsa.PrivateKey
	parseOnce     sync.Once
	parseErr      error
)

func getPrivateKey() (*rsa.PrivateKey, error) {
	parseOnce.Do(func() {
		block, _ := pem.Decode([]byte(PrivateKeyPEM))
		if block == nil {
			parseErr = errors.New("解码 PEM 私钥失败")
			return
		}

		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			// 备选尝试 PKCS1
			rsaKey, err1 := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err1 != nil {
				parseErr = fmt.Errorf("解析 PKCS8/PKCS1 私钥失败: %v, %v", err, err1)
				return
			}
			parsedPrivKey = rsaKey
			return
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			parseErr = errors.New("私钥不是 RSA 密钥类型")
			return
		}
		parsedPrivKey = rsaKey
	})
	return parsedPrivKey, parseErr
}

// DecryptPKCS1v15 解密使用对应公钥加密的 Base64 密文（PKCS#1 v1.5 填充模式）
func DecryptPKCS1v15(cipherBase64 string) (string, error) {
	cipherBase64 = strings.TrimSpace(cipherBase64)
	if cipherBase64 == "" {
		return "", nil
	}

	privKey, err := getPrivateKey()
	if err != nil {
		return "", err
	}

	cipherBytes, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	plainBytes, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, cipherBytes)
	if err != nil {
		return "", fmt.Errorf("RSA PKCS1v15 解密失败: %w", err)
	}

	return string(plainBytes), nil
}

// DecryptCredentials 一次性解密返回的三段凭证 [encAK, encSK, encSTS]
func DecryptCredentials(encAK, encSK, encSTS string) (ak, sk, sts string, err error) {
	ak, err = DecryptPKCS1v15(encAK)
	if err != nil {
		return "", "", "", fmt.Errorf("解密 AccessKey ID 失败: %w", err)
	}

	sk, err = DecryptPKCS1v15(encSK)
	if err != nil {
		return "", "", "", fmt.Errorf("解密 AccessKey Secret 失败: %w", err)
	}

	sts, err = DecryptPKCS1v15(encSTS)
	if err != nil {
		return "", "", "", fmt.Errorf("解密 SecurityToken 失败: %w", err)
	}

	return ak, sk, sts, nil
}
