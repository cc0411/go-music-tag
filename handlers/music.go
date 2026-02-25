package handlers

import (
	"bytes"
	"errors" // ✅ 新增
	"fmt"
	"go-music-tag/database"
	"go-music-tag/fetcher"
	"go-music-tag/models"
	"go-music-tag/parser"
	"go-music-tag/webdav"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MusicHandler struct {
	db       *gorm.DB
	parser   *parser.MP3Parser
	dav      *webdav.Client
	davMutex sync.RWMutex
	davReady bool
}

type ScanRequest struct {
	Recursive bool `json:"recursive"`
}

type MusicListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Title    string `form:"title"`
	Artist   string `form:"artist"`
	Album    string `form:"album"`
	Genre    string `form:"genre"`
}

type MusicListResponse struct {
	Total int64                  `json:"total"`
	List  []models.MusicResponse `json:"list"`
	Page  int                    `json:"page"`
	Size  int                    `json:"page_size"`
}

type WebDAVConfigRequest struct {
	URL      string `json:"url" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password"`
	RootPath string `json:"root_path"`
	Enabled  bool   `json:"enabled"`
}

type UpdateMusicRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Composer    string `json:"composer"`
	Genre       string `json:"genre"`
	Year        int    `json:"year"`
	TrackNumber int    `json:"track_number"`
	DiscNumber  int    `json:"disc_number"`
	Comment     string `json:"comment"`
}

type BatchUpdateRequest struct {
	IDs    []uint `json:"ids"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Genre  string `json:"genre"`
	Year   int    `json:"year"`
}

// BatchStatus 批量操作状态

type BatchStatus struct {
	Running   bool      `json:"running"`
	TaskType  string    `json:"task_type"`
	Total     int       `json:"total"`
	Current   int       `json:"current"`
	Success   int       `json:"success"`
	Failed    int       `json:"failed"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"` // 可选
}

var (
	batchStatus = &BatchStatus{Running: false}
	statusMutex sync.Mutex
)
var (
	scanTaskID = ""
	scanMutex  sync.Mutex
)

// ✅ 自定义状态码
const StatusBusy = 409

// ✅ 关键修复：添加 getDB 方法
func (h *MusicHandler) getDB() *gorm.DB {
	if h.db == nil {
		h.db = database.GetDB()
	}
	return h.db
}
func NewMusicHandler() (*MusicHandler, error) {
	return &MusicHandler{
		db:       database.GetDB(),
		parser:   parser.NewMP3Parser(),
		dav:      nil,
		davReady: false,
	}, nil
}

func NewMusicHandlerLazy() *MusicHandler {
	return &MusicHandler{
		db:       database.GetDB(),
		parser:   parser.NewMP3Parser(),
		dav:      nil,
		davReady: false,
	}
}

func (h *MusicHandler) getWebDAVClient() (*webdav.Client, error) {
	h.davMutex.RLock()
	if h.davReady && h.dav != nil {
		client := h.dav
		h.davMutex.RUnlock()
		return client, nil
	}
	h.davMutex.RUnlock()

	h.davMutex.Lock()
	defer h.davMutex.Unlock()

	if h.davReady && h.dav != nil {
		return h.dav, nil
	}

	var dbConfig models.WebDAVConfig
	result := h.db.First(&dbConfig)
	if result.Error != nil {
		return nil, fmt.Errorf("no WebDAV config found in database")
	}

	if !dbConfig.Enabled {
		return nil, fmt.Errorf("WebDAV is disabled")
	}

	client := webdav.NewClientNoCheck(dbConfig.URL, dbConfig.Username, dbConfig.Password, dbConfig.RootPath)

	h.dav = client
	h.davReady = true
	return h.dav, nil
}

func (h *MusicHandler) resetWebDAVClient() {
	h.davMutex.Lock()
	h.dav = nil
	h.davReady = false
	h.davMutex.Unlock()
}

