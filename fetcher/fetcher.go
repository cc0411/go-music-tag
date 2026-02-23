package fetcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Fetcher 资源获取器
type Fetcher struct {
	client    *http.Client
	lyricsDir string
	coversDir string
}

// LyricResult 歌词结果
type LyricResult struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

// CoverResult 封面结果
type CoverResult struct {
	URL    string `json:"url"`
	Data   []byte `json:"data"`
	Source string `json:"source"`
}

// NewFetcher 创建获取器实例 (注意：N 必须大写！)
func NewFetcher(lyricsDir, coversDir string) *Fetcher {
	// ✅ 创建时确保目录存在
	os.MkdirAll(lyricsDir, 0755)
	os.MkdirAll(coversDir, 0755)
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		lyricsDir: lyricsDir,
		coversDir: coversDir,
	}
}

// SearchLyrics 搜索歌词
func (f *Fetcher) SearchLyrics(artist, title string) (*LyricResult, error) {
	result, err := f.searchNeteaseLyrics(artist, title)
	if err == nil && result != nil {
		return result, nil
	}
	return nil, fmt.Errorf("lyrics not found")
}

func (f *Fetcher) searchNeteaseLyrics(artist, title string) (*LyricResult, error) {
	searchURL := fmt.Sprintf("https://music.163.com/api/search/get?type=1&s=%s",
		url.QueryEscape(title+" "+artist))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Songs []struct {
				ID int `json:"id"`
			} `json:"songs"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Result.Songs) == 0 {
		return nil, fmt.Errorf("no songs found")
	}

	songID := result.Result.Songs[0].ID
	lyricURL := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%d&lv=1", songID)

	req, err = http.NewRequest("GET", lyricURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err = f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lyricResult struct {
		Lrc struct {
			Lyric string `json:"lyric"`
		} `json:"lrc"`
	}

	if err := json.Unmarshal(body, &lyricResult); err != nil {
		return nil, err
	}

	if lyricResult.Lrc.Lyric == "" {
		return nil, fmt.Errorf("no lyrics")
	}

	return &LyricResult{
		Content: lyricResult.Lrc.Lyric,
		Source:  "netease",
	}, nil
}

// SearchCover 搜索封面
func (f *Fetcher) SearchCover(artist, album string) (*CoverResult, error) {
	result, err := f.searchNeteaseCover(artist, album)
	if err == nil && result != nil {
		return result, nil
	}
	return nil, fmt.Errorf("cover not found")
}

func (f *Fetcher) searchNeteaseCover(artist, album string) (*CoverResult, error) {
	searchURL := fmt.Sprintf("https://music.163.com/api/search/get?type=1&s=%s",
		url.QueryEscape(album+" "+artist))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Songs []struct {
				Album struct {
					PicURL string `json:"picUrl"`
				} `json:"album"`
			} `json:"songs"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Result.Songs) == 0 {
		return nil, fmt.Errorf("no songs found")
	}

	picURL := result.Result.Songs[0].Album.PicURL
	if picURL == "" {
		return nil, fmt.Errorf("no cover")
	}

	picURL = strings.Replace(picURL, "http://", "https://", 1)

	coverData, err := f.downloadImage(picURL)
	if err != nil {
		return nil, err
	}

	return &CoverResult{
		URL:    picURL,
		Data:   coverData,
		Source: "netease",
	}, nil
}

func (f *Fetcher) downloadImage(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// SaveLyrics 保存歌词到本地
func (f *Fetcher) SaveLyrics(artist, title, content string) (string, error) {
	filename := f.generateFilename(artist, title) + ".lrc"
	filepath := filepath.Join(f.lyricsDir, filename)

	// ✅ 关键：如果目录不存在则创建
	if err := os.MkdirAll(f.lyricsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create lyrics directory: %w", err)
	}

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filepath, nil
}

// SaveCover 保存封面到本地
func (f *Fetcher) SaveCover(artist, album string, data []byte) (string, error) {
	filename := f.generateFilename(artist, album) + ".jpg"
	filepath := filepath.Join(f.coversDir, filename)

	// ✅ 关键：每次保存前都确保目录存在
	// 即使 NewFetcher 时创建过，这里再保险一次
	if err := os.MkdirAll(f.coversDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create covers directory: %w", err)
	}

	// 调试日志：打印路径
	fmt.Printf("Saving cover to: %s\n", filepath)

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write cover file: %w", err)
	}

	return filepath, nil
}

func (f *Fetcher) generateFilename(artist, title string) string {
	raw := artist + " - " + title
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// GetLocalLyricsPath 获取本地歌词路径
func (f *Fetcher) GetLocalLyricsPath(artist, title string) string {
	filename := f.generateFilename(artist, title) + ".lrc"
	return filepath.Join(f.lyricsDir, filename)
}

// GetLocalCoverPath 获取本地封面路径
func (f *Fetcher) GetLocalCoverPath(artist, album string) string {
	filename := f.generateFilename(artist, album) + ".jpg"
	return filepath.Join(f.coversDir, filename)
}

// FetchAndSave 获取并保存
func (f *Fetcher) FetchAndSave(artist, title, album string) (lyricsPath, coverPath string, err error) {
	if artist != "" && title != "" {
		lyric, err := f.SearchLyrics(artist, title)
		if err == nil && lyric.Content != "" {
			lyricsPath, err = f.SaveLyrics(artist, title, lyric.Content)
			if err != nil {
				lyricsPath = ""
			}
		}
	}

	if artist != "" || album != "" {
		cover, err := f.SearchCover(artist, album)
		if err == nil && cover.Data != nil {
			coverPath, err = f.SaveCover(artist, album, cover.Data)
			if err != nil {
				coverPath = ""
			}
		}
	}

	return lyricsPath, coverPath, nil
}
