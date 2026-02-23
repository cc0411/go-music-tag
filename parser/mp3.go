package parser

import (
	"bytes"
	"go-music-tag/models"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type MP3Parser struct{}

func NewMP3Parser() *MP3Parser {
	return &MP3Parser{}
}

func (p *MP3Parser) Parse(data []byte, filePath string, fileName string, fileSize int64) (*models.Music, error) {
	reader := bytes.NewReader(data)

	md, err := tag.ReadFrom(reader)
	if err != nil {
		return p.parseFromFileNameOnly(filePath, fileName, fileSize)
	}

	now := time.Now()
	music := &models.Music{
		FilePath:   filePath,
		FileName:   fileName,
		FileSize:   fileSize,
		ScanStatus: "success",
		ScannedAt:  &now,
		Format:     "MP3",
	}

	if title := md.Title(); title != "" {
		music.Title = title
	}
	if artist := md.Artist(); artist != "" {
		music.Artist = artist
	}
	if album := md.Album(); album != "" {
		music.Album = album
	}
	if albumArtist := md.AlbumArtist(); albumArtist != "" {
		music.AlbumArtist = albumArtist
	}
	if composer := md.Composer(); composer != "" {
		music.Composer = composer
	}
	if genre := md.Genre(); genre != "" {
		music.Genre = genre
	}
	if year := md.Year(); year != 0 {
		music.Year = year
	}
	if track, _ := md.Track(); track != 0 {
		music.TrackNumber = track
	}
	if disc, _ := md.Disc(); disc != 0 {
		music.DiscNumber = disc
	}
	if comment := md.Comment(); comment != "" {
		music.Comment = comment
	}

	music.BitRate = p.estimateBitRate(fileSize)
	music.SampleRate = 44100
	music.Duration = p.estimateDuration(fileSize, music.BitRate)

	if artwork := md.Picture(); artwork != nil {
		music.HasCover = true
		music.CoverMIME = artwork.MIMEType
	}

	if music.Title == "" {
		p.parseFromFileName(fileName, music)
	}

	return music, nil
}

func (p *MP3Parser) parseFromFileNameOnly(filePath string, fileName string, fileSize int64) (*models.Music, error) {
	now := time.Now()
	music := &models.Music{
		FilePath:   filePath,
		FileName:   fileName,
		FileSize:   fileSize,
		ScanStatus: "success",
		ScannedAt:  &now,
		Format:     "MP3",
		BitRate:    p.estimateBitRate(fileSize),
		SampleRate: 44100,
		Duration:   p.estimateDuration(fileSize, p.estimateBitRate(fileSize)),
	}

	p.parseFromFileName(fileName, music)
	return music, nil
}

func (p *MP3Parser) parseFromFileName(fileName string, music *models.Music) {
	name := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	name = regexp.MustCompile(`^\d+[\.\-\s]+`).ReplaceAllString(name, "")

	if parts := strings.SplitN(name, " - ", 2); len(parts) == 2 {
		if music.Artist == "" {
			music.Artist = strings.TrimSpace(parts[0])
		}
		if music.Title == "" {
			music.Title = strings.TrimSpace(parts[1])
		}
	} else if music.Title == "" {
		music.Title = name
	}
}

func (p *MP3Parser) estimateBitRate(fileSize int64) int {
	switch {
	case fileSize < 3*1024*1024:
		return 128
	case fileSize < 5*1024*1024:
		return 192
	case fileSize < 8*1024*1024:
		return 256
	default:
		return 320
	}
}

func (p *MP3Parser) estimateDuration(fileSize int64, bitRate int) int {
	if bitRate <= 0 {
		return 0
	}
	return int((fileSize * 8) / (int64(bitRate) * 1000))
}
