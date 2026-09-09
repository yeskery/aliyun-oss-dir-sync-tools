package ossclient

import (
	"fmt"
	"io"
	"strings"
	"time"

	"aliyun_oss_dir_sync-tools/internal/config"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSClient 封装阿里云 OSS SDK 客户端
type OSSClient struct {
	cfg    config.BucketConfig
	client *oss.Client
	bucket *oss.Bucket
}

// NewClient 创建并初始化 OSS 客户端
func NewClient(cfg config.BucketConfig) (*OSSClient, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	ak := strings.TrimSpace(cfg.AccessKeyID)
	sk := strings.TrimSpace(cfg.AccessKeySecret)
	bucketName := strings.TrimSpace(cfg.BucketName)

	if endpoint == "" || ak == "" || sk == "" || bucketName == "" {
		return nil, fmt.Errorf("Bucket [%s] 配置不完整，必须填写 Endpoint、AccessKeyID、AccessKeySecret 和 BucketName", cfg.Name)
	}

	// 自动修正 endpoint 格式（若携带 http:// 或 https:// 则去除，官方 SDK endpoint 建议不带前缀）
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimRight(endpoint, "/")

	var clientOptions []oss.ClientOption
	if token := strings.TrimSpace(cfg.SecurityToken); token != "" {
		clientOptions = append(clientOptions, oss.SecurityToken(token))
	}

	client, err := oss.New(endpoint, ak, sk, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("初始化 OSS 客户端失败: %w", err)
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("获取 Bucket [%s] 句柄失败: %w", bucketName, err)
	}

	return &OSSClient{
		cfg:    cfg,
		client: client,
		bucket: bucket,
	}, nil
}

// GetConfig 返回 Bucket 配置
func (c *OSSClient) GetConfig() config.BucketConfig {
	return c.cfg
}

// GetBucket 返回底层 *oss.Bucket
func (c *OSSClient) GetBucket() *oss.Bucket {
	return c.bucket
}

// TestConnection 测试当前 Bucket 凭据与网络连通性
func (c *OSSClient) TestConnection() error {
	// 使用 ListObjectsV2 请求 1 个对象以最小开销验证权限
	_, err := c.bucket.ListObjectsV2(oss.MaxKeys(1))
	if err != nil {
		return fmt.Errorf("连接 Bucket [%s] 失败: %w", c.cfg.BucketName, err)
	}
	return nil
}

// ListObjectsUnderPrefix 递归列出指定目录前缀下的所有对象
func (c *OSSClient) ListObjectsUnderPrefix(prefix string) (map[string]ObjectMeta, error) {
	objects := make(map[string]ObjectMeta)
	continuationToken := ""

	for {
		var opts []oss.Option
		opts = append(opts, oss.MaxKeys(1000))
		if prefix != "" {
			opts = append(opts, oss.Prefix(prefix))
		}
		if continuationToken != "" {
			opts = append(opts, oss.ContinuationToken(continuationToken))
		}

		lor, err := c.bucket.ListObjectsV2(opts...)
		if err != nil {
			return nil, fmt.Errorf("列出目录 [%s] 下文件失败: %w", prefix, err)
		}

		for _, item := range lor.Objects {
			// 忽略以 / 结尾的纯目录标记伪对象
			if strings.HasSuffix(item.Key, "/") && item.Size == 0 {
				continue
			}

			objects[item.Key] = ObjectMeta{
				Key:          item.Key,
				ETag:         strings.Trim(item.ETag, "\""),
				LastModified: item.LastModified,
				Size:         item.Size,
			}
		}

		if !lor.IsTruncated {
			break
		}
		continuationToken = lor.NextContinuationToken
	}

	return objects, nil
}

// CopyFromBucket 将另一个 Bucket 的指定对象同步到当前 Bucket
// 优先采用服务端内网 CopyObjectFrom；如遇跨账号无权限等错误，自动降级为流式直通复制（零临时文件）
func (c *OSSClient) CopyFromBucket(srcClient *OSSClient, key string) error {
	// 尝试 1: 服务端内网拷贝
	_, err := c.bucket.CopyObjectFrom(srcClient.cfg.BucketName, key, key)
	if err == nil {
		return nil
	}

	// 若服务端 Copy 失败（例如跨主账号授权不足），使用流式直通降级传输
	rc, getErr := srcClient.bucket.GetObject(key)
	if getErr != nil {
		return fmt.Errorf("读取源文件失败 [src=%s, key=%s]: %w (原 Copy 错误: %v)", srcClient.cfg.BucketName, key, getErr, err)
	}
	defer rc.Close()

	putErr := c.bucket.PutObject(key, rc)
	if putErr != nil {
		return fmt.Errorf("流式写入目标 Bucket 失败 [dest=%s, key=%s]: %w", c.cfg.BucketName, key, putErr)
	}

	return nil
}

// DeleteObject 删除当前 Bucket 中的指定对象
func (c *OSSClient) DeleteObject(key string) error {
	err := c.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("删除文件失败 [bucket=%s, key=%s]: %w", c.cfg.BucketName, key, err)
	}
	return nil
}

