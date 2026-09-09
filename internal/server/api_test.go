package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/manager"
)

func TestAuthAndStatusAPI(t *testing.T) {
	tempCfg := filepath.Join(t.TempDir(), "test_config.json")
	cm, err := config.NewConfigManager(tempCfg)
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	ws := manager.NewWorkspaceService(cm)
	handler := NewAPIHandler(cm, ws)

	// 1. Test /api/status
	reqStatus := httptest.NewRequest("GET", "/api/status", nil)
	wStatus := httptest.NewRecorder()
	handler.GetStatus(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Errorf("GetStatus status code = %d, want 200", wStatus.Code)
	}

	// 2. Test /api/auth/parse-code
	rawAuthJSON := `{"id":"TEST_AK_123","secret":"TEST_SK_456","stoken":"TEST_TOKEN","eptpl":"https://{bucket}.oss-cn-shanghai.aliyuncs.com","osspath":"oss://my-demo-bucket/images/"}`
	b64Code := base64.StdEncoding.EncodeToString([]byte(rawAuthJSON))

	bodyBytes, _ := json.Marshal(map[string]string{"code": b64Code})
	reqParse := httptest.NewRequest("POST", "/api/auth/parse-code", bytes.NewReader(bodyBytes))
	wParse := httptest.NewRecorder()
	handler.ParseAuthCode(wParse, reqParse)

	if wParse.Code != http.StatusOK {
		t.Errorf("ParseAuthCode status code = %d, want 200", wParse.Code)
	}

	var resp JSONResponse
	if err := json.Unmarshal(wParse.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal parse response failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success = true, got false")
	}

	// 3. Test /api/auth/login-status
	reqLoginStatus := httptest.NewRequest("GET", "/api/auth/login-status?target=bucket_a", nil)
	wLoginStatus := httptest.NewRecorder()
	handler.GetAccountLoginStatus(wLoginStatus, reqLoginStatus)
	if wLoginStatus.Code != http.StatusOK {
		t.Errorf("GetAccountLoginStatus code = %d, want 200", wLoginStatus.Code)
	}

	// 4. Test Config Save & Status with CDNDomain
	savePayload := map[string]interface{}{
		"bucket_a": map[string]interface{}{
			"name":            "Bucket A",
			"access_key_id":   "test_ak_a",
			"access_key_secret": "test_sk_a",
			"endpoint":        "oss-cn-hangzhou.aliyuncs.com",
			"bucket_name":     "test-bucket-a",
			"cdn_domain":      "cdn-a.example.com",
		},
		"bucket_b": map[string]interface{}{
			"name":            "Bucket B",
			"access_key_id":   "test_ak_b",
			"access_key_secret": "test_sk_b",
			"endpoint":        "oss-cn-shenzhen.aliyuncs.com",
			"bucket_name":     "test-bucket-b",
			"cdn_domain":      "assets-dev.dongvoyage.cn",
		},
		"sync_dirs": []string{"static/"},
	}
	saveBytes, _ := json.Marshal(savePayload)
	reqSave := httptest.NewRequest("POST", "/api/config", bytes.NewReader(saveBytes))
	wSave := httptest.NewRecorder()
	handler.SaveConfig(wSave, reqSave)
	if wSave.Code != http.StatusOK {
		t.Errorf("SaveConfig code = %d, want 200", wSave.Code)
	}

	// Check status has cdn_domain
	reqStatus2 := httptest.NewRequest("GET", "/api/status", nil)
	wStatus2 := httptest.NewRecorder()
	handler.GetStatus(wStatus2, reqStatus2)
	var statusResp JSONResponse
	_ = json.Unmarshal(wStatus2.Body.Bytes(), &statusResp)
	statusData, _ := statusResp.Data.(map[string]interface{})
	bucketBStatus, _ := statusData["bucket_b"].(map[string]interface{})
	if bucketBStatus["cdn_domain"] != "assets-dev.dongvoyage.cn" {
		t.Errorf("expected cdn_domain = assets-dev.dongvoyage.cn, got %v", bucketBStatus["cdn_domain"])
	}

	// 5. Test /api/file-acl parameter validation
	reqAclEmpty := httptest.NewRequest("GET", "/api/file-acl", nil)
	wAclEmpty := httptest.NewRecorder()
	handler.GetFileACL(wAclEmpty, reqAclEmpty)
	var aclResp JSONResponse
	_ = json.Unmarshal(wAclEmpty.Body.Bytes(), &aclResp)
	if aclResp.Success {
		t.Errorf("expected GetFileACL without key to fail")
	}

	// 6. Test /api/file-urls
	reqURLsEmpty := httptest.NewRequest("GET", "/api/file-urls", nil)
	wURLsEmpty := httptest.NewRecorder()
	handler.GetFileURLs(wURLsEmpty, reqURLsEmpty)
	if wURLsEmpty.Code != http.StatusBadRequest {
		t.Errorf("GetFileURLs without key code = %d, want 400", wURLsEmpty.Code)
	}

	reqURLsValid := httptest.NewRequest("GET", "/api/file-urls?key=static/demo.png&signed=1&expires=7200", nil)
	wURLsValid := httptest.NewRecorder()
	handler.GetFileURLs(wURLsValid, reqURLsValid)
	if wURLsValid.Code != http.StatusOK {
		t.Errorf("GetFileURLs code = %d, want 200", wURLsValid.Code)
	}
	var urlsResp JSONResponse
	_ = json.Unmarshal(wURLsValid.Body.Bytes(), &urlsResp)
	if !urlsResp.Success {
		t.Errorf("expected GetFileURLs success = true")
	}

	// Cleanup
	_ = os.Remove(tempCfg)
}
