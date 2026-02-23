package parser

import (
	"encoding/json"
	"fmt"
	"go-music-tag/models"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MusicBrainzClient MusicBrainz API 客户端
type MusicBrainzClient struct {
	client *http.Client
}

// MBSearchResult MusicBrainz 搜索结果
type MBSearchResult struct {
	Recordings []struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		ArtistCredit []struct {
			Name string `json:"name"`
		} `json:"artist-credit"`
		Releases []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Date  string `json:"date"`
		} `json:"releases"`
	} `json:"recordings"`
}

// NewMusicBrainzClient 创建客户端
func NewMusicBrainzClient() *MusicBrainzClient {
	return &MusicBrainzClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SearchTrack 搜索曲目信息
func (mb *MusicBrainzClient) SearchTrack(artist, title string) (*models.Music, error) {
	if artist == "" && title == "" {
		return nil, fmt.Errorf("no search query")
	}

	query := url.QueryEscape(fmt.Sprintf(`artist:"%s" AND recording:"%s"`, artist, title))
	apiURL := fmt.Sprintf("https://musicbrainz.org/ws/2/recording/?query=%s&fmt=json&limit=5", query)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// MusicBrainz 要求 User-Agent
	req.Header.Set("User-Agent", "MusicTagManager/1.0 ( https://github.com/your-repo )")

	resp, err := mb.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result MBSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Recordings) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	rec := result.Recordings[0]
	music := &models.Music{}

	// 使用 MusicBrainz 数据补充标签
	if rec.Title != "" {
		music.Title = rec.Title
	}

	if len(rec.ArtistCredit) > 0 {
		music.Artist = rec.ArtistCredit[0].Name
	}

	if len(rec.Releases) > 0 {
		music.Album = rec.Releases[0].Title
		if rec.Releases[0].Date != "" && len(rec.Releases[0].Date) >= 4 {
			if year, err := strconv.Atoi(rec.Releases[0].Date[:4]); err == nil {
				music.Year = year
			}
		}
	}

	return music, nil
}

// SearchArtist 搜索艺术家信息
func (mb *MusicBrainzClient) SearchArtist(artistName string) (string, error) {
	if artistName == "" {
		return "", fmt.Errorf("no artist name")
	}

	query := url.QueryEscape(artistName)
	apiURL := fmt.Sprintf("https://musicbrainz.org/ws/2/artist/?query=%s&fmt=json&limit=1", query)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "MusicTagManager/1.0")

	resp, err := mb.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Artists []struct {
			Name     string `json:"name"`
			SortName string `json:"sort-name"`
		} `json:"artists"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Artists) > 0 {
		return result.Artists[0].Name, nil
	}

	return "", fmt.Errorf("no artist found")
}

// LookupByMBID 通过 MusicBrainz ID 查询
func (mb *MusicBrainzClient) LookupByMBID(mbid string) (*models.Music, error) {
	if mbid == "" {
		return nil, fmt.Errorf("no MBID provided")
	}

	apiURL := fmt.Sprintf("https://musicbrainz.org/ws/2/recording/%s?fmt=json&inc=releases+artists", mbid)

	resp, err := mb.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		ArtistCredit []struct {
			Name string `json:"name"`
		} `json:"artist-credit"`
		Releases []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Date  string `json:"date"`
		} `json:"releases"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	music := &models.Music{
		Title: result.Title,
	}

	if len(result.ArtistCredit) > 0 {
		music.Artist = result.ArtistCredit[0].Name
	}

	if len(result.Releases) > 0 {
		music.Album = result.Releases[0].Title
		if result.Releases[0].Date != "" && len(result.Releases[0].Date) >= 4 {
			if year, err := strconv.Atoi(result.Releases[0].Date[:4]); err == nil {
				music.Year = year
			}
		}
	}

	return music, nil
}
