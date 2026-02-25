package webdav

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"go-music-tag/config"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

type Client struct {
	listClient *gowebdav.Client
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	rootPath   string
}

type FileInfo struct {
	Path  string
	Name  string
	Size  int64
	IsDir bool
}

func (c *Client) ReadDirAll(path string) ([]os.FileInfo, error) {
	return c.listClient.ReadDir(path)
}

func NewClientWithConfig(urlStr, username, password, rootPath string) (*Client, error) {
	// 核心：创建支持重定向的 HTTP 客户端
	httpClient := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil // ✅ 允许重定向
		},
	}

	listClient := gowebdav.NewClient(urlStr, username, password)

	return &Client{
		listClient: listClient,
		httpClient: httpClient,
		baseURL:    urlStr,
		username:   username,
		password:   password,
		rootPath:   rootPath,
	}, nil
}

func NewClientNoCheck(urlStr, username, password, rootPath string) *Client {
	httpClient := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	listClient := gowebdav.NewClient(urlStr, username, password)

	return &Client{
		listClient: listClient,
		httpClient: httpClient,
		baseURL:    urlStr,
		username:   username,
		password:   password,
		rootPath:   rootPath,
	}
}

func (c *Client) GetFile(filePath string) ([]byte, error) {
	return c.getFileViaHTTP(filePath)
}

func (c *Client) getFileViaHTTP(filePath string) ([]byte, error) {
	// 1. 构建 URL (处理路径拼接)
	cleanPath := strings.TrimPrefix(filePath, c.rootPath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	if cleanPath == "" {
		return nil, fmt.Errorf("empty file path")
	}

	parts := strings.Split(cleanPath, "/")
	for i, part := range parts {
		if part != "" {
			parts[i] = url.PathEscape(part)
		}
	}
	encodedPath := strings.Join(parts, "/")

	finalPath := strings.TrimSuffix(c.rootPath, "/") + "/" + encodedPath
	base := strings.TrimSuffix(c.baseURL, "/")
	fullURL := fmt.Sprintf("%s%s", base, finalPath)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	// 2. 【关键修复】设置完整的浏览器级请求头
	auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))

	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	// 【最关键】设置 Referer，假装是从 Alist 首页点进来的
	req.Header.Set("Referer", c.baseURL+"/")
	req.Header.Set("Origin", c.baseURL)

	// 3. 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 如果是 302，说明重定向配置没生效
		if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect {
			location := resp.Header.Get("Location")
			return nil, fmt.Errorf("got redirect but failed to follow: %s", location)
		}
		// 打印响应体内容，看看 Alist 返回了什么错误信息
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %s (URL: %s) Body: %s", resp.Status, fullURL, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetFileSize(filePath string) (int64, error) {
	info, err := c.listClient.Stat(filePath)
	if err == nil {
		return info.Size(), nil
	}
	return c.getFileSizeViaHTTP(filePath)
}

func (c *Client) getFileSizeViaHTTP(filePath string) (int64, error) {
	fullURL, err := c.buildURL(filePath)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("HEAD", fullURL, nil)
	if err != nil {
		return 0, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return resp.ContentLength, nil
}

func (c *Client) buildURL(filePath string) (string, error) {
	cleanPath := strings.TrimPrefix(filePath, "/")
	parts := strings.Split(cleanPath, "/")
	for i, part := range parts {
		if part != "" {
			parts[i] = url.PathEscape(part)
		}
	}
	encodedPath := strings.Join(parts, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.baseURL, "/"), encodedPath), nil
}

func (c *Client) ListMP3Files() ([]FileInfo, error) {
	files, err := c.listClient.ReadDir(c.rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	var mp3Files []FileInfo
	cfg := config.GetConfig()
	for _, file := range files {
		if !file.IsDir() && isMP3File(file.Name(), cfg.Scan.Extensions) {
			mp3Files = append(mp3Files, FileInfo{
				Path:  path.Join(c.rootPath, file.Name()),
				Name:  file.Name(),
				Size:  file.Size(),
				IsDir: file.IsDir(),
			})
		}
	}
	return mp3Files, nil
}

func (c *Client) ListMP3FilesRecursive() ([]FileInfo, error) {
	return c.walkDir(c.rootPath, config.GetConfig().Scan.Extensions)
}

func (c *Client) walkDir(dirPath string, extensions []string) ([]FileInfo, error) {
	files, err := c.listClient.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var mp3Files []FileInfo
	for _, file := range files {
		fullPath := path.Join(dirPath, file.Name())
		if file.IsDir() {
			subFiles, err := c.walkDir(fullPath, extensions)
			if err != nil {
				continue
			}
			mp3Files = append(mp3Files, subFiles...)
		} else if isMP3File(file.Name(), extensions) {
			mp3Files = append(mp3Files, FileInfo{
				Path:  fullPath,
				Name:  file.Name(),
				Size:  file.Size(),
				IsDir: false,
			})
		}
	}
	return mp3Files, nil
}

func isMP3File(filename string, extensions []string) bool {
	lowerName := strings.ToLower(filename)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerName, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