// Scan 扫描音乐库
func (h *MusicHandler) Scan(c *gin.Context) {
	db := h.getDB()

	// ✅ 修复：检查是否有批量任务在运行
	statusMutex.Lock()
	if batchStatus.Running {
		statusMutex.Unlock()
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "Please wait for batch task to complete before scanning",
		})
		return
	}
	statusMutex.Unlock()

	// ✅ 修复：使用 TRUNCATE 或 DELETE 清空旧数据
	db.Exec("DELETE FROM music")
	db.Exec("DELETE FROM scan_logs")

	var cfg models.WebDAVConfig
	if err := db.First(&cfg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "WebDAV not configured"})
		return
	}

	client, err := webdav.NewClientWithConfig(cfg.URL, cfg.Username, cfg.Password, cfg.RootPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	files, err := client.ListMP3FilesRecursive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	taskID := time.Now().Format("20060102150405")
	total := len(files)
	success := 0
	failed := 0

	go func() {
		for i, file := range files {
			log.Printf("Processing [%d/%d]: %s", i+1, total, file.Name)
			h.logScan(taskID, fmt.Sprintf("Processing [%d/%d]: %s", i+1, total, file.Name), "info")

			data, err := client.GetFile(file.Path)
			if err != nil {
				log.Printf("Failed to get file %s: %v", file.Name, err)
				h.logScan(taskID, fmt.Sprintf("Failed to get file %s: %v", file.Name, err), "error")
				failed++
				continue
			}

			metadata, err := tag.ReadFrom(strings.NewReader(string(data)))

			scanTime := time.Now()
			music := models.Music{
				FilePath:   file.Path,
				FileName:   file.Name,
				FileSize:   file.Size,
				ScanStatus: "success",
				ScannedAt:  &scanTime,
			}

			if err == nil && metadata != nil {
				music.Title = metadata.Title()
				music.Artist = metadata.Artist()
				music.Album = metadata.Album()
				music.Year = metadata.Year()
				music.Genre = metadata.Genre()
				music.Duration = 0

				picture := metadata.Picture()
				if picture != nil && len(picture.Data) > 0 {
					music.HasCover = true
					music.CoverMIME = "image/jpeg"
				}
			} else {
				music.Title = file.Name
			}

			// ✅ 修复：使用 FirstOrCreate 避免唯一约束冲突
			var existing models.Music
			result := db.Where("file_path = ?", file.Path).First(&existing)
			if result.Error == nil {
				// 文件已存在，更新
				db.Model(&existing).Updates(music)
			} else {
				// 新文件，插入
				db.Create(&music)
			}

			if result.Error == nil || errors.Is(result.Error, gorm.ErrRecordNotFound) {
				success++
			} else {
				log.Printf("Failed to save %s: %v", file.Name, result.Error)
				h.logScan(taskID, fmt.Sprintf("Failed to save %s: %v", file.Name, result.Error), "error")
				failed++
			}
		}
		h.logScan(taskID, fmt.Sprintf("Scan completed. Total: %d, Success: %d, Failed: %d", total, success, failed), "info")
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Scan started", "task_id": taskID})
}

func (h *MusicHandler) runScan(taskID string, recursive bool, client *webdav.Client) {
	defer func() {
		scanMutex.Lock()
		scanTaskID = ""
		scanMutex.Unlock()
	}()

	h.logScan(taskID, "Scan started", "info")

	var files []webdav.FileInfo
	var err error

	if recursive {
		files, err = client.ListMP3FilesRecursive()
	} else {
		files, err = client.ListMP3Files()
	}

	if err != nil {
		h.logScan(taskID, fmt.Sprintf("Failed to list files: %v", err), "error")
		return
	}

	total := len(files)
	success := 0
	failed := 0

	h.logScan(taskID, fmt.Sprintf("Found %d MP3 files", total), "info")

	for i, file := range files {
		h.logScan(taskID, fmt.Sprintf("Processing [%d/%d]: %s", i+1, total, file.Name), "info")

		data, err := client.GetFile(file.Path)
		if err != nil {
			failed++
			h.logScan(taskID, fmt.Sprintf("Failed to get file %s: %v", file.Name, err), "error")
			h.saveFailedMusic(file, err.Error())
			continue
		}

		music, err := h.parser.Parse(data, file.Path, file.Name, file.Size)
		if err != nil {
			failed++
			h.logScan(taskID, fmt.Sprintf("Failed to parse %s: %v", file.Name, err), "error")
			h.saveFailedMusic(file, err.Error())
			continue
		}

		if err := h.saveMusic(music); err != nil {
			failed++
			h.logScan(taskID, fmt.Sprintf("Failed to save %s: %v", file.Name, err), "error")
			continue
		}

		success++
		h.logScan(taskID, fmt.Sprintf("Success: %s", file.Name), "info")
	}

	h.logScan(taskID, fmt.Sprintf("Scan completed. Total: %d, Success: %d, Failed: %d", total, success, failed), "info")
}

func (h *MusicHandler) saveMusic(music *models.Music) error {
	return h.db.Where("file_path = ?", music.FilePath).
		FirstOrCreate(music, music).
		Error
}

func (h *MusicHandler) saveFailedMusic(file webdav.FileInfo, errMsg string) {
	now := time.Now()
	music := &models.Music{
		FilePath:   file.Path,
		FileName:   file.Name,
		FileSize:   file.Size,
		ScanStatus: "failed",
		ScanError:  errMsg,
		ScannedAt:  &now,
	}
	h.db.Where("file_path = ?", music.FilePath).
		FirstOrCreate(music, music)
}

func (h *MusicHandler) logScan(taskID, message, level string) {
	log := &models.ScanLog{
		TaskID:  taskID,
		Message: message,
		Level:   level,
	}
	h.db.Create(log)
}

func (h *MusicHandler) GetScanStatus(c *gin.Context) {
	scanMutex.Lock()
	running := scanTaskID != ""
	currentTaskID := scanTaskID
	scanMutex.Unlock()

	var lastLog models.ScanLog
	h.db.Where("task_id = ?", currentTaskID).
		Order("created_at DESC").
		First(&lastLog)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"running":    running,
			"task_id":    currentTaskID,
			"last_log":   lastLog.Message,
			"last_level": lastLog.Level,
			"last_time":  lastLog.CreatedAt,
		},
	})
}

func (h *MusicHandler) GetScanLogs(c *gin.Context) {
	taskID := c.Query("task_id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	var logs []models.ScanLog
	query := h.db.Model(&models.ScanLog{})

	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	var total int64
	query.Count(&total)

	offset := (getInt(page) - 1) * getInt(pageSize)
	query.Order("created_at DESC").
		Offset(offset).
		Limit(getInt(pageSize)).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"total":     total,
		"page":      getInt(page),
		"page_size": getInt(pageSize),
		"list":      logs,
	})
}

