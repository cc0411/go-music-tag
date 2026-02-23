package models

import (
	"fmt"
	"time"
)

type Music struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	FilePath    string     `gorm:"uniqueIndex;size:500;not null" json:"file_path"`
	FileName    string     `gorm:"size:255;not null" json:"file_name"`
	FileSize    int64      `gorm:"not null" json:"file_size"`
	Title       string     `gorm:"size:255" json:"title"`
	Artist      string     `gorm:"size:255" json:"artist"`
	Album       string     `gorm:"size:255" json:"album"`
	AlbumArtist string     `gorm:"size:255;column:album_artist" json:"album_artist"`
	Composer    string     `gorm:"size:255" json:"composer"`
	Genre       string     `gorm:"size:100" json:"genre"`
	Year        int        `gorm:"default:0" json:"year"`
	TrackNumber int        `gorm:"column:track_number;default:0" json:"track_number"`
	DiscNumber  int        `gorm:"column:disc_number;default:0" json:"disc_number"`
	Duration    int        `gorm:"default:0" json:"duration"`
	BitRate     int        `gorm:"column:bit_rate;default:0" json:"bit_rate"`
	SampleRate  int        `gorm:"column:sample_rate;default:0" json:"sample_rate"`
	Format      string     `gorm:"size:50" json:"format"`
	HasLyrics   bool       `gorm:"column:has_lyrics;default:false" json:"has_lyrics"`
	HasCover    bool       `gorm:"column:has_cover;default:false" json:"has_cover"`
	CoverMIME   string     `gorm:"column:cover_mime;size:50" json:"cover_mime"`
	Comment     string     `gorm:"size:500" json:"comment"`
	ScanStatus  string     `gorm:"size:20;default:pending" json:"scan_status"`
	ScanError   string     `gorm:"size:500" json:"scan_error"`
	ScannedAt   *time.Time `json:"scanned_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Music) TableName() string {
	return "music"
}

type MusicResponse struct {
	ID          uint       `json:"id"`
	FilePath    string     `json:"file_path"`
	FileName    string     `json:"file_name"`
	FileSize    int64      `json:"file_size"`
	FileSizeStr string     `json:"file_size_str"`
	Title       string     `json:"title"`
	Artist      string     `json:"artist"`
	Album       string     `json:"album"`
	AlbumArtist string     `json:"album_artist"`
	Composer    string     `json:"composer"`
	Genre       string     `json:"genre"`
	Year        int        `json:"year"`
	TrackNumber int        `json:"track_number"`
	DiscNumber  int        `json:"disc_number"`
	Duration    int        `json:"duration"`
	DurationStr string     `json:"duration_str"`
	BitRate     int        `json:"bit_rate"`
	SampleRate  int        `json:"sample_rate"`
	Format      string     `json:"format"`
	HasCover    bool       `json:"has_cover"`
	Comment     string     `json:"comment"`
	ScanStatus  string     `json:"scan_status"`
	ScanError   string     `json:"scan_error"`
	ScannedAt   *time.Time `json:"scanned_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (m *Music) ToResponse() MusicResponse {
	return MusicResponse{
		ID:          m.ID,
		FilePath:    m.FilePath,
		FileName:    m.FileName,
		FileSize:    m.FileSize,
		FileSizeStr: formatFileSize(m.FileSize),
		Title:       m.Title,
		Artist:      m.Artist,
		Album:       m.Album,
		AlbumArtist: m.AlbumArtist,
		Composer:    m.Composer,
		Genre:       m.Genre,
		Year:        m.Year,
		TrackNumber: m.TrackNumber,
		DiscNumber:  m.DiscNumber,
		Duration:    m.Duration,
		DurationStr: formatDuration(m.Duration),
		BitRate:     m.BitRate,
		SampleRate:  m.SampleRate,
		Format:      m.Format,
		HasCover:    m.HasCover,
		Comment:     m.Comment,
		ScanStatus:  m.ScanStatus,
		ScanError:   m.ScanError,
		ScannedAt:   m.ScannedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

type ScanLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"size:64;index" json:"task_id"`
	Message   string    `gorm:"size:1000" json:"message"`
	Level     string    `gorm:"size:20;default:info" json:"level"`
	CreatedAt time.Time `json:"created_at"`
}

func (ScanLog) TableName() string {
	return "scan_logs"
}

type WebDAVConfig struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	URL        string     `gorm:"size:500;not null" json:"url"`
	Username   string     `gorm:"size:255" json:"username"`
	Password   string     `gorm:"size:255" json:"password"`
	RootPath   string     `gorm:"size:500;default:/" json:"root_path"`
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastTest   *time.Time `json:"last_test"`
	TestStatus string     `gorm:"size:50" json:"test_status"`
	TestError  string     `gorm:"size:500" json:"test_error"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (WebDAVConfig) TableName() string {
	return "webdav_config"
}

type WebDAVConfigResponse struct {
	ID         uint       `json:"id"`
	URL        string     `json:"url"`
	Username   string     `json:"username"`
	Password   string     `json:"password"`
	RootPath   string     `json:"root_path"`
	Enabled    bool       `json:"enabled"`
	LastTest   *time.Time `json:"last_test"`
	TestStatus string     `json:"test_status"`
	TestError  string     `json:"test_error"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (c *WebDAVConfig) ToResponse() WebDAVConfigResponse {
	return WebDAVConfigResponse{
		ID:         c.ID,
		URL:        c.URL,
		Username:   c.Username,
		Password:   c.Password,
		RootPath:   c.RootPath,
		Enabled:    c.Enabled,
		LastTest:   c.LastTest,
		TestStatus: c.TestStatus,
		TestError:  c.TestError,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func formatDuration(seconds int) string {
	mins := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}
