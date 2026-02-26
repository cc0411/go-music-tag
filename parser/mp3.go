package parser

import (
	"bytes"
	"go-music-tag/models"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/tcolgate/mp3" // ✅ 新增：用于精确计算时长
)

type MP3Parser struct{}

func NewMP3Parser() *MP3Parser {
	return &MP3Parser{}
}

func (p *MP3Parser) Parse(data []byte, filePath string, fileName string, fileSize int64) (*models.Music, error) {
	reader := bytes.NewReader(data)

	// 1. 尝试读取 ID3 标签
	md, err := tag.ReadFrom(reader)

	now := time.Now()
	music := &models.Music{
		FilePath:   filePath,
		FileName:   fileName,
		FileSize:   fileSize,
		ScanStatus: "success",
		ScannedAt:  &now,
		Format:     "MP3",
	}

	// 2. 提取基本标签
	if err == nil && md != nil {
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

		// 提取封面
		if artwork := md.Picture(); artwork != nil {
			music.HasCover = true
			music.CoverMIME = artwork.MIMEType
		}

		// 尝试从标签获取时长 (很多标签库不支持，通常为 0)
		// dhowden/tag 不直接支持 Duration，所以这里通常是 0
	}

	// 3. ✅ 关键修复：使用 tcolgate/mp3 精确计算时长和比特率
	// 重新创建一个 reader，因为上面的 ReadFrom 可能已经读到了文件末尾
	mp3Reader := bytes.NewReader(data)
	frameReader, err := mp3.NewFrameReader(mp3Reader)
	if err != nil {
		// 如果无法解析帧，降级使用估算
		music.BitRate = p.estimateBitRate(fileSize)
		music.Duration = p.estimateDuration(fileSize, music.BitRate)
	} else {
		frames, err := frameReader.ReadFrames()
		if err == nil && len(frames) > 0 {
			var totalDuration float64
			var totalBits int64

			for _, frame := range frames {
				totalDuration += frame.Duration().Seconds()
				totalBits += int64(frame.Size()) * 8
			}

			music.Duration = int(totalDuration)
			if totalDuration > 0 {
				// 计算平均比特率 (kbps)
				music.BitRate = int((totalBits / totalDuration) / 1000)
			}

			// 获取采样率 (取第一帧的即可，通常整首歌不变)
			music.SampleRate = int(frames[0].SampleRate())
		} else {
			// 降级处理
			music.BitRate = p.estimateBitRate(fileSize)
			music.Duration = p.estimateDuration(fileSize, music.BitRate)
		}
	}

	// 4. 如果标题仍为空，尝试从文件名解析
	if music.Title == "" {
		p.parseFromFileName(fileName, music)
	}

	// 5. 标记状态
	if music.Duration <= 0 {
		// 如果算出来还是 0，标记一下（可选）
		// music.ScanStatus = "warning"
	}

	return music, nil
}

// 下面的函数保持不变...

func (p *MP3Parser) parseFromFileNameOnly(filePath string, fileName string, fileSize int64) (*models.Music, error) {
	now := time.Now()
	music := &models.Music{
		FilePath:   filePath,
		FileName:   fileName,
		FileSize:   fileSize,
		ScanStatus: "success",
		ScannedAt:  &now,
		Format:     "MP3",
	}

	// 同样尝试计算时长
	mp3Reader := bytes.NewReader(nil) // 这里没有数据，只能估算
	// 由于没有 data，只能估算
	music.BitRate = p.estimateBitRate(fileSize)
	music.Duration = p.estimateDuration(fileSize, music.BitRate)
	music.SampleRate = 44100

	p.parseFromFileName(fileName, music)
	return music, nil
}

func (p *MP3Parser) parseFromFileName(fileName string, music *Music) {
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