// FetchLyrics 从网络获取歌词
func (h *MusicHandler) FetchLyrics(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	fetcher := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")

	lyricsPath, _, err := fetcher.FetchAndSave(music.Artist, music.Title, music.Album)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "Failed to fetch lyrics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Lyrics fetched successfully",
		"data": gin.H{
			"lyrics_path": lyricsPath,
			"has_lyrics":  true,
		},
	})
}

// FetchCover 从网络获取封面
func (h *MusicHandler) FetchCover(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	fetcher := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")

	_, coverPath, err := fetcher.FetchAndSave(music.Artist, music.Title, music.Album)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "Failed to fetch cover: " + err.Error(),
		})
		return
	}

	// 更新数据库
	music.HasCover = true
	music.CoverMIME = "image/jpeg"
	h.db.Save(&music)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Cover fetched successfully",
		"data": gin.H{
			"cover_path": coverPath,
			"has_cover":  true,
		},
	})
}

// GetBatchStatus 获取批量操作状态
func (h *MusicHandler) GetBatchStatus(c *gin.Context) {
	// ✅ 第一步：加锁
	statusMutex.Lock()

	// ✅ 第二步：安全地拷贝当前状态 (避免返回指针导致的数据竞争)
	currentStatus := *batchStatus

	// ✅ 第三步：立即解锁 (减少锁持有时间，提高并发性能)
	statusMutex.Unlock()

	// ✅ 第四步：使用拷贝出来的局部变量返回给前端
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": currentStatus, // <--- 注意这里用的是 currentStatus，不是 batchStatus
	})
}

// BatchFetchLyrics 批量获取所有音乐歌词
func (h *MusicHandler) BatchFetchLyrics(c *gin.Context) {
	var musicList []models.Music
	// 只获取没有歌词的音乐
	if err := h.getDB().Where("has_lyrics = ? OR has_lyrics IS NULL", false).Find(&musicList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get music list: " + err.Error(),
		})
		return
	}

	if len(musicList) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "No music needs lyrics",
			"data":    gin.H{"total": 0, "success": 0, "failed": 0},
		})
		return
	}
	// ✅ 修复：使用锁保护状态检查
	statusMutex.Lock()
	// ✅ 关键修复：检查并返回
	if batchStatus.Running {
		c.JSON(409, gin.H{
			"code":    409,
			"message": "Another batch task is running",
		})
		return // <--- 必须加这个 return
	}

	// 初始化状态
	batchStatus = &BatchStatus{
		Running:   true,
		TaskType:  "lyrics",
		Total:     len(musicList),
		Current:   0,
		Success:   0,
		Failed:    0,
		Message:   "Starting...",
		CreatedAt: time.Now(),
	}
	statusMutex.Unlock()

	go func() {
		f := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
		success := 0
		failed := 0

		for i, music := range musicList {
			statusMutex.Lock()
			batchStatus.Current = i + 1
			batchStatus.Message = fmt.Sprintf("Processing: %s", music.Title)
			batchStatus.Total = len(musicList) // 确保 Total 始终正确
			statusMutex.Unlock()
			lyricsPath, _, err := f.FetchAndSave(music.Artist, music.Title, music.Album)

			// ✅ 修复：只要 lyricsPath 不为空，就算成功
			if lyricsPath != "" {
				music.HasLyrics = true
				h.getDB().Save(&music)
				success++
				log.Printf("[Batch] ✅ %s: Lyrics fetched", music.Title)
			} else {
				failed++
				log.Printf("[Batch] ❌ %s: Lyrics failed (%v)", music.Title, err)
			}

			// 实时更新状态
			statusMutex.Lock()
			batchStatus.Success = success
			batchStatus.Failed = failed
			statusMutex.Unlock()
			time.Sleep(800 * time.Millisecond)
		}
		statusMutex.Lock()
		batchStatus.Running = false
		batchStatus.Message = "Completed"
		batchStatus.Total = len(musicList)
		batchStatus.Success = success
		batchStatus.Failed = failed
		statusMutex.Unlock()
		log.Printf("[Batch] 🎉 Lyrics batch done: total=%d, success=%d, failed=%d", len(musicList), success, failed)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Batch fetch started",
		"data": gin.H{
			"total":   len(musicList),
			"success": 0,
			"failed":  0,
		},
	})
}

