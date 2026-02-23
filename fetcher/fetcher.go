package fetcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Fetcher struct {
	client    *http.Client
	lyricsDir string
	coversDir string
}

type LyricResult struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

type CoverResult struct {
	URL    string `json:"url"`
	Data   []byte `json:"data"`
	Source string `json:"source"`
}

// NewFetcher 创建抓取器
// ✅ 修复：强制将相对路径转换为绝对路径 (基于 /app/data)
func NewFetcher(lyricsDir, coversDir string) *Fetcher {
	// 确保使用绝对路径
	if !filepath.IsAbs(lyricsDir) {
		lyricsDir = filepath.Join("/app/data", lyricsDir)
	}
	if !filepath.IsAbs(coversDir) {
		coversDir = filepath.Join("/app/data", coversDir)
	}

	log.Printf("[Fetcher] Initializing with lyricsDir=%s, coversDir=%s", lyricsDir, coversDir)

	// 提前创建目录，如果失败则记录日志但不 panic
	if err := os.MkdirAll(lyricsDir, 0755); err != nil {
		log.Printf("[Fetcher] Warning: failed to create lyrics dir %s: %v", lyricsDir, err)
	}
	if err := os.MkdirAll(coversDir, 0755); err != nil {
		log.Printf("[Fetcher] Warning: failed to create covers dir %s: %v", coversDir, err)
	}

	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		lyricsDir: lyricsDir,
		coversDir: coversDir,
	}
}

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

	log.Printf("[Fetcher] Saving lyrics to: %s", filepath)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		log.Printf("[Fetcher] Error saving lyrics: %v", err)
		return "", err
	}

	return filepath, nil
}

// SaveCover 保存封面到本地
func (f *Fetcher) SaveCover(artist, album string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cover data is empty")
	}

	filename := f.generateFilename(artist, album) + ".jpg"
	filepath := filepath.Join(f.coversDir, filename)

	log.Printf("[Fetcher] Saving cover to: %s (size: %d bytes)", filepath, len(data))

	// 再次确保目录存在
	if err := os.MkdirAll(f.coversDir, 0755); err != nil {
		log.Printf("[Fetcher] Error creating directory: %v", err)
		return "", fmt.Errorf("failed to create covers directory: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		log.Printf("[Fetcher] Error writing file: %v", err)
		return "", fmt.Errorf("failed to write cover file: %w", err)
	}

	// 验证文件是否真的存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		log.Printf("[Fetcher] CRITICAL: File written but does not exist!")
		return "", fmt.Errorf("file system error")
	}

	log.Printf("[Fetcher] Cover saved successfully: %s", filepath)
	return filepath, nil
}

func (f *Fetcher) generateFilename(artist, title string) string {
	raw := artist + " - " + title
	if raw == " - " {
		raw = time.Now().Format("20060102150405") // 防止空文件名
	}
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func (f *Fetcher) GetLocalLyricsPath(artist, title string) string {
	filename := f.generateFilename(artist, title) + ".lrc"
	return filepath.Join(f.lyricsDir, filename)
}

func (f *Fetcher) GetLocalCoverPath(artist, album string) string {
	filename := f.generateFilename(artist, album) + ".jpg"
	return filepath.Join(f.coversDir, filename)
}

// FetchAndSave 获取并保存歌词和封面
// ✅ 修复：严格分离歌词和封面逻辑，各自独立返回结果
func (f *Fetcher) FetchAndSave(artist, title, album string) (lyricsPath, coverPath string, err error) {
	// --- 1. 获取歌词 ---
	if artist != "" && title != "" {
		lyric, lyricErr := f.SearchLyrics(artist, title)
		if lyricErr == nil && lyric.Content != "" {
			path, saveErr := f.SaveLyrics(artist, title, lyric.Content)
			if saveErr != nil {
				log.Printf("[Fetcher] ⚠️ Lyrics fetched but save failed: %v", saveErr)
			} else {
				lyricsPath = path
				log.Printf("[Fetcher] ✅ Lyrics saved: %s", lyricsPath)
			}
		} else {
			log.Printf("[Fetcher] ℹ️ Lyrics not found for: %s - %s", artist, title)
		}
	}

	// --- 2. 获取封面 (独立逻辑) ---
	searchArtist := artist
	if searchArtist == "" {
		searchArtist = "Various Artists"
	}

	// 优先用专辑名，没有则用歌名
	searchAlbum := album
	if searchAlbum == "" && title != "" {
		searchAlbum = title
	}

	if searchArtist != "" || searchAlbum != "" {
		cover, coverErr := f.SearchCover(searchArtist, searchAlbum)

		// ✅ 关键修复：严格检查 cover.Data 是否存在且不为空
		if coverErr == nil && cover != nil && cover.Data != nil && len(cover.Data) > 0 {
			path, saveErr := f.SaveCover(searchArtist, searchAlbum, cover.Data)
			if saveErr != nil {
				log.Printf("[Fetcher] ❌ Cover fetched but save failed: %v", saveErr)
				// 封面保存失败，返回错误，让上层知道
				return lyricsPath, "", fmt.Errorf("cover save failed: %w", saveErr)
			}
			coverPath = path
			log.Printf("[Fetcher] ✅ Cover saved: %s", coverPath)
		} else {
			// 记录为什么没找到封面
			if coverErr != nil {
				log.Printf("[Fetcher] ❌ Cover search failed: %v (Artist: %s, Album: %s)", coverErr, searchArtist, searchAlbum)
			} else if cover == nil {
				log.Printf("[Fetcher] ❌ Cover result is nil")
			} else if len(cover.Data) == 0 {
				log.Printf("[Fetcher] ❌ Cover data is empty (URL: %s)", cover.URL)
			}
		}
	} else {
		log.Printf("[Fetcher] ℹ️ Skip cover search: no artist or album info")
	}

	// --- 3. 返回结果 ---
	// 如果两者都失败，才返回错误
	if lyricsPath == "" && coverPath == "" {
		return "", "", fmt.Errorf("failed to fetch both lyrics and cover")
	}

	// 如果至少有一个成功，返回 nil 错误（但上层需要分别检查路径是否为空）
	return lyricsPath, coverPath, nil
}