// DeletePrefix 递归删除指定前缀目录下的所有对象（含子文件与目录标记本身）
func (c *OSSClient) DeletePrefix(prefix string) (int, error) {
	cleanPrefix := prefix
	if cleanPrefix != "" && !strings.HasSuffix(cleanPrefix, "/") {
		cleanPrefix += "/"
	}
	continuationToken := ""
	totalDeleted := 0

	for {
		var opts []oss.Option
		opts = append(opts, oss.MaxKeys(1000))
		if cleanPrefix != "" {
			opts = append(opts, oss.Prefix(cleanPrefix))
		}
		if continuationToken != "" {
			opts = append(opts, oss.ContinuationToken(continuationToken))
		}

		lor, err := c.bucket.ListObjectsV2(opts...)
		if err != nil {
			return totalDeleted, fmt.Errorf("列出待删除前缀 [%s] 失败: %w", cleanPrefix, err)
		}

		var keysToDelete []string
		for _, item := range lor.Objects {
			keysToDelete = append(keysToDelete, item.Key)
		}

		if len(keysToDelete) > 0 {
			delRes, delErr := c.bucket.DeleteObjects(keysToDelete)
			if delErr != nil {
				return totalDeleted, fmt.Errorf("批量删除对象失败: %w", delErr)
			}
			totalDeleted += len(delRes.DeletedObjects)
		}

		if !lor.IsTruncated {
			break
		}
		continuationToken = lor.NextContinuationToken
	}

	// 确保目录占位标记本身也被删除
	_ = c.bucket.DeleteObject(cleanPrefix)

	return totalDeleted, nil
}

// GetObjectCRC64 获取对象的 CRC64 校验值（通过轻量级的 HEAD 请求）
func (c *OSSClient) GetObjectCRC64(key string) (string, error) {
	hdr, err := c.bucket.GetObjectMeta(key)
	if err != nil {
		return "", err
	}
	return hdr.Get("X-Oss-Hash-Crc64ecma"), nil
}

// ListDirectory 列出指定单层目录下的直接子目录 (CommonPrefixes) 和直接文件 (Objects)
func (c *OSSClient) ListDirectory(dir string) (subDirs []string, files []ObjectMeta, err error) {
	prefix := dir
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	continuationToken := ""
	dirSet := make(map[string]bool)

	for {
		var opts []oss.Option
		opts = append(opts, oss.MaxKeys(1000), oss.Delimiter("/"))
		if prefix != "" {
			opts = append(opts, oss.Prefix(prefix))
		}
		if continuationToken != "" {
			opts = append(opts, oss.ContinuationToken(continuationToken))
		}

		lor, err := c.bucket.ListObjectsV2(opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("列出目录 [%s] 失败: %w", prefix, err)
		}

		// 收集子目录
		for _, cp := range lor.CommonPrefixes {
			if !dirSet[cp] {
				dirSet[cp] = true
				subDirs = append(subDirs, cp)
			}
		}

		// 收集文件
		for _, item := range lor.Objects {
			// 排除当前目录自身的占位 marker
			if item.Key == prefix || (strings.HasSuffix(item.Key, "/") && item.Size == 0) {
				continue
			}
			files = append(files, ObjectMeta{
				Key:          item.Key,
				ETag:         strings.Trim(item.ETag, "\""),
				LastModified: item.LastModified,
				Size:         item.Size,
			})
		}

		if !lor.IsTruncated {
			break
		}
		continuationToken = lor.NextContinuationToken
	}

	return subDirs, files, nil
}

// PutObject 上传文件流至当前 Bucket
func (c *OSSClient) PutObject(key string, reader io.Reader) error {
	err := c.bucket.PutObject(key, reader)
	if err != nil {
		return fmt.Errorf("上传至 Bucket [%s] 失败 [%s]: %w", c.cfg.BucketName, key, err)
	}
	return nil
}

// GetObject 获取文件流读取句柄
func (c *OSSClient) GetObject(key string) (io.ReadCloser, error) {
	return c.bucket.GetObject(key)
}

// GetSignURL 生成带有效期的只读访问链接（支持自定义有效期秒数，默认 3600 秒）
func (c *OSSClient) GetSignURL(key string, expiredInSec int64) (string, error) {
	if c == nil || c.bucket == nil {
		return "", fmt.Errorf("OSS 客户端未就绪")
	}
	if expiredInSec <= 0 {
		expiredInSec = 3600
	}
	return c.bucket.SignURL(key, oss.HTTPGet, expiredInSec)
}

// GetObjectMeta 获取指定 Key 的最新元数据
func (c *OSSClient) GetObjectMeta(key string) (*ObjectMeta, error) {
	props, err := c.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return nil, err
	}

	etag := strings.Trim(props.Get("ETag"), "\"")
	lastModStr := props.Get("Last-Modified")
	lastMod, _ := time.Parse(time.RFC1123, lastModStr)

	return &ObjectMeta{
		Key:          key,
		ETag:         etag,
		LastModified: lastMod,
	}, nil
}

// SetObjectACL 设置对象的读写权限 ACL (default, private, public-read, public-read-write)
func (c *OSSClient) SetObjectACL(key string, acl oss.ACLType) error {
	if c.bucket == nil {
		return fmt.Errorf("OSS Bucket 未初始化")
	}
	err := c.bucket.SetObjectACL(key, acl)
	if err != nil {
		return fmt.Errorf("设置对象 ACL 失败 [bucket=%s, key=%s]: %w", c.cfg.BucketName, key, err)
	}
	return nil
}

// GetObjectACL 获取对象的读写权限 ACL
func (c *OSSClient) GetObjectACL(key string) (string, error) {
	if c.bucket == nil {
		return "", fmt.Errorf("OSS Bucket 未初始化")
	}
	res, err := c.bucket.GetObjectACL(key)
	if err != nil {
		return "", fmt.Errorf("获取对象 ACL 失败 [bucket=%s, key=%s]: %w", c.cfg.BucketName, key, err)
	}
	return res.ACL, nil
}