// BatchFetchCovers 批量获取所有音乐封面
func (h *MusicHandler) BatchFetchCovers(c *gin.Context) {
	var musicList []models.Music
	if err := h.getDB().Where("has_cover = ? OR has_cover IS NULL", false).Find(&musicList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get music list: " + err.Error(),
		})
		return
	}

	if len(musicList) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "No music needs covers",
			"data":    gin.H{"total": 0, "success": 0, "failed": 0},
		})
		return
	}
	statusMutex.Lock()
	// ✅ 关键修复：检查并返回
	if batchStatus.Running {
		statusMutex.Unlock()
		c.JSON(409, gin.H{
			"code":    409,
			"message": "Another batch task is running",
		})
		return // <--- 必须加这个 return
	}

	batchStatus = &BatchStatus{
		Running:   true,
		TaskType:  "covers",
		Total:     len(musicList),
		Current:   0,
		Success:   0,
		Failed:    0,
		Message:   "Starting...",
		CreatedAt: time.Now(),
	}
	statusMutex.Unlock()
	go func() {
		f := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
		success := 0
		failed := 0

		for i, music := range musicList {
			statusMutex.Lock()
			batchStatus.Current = i + 1
			batchStatus.Message = fmt.Sprintf("Processing: %s", music.Title)
			batchStatus.Total = len(musicList)
			statusMutex.Unlock()
			_, coverPath, err := f.FetchAndSave(music.Artist, music.Title, music.Album)

			// ✅ 修复：只要 coverPath 不为空，就算成功
			if coverPath != "" {
				music.HasCover = true
				music.CoverMIME = "image/jpeg"
				h.getDB().Save(&music)
				success++
				log.Printf("[Batch] ✅ %s: Cover fetched", music.Title)
			} else {
				failed++
				log.Printf("[Batch] ❌ %s: Cover failed (%v)", music.Title, err)
			}
			statusMutex.Lock()
			batchStatus.Success = success
			batchStatus.Failed = failed
			statusMutex.Unlock()
			time.Sleep(800 * time.Millisecond)
		}
		statusMutex.Lock()
		batchStatus.Running = false
		batchStatus.Message = "Completed"
		batchStatus.Total = len(musicList) // <--- 结束前再设一次
		batchStatus.Success = success
		batchStatus.Failed = failed
		statusMutex.Unlock()
		log.Printf("[Batch] 🎉 Cover batch done: total=%d, success=%d, failed=%d", len(musicList), success, failed)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Batch fetch started",
		"data": gin.H{
			"total":   len(musicList),
			"success": 0,
			"failed":  0,
		},
	})
}

// BatchFetchAll 批量获取歌词和封面
func (h *MusicHandler) BatchFetchAll(c *gin.Context) {
	var musicList []models.Music
	if err := h.getDB().Where("scan_status = ?", "success").Find(&musicList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get music list: " + err.Error(),
		})
		return
	}

	if len(musicList) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "No music found",
			"data":    gin.H{"total": 0, "lyrics_success": 0, "cover_success": 0},
		})
		return
	}

	// ✅ 关键修复：检查并返回
	if batchStatus.Running {
		c.JSON(409, gin.H{
			"code":    409,
			"message": "Another batch task is running",
		})
		return // <--- 必须加这个 return
	}

	batchStatus = &BatchStatus{
		Running:  true,
		TaskType: "all",
		Total:    len(musicList),
		Current:  0,
		Success:  0,
		Failed:   0,
		Message:  "Starting...",
	}

	go func() {
		f := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
		lyricsSuccess := 0
		lyricsFailed := 0
		coverSuccess := 0
		coverFailed := 0

		for i, music := range musicList {
			batchStatus.Current = i + 1
			batchStatus.Message = fmt.Sprintf("Processing: %s", music.Title)
			batchStatus.Total = len(musicList)

			updated := false

			// 获取歌词
			if !music.HasLyrics {
				lyricsPath, _, _ := f.FetchAndSave(music.Artist, music.Title, music.Album)
				if lyricsPath != "" {
					music.HasLyrics = true
					lyricsSuccess++
					updated = true
					log.Printf("[Batch] ✅ %s: Lyrics fetched", music.Title)
				} else {
					lyricsFailed++
					log.Printf("[Batch] ❌ %s: Lyrics failed", music.Title)
				}
			}

			// 获取封面
			if !music.HasCover {
				_, coverPath, _ := f.FetchAndSave(music.Artist, music.Title, music.Album)
				if coverPath != "" {
					music.HasCover = true
					music.CoverMIME = "image/jpeg"
					coverSuccess++
					updated = true
					log.Printf("[Batch] ✅ %s: Cover fetched", music.Title)
				} else {
					coverFailed++
					log.Printf("[Batch] ❌ %s: Cover failed", music.Title)
				}
			}

			if updated {
				h.getDB().Save(&music)
			}

			batchStatus.Success = lyricsSuccess + coverSuccess
			batchStatus.Failed = lyricsFailed + coverFailed

			time.Sleep(800 * time.Millisecond)
		}

		batchStatus.Running = false
		batchStatus.Message = "Completed"
		log.Printf("[Batch] 🎉 All done: Lyrics=%d/%d, Covers=%d/%d",
			lyricsSuccess, lyricsSuccess+lyricsFailed,
			coverSuccess, coverSuccess+coverFailed)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Batch fetch started",
		"data": gin.H{
			"total":   len(musicList),
			"success": 0,
			"failed":  0,
		},
	})
}

// FetchAll 批量获取所有音乐的歌词和封面
func (h *MusicHandler) FetchAll(c *gin.Context) {
	var musicList []models.Music
	h.db.Where("scan_status = ?", "success").Find(&musicList)

	success := 0
	failed := 0

	fetcher := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")

	for _, music := range musicList {
		_, _, err := fetcher.FetchAndSave(music.Artist, music.Title, music.Album)
		if err == nil {
			success++
		} else {
			failed++
		}
		// 避免请求过快被封
		time.Sleep(500 * time.Millisecond)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Batch fetch completed",
		"data": gin.H{
			"success": success,
			"failed":  failed,
			"total":   len(musicList),
		},
	})
}

