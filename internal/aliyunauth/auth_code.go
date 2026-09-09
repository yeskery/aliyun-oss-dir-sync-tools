package aliyunauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ParsedAuthCode 授权码解析后的凭据与 Bucket 信息
type ParsedAuthCode struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SecurityToken   string `json:"security_token"`
	Endpoint        string `json:"endpoint"`
	BucketName      string `json:"bucket_name"`
	PathPrefix      string `json:"path_prefix,omitempty"`
}

// ParseAuthCode 解析授权码（支持 Base64 或原始 JSON 格式）
func ParseAuthCode(input string) (*ParsedAuthCode, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("授权码内容为空")
	}

	var jsonBytes []byte
	// 尝试 Base64 解码
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(decoded) > 0 {
		jsonBytes = decoded
	} else {
		// 尝试 URL 安全的 Base64 解码
		decodedURL, err2 := base64.URLEncoding.DecodeString(raw)
		if err2 == nil && len(decodedURL) > 0 {
			jsonBytes = decodedURL
		} else {
			// 直接作为 JSON 文本解析
			jsonBytes = []byte(raw)
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("无法将授权码解析为 JSON 对象: %w", err)
	}

	res := &ParsedAuthCode{}

	// 提取 AccessKey ID
	for _, key := range []string{"id", "ak", "accessKeyId", "access_key_id"} {
		if v, ok := data[key].(string); ok && strings.TrimSpace(v) != "" {
			res.AccessKeyID = strings.TrimSpace(v)
			break
		}
	}

	// 提取 AccessKey Secret
	for _, key := range []string{"secret", "sk", "accessKeySecret", "access_key_secret"} {
		if v, ok := data[key].(string); ok && strings.TrimSpace(v) != "" {
			res.AccessKeySecret = strings.TrimSpace(v)
			break
		}
	}

	// 提取 Security Token
	for _, key := range []string{"stoken", "stsToken", "securityToken", "security_token", "token"} {
		if v, ok := data[key].(string); ok && strings.TrimSpace(v) != "" {
			res.SecurityToken = strings.TrimSpace(v)
			break
		}
	}

	// 提取 Endpoint / Region
	for _, key := range []string{"endpoint", "eptpl", "region"} {
		if v, ok := data[key].(string); ok && strings.TrimSpace(v) != "" {
			ep := strings.TrimSpace(v)
			// 如果是模板形式如 http://{bucket}.oss-cn-hangzhou.aliyuncs.com
			if strings.Contains(ep, "{bucket}.") {
				ep = strings.ReplaceAll(ep, "{bucket}.", "")
			}
			ep = strings.TrimPrefix(ep, "https://")
			ep = strings.TrimPrefix(ep, "http://")
			ep = strings.TrimRight(ep, "/")

			// 如果单纯只填了 region，如 "oss-cn-hangzhou"
			if !strings.Contains(ep, ".aliyuncs.com") && strings.HasPrefix(ep, "oss-") {
				ep = ep + ".aliyuncs.com"
			}
			res.Endpoint = ep
			break
		}
	}

	// 提取 BucketName 与 PathPrefix
	for _, key := range []string{"osspath", "bucket", "bucketName", "bucket_name"} {
		if v, ok := data[key].(string); ok && strings.TrimSpace(v) != "" {
			rawPath := strings.TrimSpace(v)
			if strings.HasPrefix(rawPath, "oss://") {
				rawPath = strings.TrimPrefix(rawPath, "oss://")
				parts := strings.SplitN(rawPath, "/", 2)
				res.BucketName = parts[0]
				if len(parts) > 1 && parts[1] != "" {
					prefix := parts[1]
					if !strings.HasSuffix(prefix, "/") {
						prefix += "/"
					}
					res.PathPrefix = prefix
				}
			} else {
				res.BucketName = rawPath
			}
			break
		}
	}

	if res.AccessKeyID == "" || res.AccessKeySecret == "" {
		return nil, fmt.Errorf("授权码缺少关键认证字段 (AccessKey ID / Secret)")
	}

	return res, nil
}

// EnsureEndpoint 补全 endpoint 默认值
func (p *ParsedAuthCode) EnsureEndpoint() string {
	if p.Endpoint != "" {
		return p.Endpoint
	}
	return "oss-cn-hangzhou.aliyuncs.com"
}
