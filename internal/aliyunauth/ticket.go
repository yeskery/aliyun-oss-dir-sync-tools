package aliyunauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// STSCredentials 换取并解密后的临时访问凭证
type STSCredentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	AccessKeySecret string    `json:"access_key_secret"`
	SecurityToken   string    `json:"security_token"`
	Expiration      time.Time `json:"expiration,omitempty"`
}

type encryptResp struct {
	Code    interface{} `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    []string    `json:"data"`
}

// ExchangeTicketWithCookies 使用捕获到的 Cookie 列表与票据换取临时 STS 凭证
func ExchangeTicketWithCookies(cookieStr string, ticketVal string) (*STSCredentials, error) {
	if strings.TrimSpace(ticketVal) == "" && strings.TrimSpace(cookieStr) == "" {
		return nil, fmt.Errorf("登录凭据与 Cookie 为空，无法换取凭据")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("POST", "https://oss.console.aliyun.com/ajax/ak/encrypt.json", nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://oss.console.aliyun.com/")
	req.Header.Set("Origin", "https://oss.console.aliyun.com")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 优先确保带有 login_aliyunid_ticket
	finalCookie := strings.TrimSpace(cookieStr)
	if !strings.Contains(finalCookie, "login_aliyunid_ticket=") && ticketVal != "" {
		if finalCookie != "" {
			finalCookie = fmt.Sprintf("login_aliyunid_ticket=%s; %s", ticketVal, finalCookie)
		} else {
			finalCookie = fmt.Sprintf("login_aliyunid_ticket=%s", ticketVal)
		}
	}
	req.Header.Set("Cookie", finalCookie)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求阿里云凭据置换接口失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取接口返回内容失败: %w", err)
	}

	var res encryptResp
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("解析接口响应 JSON 失败 (状态码 %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if !res.Success || len(res.Data) < 2 {
		errMsg := res.Message
		if errMsg == "" {
			errMsg = fmt.Sprintf("置换失败，返回内容: %s", string(bodyBytes))
		}
		return nil, fmt.Errorf("阿里云鉴权服务返回错误: %s", errMsg)
	}

	var ak, sk, sts string
	if len(res.Data) == 2 {
		ak, err = DecryptPKCS1v15(res.Data[0])
		if err != nil {
			return nil, fmt.Errorf("解密 AccessKey ID 失败: %w", err)
		}
		sk, err = DecryptPKCS1v15(res.Data[1])
		if err != nil {
			return nil, fmt.Errorf("解密 AccessKey Secret 失败: %w", err)
		}
	} else {
		ak, sk, sts, err = DecryptCredentials(res.Data[0], res.Data[1], res.Data[2])
		if err != nil {
			return nil, fmt.Errorf("解密临时凭据失败: %w", err)
		}
	}

	return &STSCredentials{
		AccessKeyID:     ak,
		AccessKeySecret: sk,
		SecurityToken:   sts,
		Expiration:      time.Now().Add(6 * time.Hour), // STS 凭证通常有效期为数小时
	}, nil
}