// GetLocalCover 获取本地封面图片
func (h *MusicHandler) GetLocalCover(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	fetcher := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
	coverPath := fetcher.GetLocalCoverPath(music.Artist, music.Album)

	data, err := os.ReadFile(coverPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Cover not found",
		})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/jpeg", data)
}

// List 列表 (确保也返回正确结构)
func (h *MusicHandler) List(c *gin.Context) {
	// ... 保持原有逻辑，确保返回 data.list 结构 ...
	// 如果前端调用的是 /search，主要修复 Search 函数即可
	// 为了兼容，我们让 List 也返回类似结构或者直接返回列表
	// 这里简化为直接返回列表，因为前端 loadMusicList 主要用 search 接口

	var req MusicListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	query := h.db.Model(&models.Music{}).Where("scan_status = ?", "success")
	// ... 省略筛选逻辑 ...

	var total int64
	query.Count(&total)

	var musicList []models.Music
	offset := (req.Page - 1) * req.PageSize
	query.Offset(offset).Limit(req.PageSize).Find(&musicList)

	var responseList []models.MusicResponse
	for _, m := range musicList {
		responseList = append(responseList, m.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    responseList,
		"total":   total,
	})
}

func (h *MusicHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    music.ToResponse(),
	})
}

func (h *MusicHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	var req UpdateMusicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Title != "" {
		music.Title = req.Title
	}
	if req.Artist != "" {
		music.Artist = req.Artist
	}
	if req.Album != "" {
		music.Album = req.Album
	}
	if req.AlbumArtist != "" {
		music.AlbumArtist = req.AlbumArtist
	}
	if req.Composer != "" {
		music.Composer = req.Composer
	}
	if req.Genre != "" {
		music.Genre = req.Genre
	}
	if req.Year > 0 {
		music.Year = req.Year
	}
	if req.TrackNumber > 0 {
		music.TrackNumber = req.TrackNumber
	}
	if req.DiscNumber > 0 {
		music.DiscNumber = req.DiscNumber
	}
	if req.Comment != "" {
		music.Comment = req.Comment
	}

	music.UpdatedAt = time.Now()

	if err := h.db.Save(&music).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to update: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Music updated successfully",
		"data":    music.ToResponse(),
	})
}

func (h *MusicHandler) BatchUpdate(c *gin.Context) {
	var req BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "No music IDs provided",
		})
		return
	}

	updated := 0
	failed := 0

	for _, id := range req.IDs {
		var music models.Music
		if err := h.db.First(&music, id).Error; err != nil {
			failed++
			continue
		}

		if req.Artist != "" {
			music.Artist = req.Artist
		}
		if req.Album != "" {
			music.Album = req.Album
		}
		if req.Genre != "" {
			music.Genre = req.Genre
		}
		if req.Year > 0 {
			music.Year = req.Year
		}

		music.UpdatedAt = time.Now()
		if err := h.db.Save(&music).Error; err != nil {
			failed++
			continue
		}
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Batch update completed",
		"data": gin.H{
			"updated": updated,
			"failed":  failed,
		},
	})
}

// GetCover 获取封面图片
func (h *MusicHandler) GetCover(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 检查本地封面
	f := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
	localPath := f.GetLocalCoverPath(music.Artist, music.Album)

	// 修复：直接使用，不声明未使用的变量
	if _, err := os.Stat(localPath); err == nil {
		data, err := os.ReadFile(localPath)
		if err == nil {
			c.Header("Content-Type", "image/jpeg")
			c.Header("Cache-Control", "public, max-age=86400")
			c.Data(http.StatusOK, "image/jpeg", data)
			return
		}
	}

	// 没有本地封面，返回 404
	c.Status(http.StatusNotFound)
}

// Play 音乐流式播放 (最终修复版：URL 编码 + HTTP 反向代理)
func (h *MusicHandler) Play(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Music not found"})
		return
	}

	// 1. 获取 WebDAV 配置
	var dbConfig models.WebDAVConfig
	if err := h.db.First(&dbConfig).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "WebDAV config not found"})
		return
	}

	// 2. 【关键修复】构建经过 URL 编码的目标地址
	// 去掉 filePath 开头的斜杠，防止编码后变成 %2F 导致路径错误
	cleanPath := strings.TrimPrefix(music.FilePath, "/")

	// 对路径的每一段分别编码 (保留斜杠 /)
	parts := strings.Split(cleanPath, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part) // 将中文转为 %E8... 格式
	}
	encodedPath := strings.Join(parts, "/")

	// 拼接完整 URL: http://ip:port/dav/%E8%B5%B5...mp3
	targetURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(dbConfig.URL, "/"), encodedPath)

	// 3. 创建代理请求
	req, err := http.NewRequest(c.Request.Method, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to create request: " + err.Error()})
		return
	}

	// 4. 设置认证头 (Basic Auth)
	req.SetBasicAuth(dbConfig.Username, dbConfig.Password)

	// 5. 透传 Range 头 (支持进度条拖拽)
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	// 设置 User-Agent (有些服务器会校验)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// 6. 发送请求到 WebDAV (openlist)
	client := &http.Client{
		Timeout: 0, // 不设置超时，让流式传输自然完成
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "Upstream server error: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// 7. 将上游响应头复制给浏览器
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	if resp.Header.Get("Content-Type") == "" {
		c.Header("Content-Type", "audio/mpeg")
	}
	c.Header("Content-Length", resp.Header.Get("Content-Length"))
	c.Header("Accept-Ranges", resp.Header.Get("Accept-Ranges"))
	c.Header("Content-Range", resp.Header.Get("Content-Range"))
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")

	// 8. 复制状态码 (200 或 206)
	c.Status(resp.StatusCode)

	// 9. 流式拷贝数据 (边读边写，不占内存)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		// 记录错误但不返回 JSON，因为响应流已经开始
		log.Printf("Stream copy error: %v", err)
	}

	// 强制刷新缓冲区
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

// handleRangeRequest 处理 Range 请求
func (h *MusicHandler) handleRangeRequest(c *gin.Context, data []byte, fileSize int64, rangeHeader string) {
	ranges := strings.Split(rangeHeader, "=")
	if len(ranges) < 2 {
		c.Status(http.StatusBadRequest)
		return
	}

	byteRanges := strings.Split(ranges[1], "-")
	start, err := strconv.ParseInt(byteRanges[0], 10, 64)
	if err != nil {
		start = 0
	}

	end := fileSize - 1
	if len(byteRanges) > 1 && byteRanges[1] != "" {
		end, err = strconv.ParseInt(byteRanges[1], 10, 64)
		if err != nil || end >= fileSize {
			end = fileSize - 1
		}
	}

	if start >= int64(len(data)) {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if end >= int64(len(data)) {
		end = int64(len(data)) - 1
	}

	contentLength := end - start + 1

	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
	c.Status(http.StatusPartialContent)

	c.Writer.Write(data[start : end+1])
}

// 修改 handlers/music.go 中的 GetLyrics 函数
func (h *MusicHandler) GetLyrics(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	// 1. 优先读取本地歌词文件
	fetcher := fetcher.NewFetcher("/app/data/lyrics", "/app/data/covers")
	localPath := fetcher.GetLocalLyricsPath(music.Artist, music.Title)

	var lyrics string
	if _, err := os.Stat(localPath); err == nil {
		data, err := os.ReadFile(localPath)
		if err == nil {
			lyrics = string(data)
		}
	}

	// 2. 本地没有则尝试嵌入式歌词
	if lyrics == "" {
		lyrics = h.getEmbeddedLyrics(music)
	}

	// 3. 最后尝试外部 .lrc 文件
	if lyrics == "" {
		lyrics = h.getExternalLyrics(music)
	}

	if lyrics == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "No lyrics available",
			"data": gin.H{
				"lyrics":     "",
				"has_lyrics": false,
			},
		})
		return
	}

	parsedLyrics := h.parseLyrics(lyrics)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"lyrics":     lyrics,
			"parsed":     parsedLyrics,
			"has_lyrics": true,
		},
	})
}

func (h *MusicHandler) getEmbeddedLyrics(music models.Music) string {
	client, err := h.getWebDAVClient()
	if err != nil {
		return ""
	}

	data, err := client.GetFile(music.FilePath)
	if err != nil {
		return ""
	}

	reader := bytes.NewReader(data)
	md, err := tag.ReadFrom(reader)
	if err != nil {
		return ""
	}

	if lyrics := md.Lyrics(); lyrics != "" {
		return lyrics
	}

	return ""
}

func (h *MusicHandler) getExternalLyrics(music models.Music) string {
	client, err := h.getWebDAVClient()
	if err != nil {
		return ""
	}

	lrcPath := strings.TrimSuffix(music.FilePath, filepath.Ext(music.FilePath)) + ".lrc"

	data, err := client.GetFile(lrcPath)
	if err != nil {
		return ""
	}

	return string(data)
}

// parseLyrics 解析歌词 (增强容错性)
func (h *MusicHandler) parseLyrics(lyrics string) []gin.H {
	var parsed []gin.H
	lines := strings.Split(lyrics, "\n")

	// 匹配时间轴 [mm:ss.xx]
	timeRegex := regexp.MustCompile(`^\[(\d{2}):(\d{2})\.(\d{2,3})\]`)

	hasTimeTag := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			hasTimeTag = true
			mins, _ := strconv.Atoi(matches[1])
			secs, _ := strconv.Atoi(matches[2])
			millis, _ := strconv.Atoi(matches[3])
			if len(matches[3]) == 2 {
				millis *= 10
			}

			seconds := float64(mins*60+secs) + float64(millis)/1000.0

			text := timeRegex.ReplaceAllString(line, "")
			text = strings.TrimSpace(text)

			// 过滤掉纯音乐标记如 [00:00.00]
			if text != "" && !strings.HasPrefix(text, "[") && !strings.HasPrefix(text, "作") && !strings.HasPrefix(text, "曲") {
				parsed = append(parsed, gin.H{
					"time":    seconds,
					"text":    text,
					"seconds": seconds,
				})
			}
		} else if !hasTimeTag {
			// 如果没有时间轴标签，尝试作为纯文本歌词处理 (每行间隔 3 秒)
			// 这样可以防止只显示一行
			parsed = append(parsed, gin.H{
				"time":    float64(len(parsed)) * 3.0,
				"text":    line,
				"seconds": float64(len(parsed)) * 3.0,
			})
		}
	}

	// 如果解析后仍然为空，返回一个提示
	if len(parsed) == 0 {
		return []gin.H{
			{"time": 0, "text": "暂无歌词", "seconds": 0},
		}
	}

	return parsed
}

func (h *MusicHandler) GetPlaylist(c *gin.Context) {
	artist := c.Query("artist")
	album := c.Query("album")
	genre := c.Query("genre")
	limit := c.DefaultQuery("limit", "50")

	query := h.db.Model(&models.Music{}).Where("scan_status = ?", "success")

	if artist != "" {
		query = query.Where("artist = ?", artist)
	}
	if album != "" {
		query = query.Where("album = ?", album)
	}
	if genre != "" {
		query = query.Where("genre = ?", genre)
	}

	var musicList []models.Music
	query.Limit(getInt(limit)).Find(&musicList)

	var responseList []models.MusicResponse
	for _, m := range musicList {
		responseList = append(responseList, m.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    responseList,
	})
}

func (h *MusicHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Music{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to delete",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *MusicHandler) DeleteAll(c *gin.Context) {
	if err := h.db.Where("1=1").Delete(&models.Music{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to delete all",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// Statistics 统计信息
func (h *MusicHandler) Statistics(c *gin.Context) {
	var total int64

	// ✅ 关键修复：统计 ALL 歌曲，不要加任何 WHERE 条件
	h.getDB().Model(&models.Music{}).Count(&total)

	// 可选：统计有歌词的
	var withLyrics int64
	h.getDB().Model(&models.Music{}).Where("has_lyrics = ?", true).Count(&withLyrics)

	// 可选：统计有封面的
	var withCover int64
	h.getDB().Model(&models.Music{}).Where("has_cover = ?", true).Count(&withCover)

	var topArtists []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	h.getDB().Table("music").
		Select("artist as name, COUNT(*) as count").
		Where("artist != '' AND artist IS NOT NULL").
		Group("artist").
		Order("count DESC").
		Limit(10).
		Scan(&topArtists)

	log.Printf("[Stats] Total: %d, WithLyrics: %d, WithCover: %d", total, withLyrics, withCover) // 调试日志

	var topAlbums []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	h.getDB().Table("music").
		Select("album as name, COUNT(*) as count").
		Where("album != '' AND album IS NOT NULL").
		Group("album").
		Order("count DESC").
		Limit(10).
		Scan(&topAlbums)

	var topGenres []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	h.getDB().Table("music").
		Select("genre as name, COUNT(*) as count").
		Where("genre != '' AND genre IS NOT NULL").
		Group("genre").
		Order("count DESC").
		Limit(10).
		Scan(&topGenres)

	log.Printf("[Stats] Total: %d, Artists: %v", total, topArtists) // 调试日志

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":       total,
			"top_artists": topArtists,
			"top_albums":  topAlbums,
			"top_genres":  topGenres,
		},
	})
}

// Search 搜索音乐 (修复返回结构)
func (h *MusicHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page := getInt(pageStr)
	pageSize := getInt(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&models.Music{}).Where("scan_status = ?", "success")

	if keyword != "" {
		query = query.Where("title LIKE ? OR artist LIKE ? OR album LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var musicList []models.Music
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&musicList)

	var responseList []models.MusicResponse
	for _, m := range musicList {
		responseList = append(responseList, m.ToResponse())
	}

	// 返回正确的分页结构
	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      responseList, // 直接返回列表
		"total":     total,        // 同时返回总数供参考
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *MusicHandler) GetWebDAVConfig(c *gin.Context) {
	var config models.WebDAVConfig
	result := h.db.First(&config)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    config.ToResponse(),
	})
}

func (h *MusicHandler) SaveWebDAVConfig(c *gin.Context) {
	var req WebDAVConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	var config models.WebDAVConfig
	result := h.db.First(&config)

	now := time.Now()
	if result.Error == gorm.ErrRecordNotFound {
		config = models.WebDAVConfig{
			URL:        req.URL,
			Username:   req.Username,
			Password:   req.Password,
			RootPath:   req.RootPath,
			Enabled:    req.Enabled,
			TestStatus: "pending",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		h.db.Create(&config)
	} else {
		config.URL = req.URL
		config.Username = req.Username
		if req.Password != "" {
			config.Password = req.Password
		}
		config.RootPath = req.RootPath
		config.Enabled = req.Enabled
		config.UpdatedAt = now
		config.TestStatus = "pending"
		h.db.Save(&config)
	}

	h.resetWebDAVClient()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "WebDAV config saved successfully",
		"data":    config.ToResponse(),
	})
}

// TestWebDAVConfig 测试 WebDAV 连接
func (h *MusicHandler) TestWebDAVConfig(c *gin.Context) {
	var req WebDAVConfigRequest

	// 1. 接收前端传来的参数 (URL, Username, Password, RootPath)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：" + err.Error(),
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "服务器地址不能为空",
		})
		return
	}

	// 默认路径处理 (对齐 app.js 可能的默认行为)
	if req.RootPath == "" {
		req.RootPath = "/dav"
	}

	log.Printf("[WebDAV Test] Testing connection to: %s (Path: %s, User: %s)", req.URL, req.RootPath, req.Username)

	// 2. 创建客户端 (使用传入的参数)
	client := webdav.NewClientNoCheck(req.URL, req.Username, req.Password, req.RootPath)

	// 3. ✅ 关键修复：使用递归查找，确保能扫到子目录下的文件 (解决显示 0 个的问题)
	files, err := client.ListMP3FilesRecursive()

	// 如果递归查找返回空或报错，尝试只查当前目录 (兼容模式)
	if err != nil || len(files) == 0 {
		log.Printf("[WebDAV Test] Recursive search found %d files (err: %v). Trying flat list...", len(files), err)
		flatFiles, flatErr := client.ListMP3Files()
		if flatErr == nil && len(flatFiles) > 0 {
			files = flatFiles
			err = nil
		} else if len(files) == 0 && len(flatFiles) == 0 {
			// 真的没找到文件，尝试列出所有文件看看目录下到底有啥 (调试用)
			allItems, readErr := client.ReadDirAll(req.RootPath)
			if readErr == nil && len(allItems) > 0 {
				log.Printf("[WebDAV Test] Directory '%s' contains %d items, but no MP3s found.", req.RootPath, len(allItems))
				// 构造一个友好的提示
				err = fmt.Errorf("目录中存在 %d 个文件，但未发现 .mp3 格式文件。请检查路径或文件格式。", len(allItems))
			} else if readErr != nil {
				err = fmt.Errorf("读取目录失败：%v", readErr)
			} else {
				err = fmt.Errorf("目录为空")
			}
		}
	}

	// 4. 处理测试结果
	now := time.Now()

	// 尝试更新数据库中的测试状态 (如果有记录的话)
	var dbConfig models.WebDAVConfig
	dbHasRecord := h.db.First(&dbConfig).Error == nil

	if err != nil {
		errMsg := err.Error()
		log.Printf("[WebDAV Test] FAILED: %s", errMsg)

		if dbHasRecord {
			dbConfig.TestStatus = "failed"
			dbConfig.TestError = errMsg
			dbConfig.LastTest = &now
			h.db.Save(&dbConfig)
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "连接失败：" + getShortErrorMsg(errMsg),
		})
		return
	}

	// 5. 成功！
	count := len(files)
	log.Printf("[WebDAV Test] SUCCESS: Found %d MP3 files.", count)

	if dbHasRecord {
		dbConfig.TestStatus = "success"
		dbConfig.TestError = ""
		dbConfig.LastTest = &now
		h.db.Save(&dbConfig)
	}

	// ✅ 返回文件数量
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "连接成功",
		"data": gin.H{
			"count": count,
		},
	})
}

// getShortErrorMsg 辅助函数：提取简短错误信息
func getShortErrorMsg(err string) string {
	if strings.Contains(err, "401") {
		return "用户名或密码错误"
	}
	if strings.Contains(err, "404") {
		return "路径不存在"
	}
	if strings.Contains(err, "405") {
		return "方法不允许 (请尝试根路径为 / 或 /dav)"
	}
	if strings.Contains(err, "timeout") {
		return "连接超时"
	}
	if strings.Contains(err, "connection refused") {
		return "无法连接服务器"
	}
	if len(err) > 100 {
		return err[:100] + "..."
	}
	return err
}

func (h *MusicHandler) TestWebDAVCustom(c *gin.Context) {
	var req WebDAVConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	client := webdav.NewClientNoCheck(req.URL, req.Username, req.Password, req.RootPath)

	files, err := client.ListMP3Files()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "Connection failed",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Connection successful",
		"data": gin.H{
			"files_found": len(files),
			"url":         req.URL,
			"root_path":   req.RootPath,
		},
	})
}
func (h *MusicHandler) RefreshTagsFromMusicBrainz(c *gin.Context) {
	id := c.Param("id")

	var music models.Music
	if err := h.db.First(&music, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Music not found",
		})
		return
	}

	// 创建 MusicBrainz 客户端
	mb := parser.NewMusicBrainzClient()

	// 搜索标签信息
	mbInfo, err := mb.SearchTrack(music.Artist, music.Title)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "MusicBrainz lookup failed: " + err.Error(),
		})
		return
	}

	// 更新数据库
	updated := false

	if mbInfo.Title != "" && music.Title == "" {
		music.Title = mbInfo.Title
		updated = true
	}
	if mbInfo.Artist != "" && music.Artist == "" {
		music.Artist = mbInfo.Artist
		updated = true
	}
	if mbInfo.Album != "" && music.Album == "" {
		music.Album = mbInfo.Album
		updated = true
	}
	if mbInfo.Year > 0 && music.Year == 0 {
		music.Year = mbInfo.Year
		updated = true
	}

	if !updated {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "No new information found",
			"data":    music.ToResponse(),
		})
		return
	}

	music.UpdatedAt = time.Now()
	if err := h.db.Save(&music).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to save: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Tags refreshed from MusicBrainz",
		"data":    music.ToResponse(),
	})
}
func (h *MusicHandler) DeleteWebDAVConfig(c *gin.Context) {
	result := h.db.Where("1=1").Delete(&models.WebDAVConfig{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": result.Error.Error(),
		})
		return
	}

	h.resetWebDAVClient()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "WebDAV config deleted",
	})
}

func getInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
