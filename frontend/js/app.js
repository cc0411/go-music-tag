const API_BASE = '/api/v1';

let currentPage = 1;
let totalPages = 1;
let scanInterval = null;
let currentMusicId = null;
let selectedMusicIds = [];

let playlist = [];
let currentTrackIndex = -1;
let isPlaying = false;
let isShuffle = false;
let repeatMode = 0;
let lyrics = [];
let currentLyricIndex = -1;
let isBuffering = false;
let lastErrorTime = 0;
let retryCount = 0;
const MAX_RETRY = 3;

const audio = document.getElementById('audio-player');

document.addEventListener('DOMContentLoaded', () => {
    initNavigation();
    checkServerStatus();
    loadDashboard();
    loadWebDAVConfig();
    initPlayer();
    setInterval(checkServerStatus, 30000);
});

function initNavigation() {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            nav(item.dataset.page);
        });
    });
    
    window.addEventListener('hashchange', () => {
        nav(window.location.hash.slice(1) || 'dashboard');
    });
    
    nav(window.location.hash.slice(1) || 'dashboard');
}

function nav(page) {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.page === page);
    });
    
    document.querySelectorAll('.page').forEach(p => {
        p.classList.toggle('active', p.id === `${page}-page`);
    });
    
    const titles = {
        dashboard: '仪表盘',
        music: '音乐库',
        player: '播放器',
        webdav: 'WebDAV 配置',
        scan: '扫描管理'
    };
    document.getElementById('page-title').textContent = titles[page] || '仪表盘';
    
    if (page === 'music') loadMusicList();
    if (page === 'scan') loadScanLogs();
    if (page === 'player') {
        if (playlist.length === 0) loadPlaylist();
    }
}

function refreshCurrentPage() {
    const activePage = document.querySelector('.page.active').id.replace('-page', '');
    if (activePage === 'dashboard') loadDashboard();
    if (activePage === 'music') loadMusicList();
    if (activePage === 'player') loadPlaylist();
    if (activePage === 'scan') loadScanLogs();
    showToast('已刷新', 'success');
}

async function checkServerStatus() {
    try {
        await fetch(`${API_BASE}/health`);
        document.getElementById('server-status').classList.add('online');
        document.getElementById('status-text').textContent = '在线';
    } catch (e) {
        document.getElementById('server-status').classList.remove('online');
        document.getElementById('status-text').textContent = '离线';
    }
}

// 修复仪表盘加载
async function loadDashboard() {
    try {
        const res = await fetch(`${API_BASE}/statistics`);
        if (!res.ok) throw new Error('Network response was not ok');
        
        const json = await res.json();
        
        // ✅ 关键：先打印出来，方便调试
        console.log('Dashboard Data Received:', json.data);

        if (!json.data) return;
        const data = json.data;

        // 1. 更新总数卡片
        document.getElementById('stat-total').textContent = data.total || 0;

        // 2. 处理艺术家数据
        const artists = data.top_artists || []; // 确保是数组
        // ✅ 修复：显示艺术家的数量（数组长度）
        const artistCountEl = document.getElementById('stat-artists');
        if (artistCountEl) {
            artistCountEl.textContent = artists.length; 
        }

        // 3. 处理专辑数据
        const albums = data.top_albums || [];
        const albumCountEl = document.getElementById('stat-albums');
        if (albumCountEl) {
            albumCountEl.textContent = albums.length;
        }

        // 4. 处理流派数据
        const genres = data.top_genres || []; // 后端返回 null 时，这里会变成 []
        const genreCountEl = document.getElementById('stat-genres');
        if (genreCountEl) {
            genreCountEl.textContent = genres.length;
        }

        // 5. ✅ 关键修复：渲染"热门艺术家"列表
        const artistsListEl = document.getElementById('top-artists');
        if (artistsListEl) {
            if (artists.length === 0) {
                artistsListEl.innerHTML = '<div class="no-data" style="padding:20px;text-align:center;color:#999;">暂无数据</div>';
            } else {
                // 生成 HTML
                artistsListEl.innerHTML = artists.map((item, index) => `
                    <div class="list-item" style="display:flex;justify-content:space-between;padding:12px;border-bottom:1px solid #eee;">
                        <div style="display:flex;align-items:center;gap:10px;">
                            <span style="background:#f3f4f6;width:24px;height:24px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:bold;color:#666;">${index + 1}</span>
                            <span style="font-weight:500;">${escapeHtml(item.name)}</span>
                        </div>
                        <span style="color:#666;font-size:0.9rem;">${item.count} 首</span>
                    </div>
                `).join('');
            }
        }

        // 6. 渲染"热门专辑"列表 (如果页面有的话)
        const albumsListEl = document.getElementById('top-albums');
        if (albumsListEl) {
             if (albums.length === 0) {
                albumsListEl.innerHTML = '<div class="no-data" style="padding:20px;text-align:center;color:#999;">暂无数据</div>';
            } else {
                albumsListEl.innerHTML = albums.map((item, index) => `
                    <div class="list-item" style="display:flex;justify-content:space-between;padding:12px;border-bottom:1px solid #eee;">
                        <div style="display:flex;align-items:center;gap:10px;">
                            <span style="background:#f3f4f6;width:24px;height:24px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:bold;color:#666;">${index + 1}</span>
                            <span style="font-weight:500;">${escapeHtml(item.name)}</span>
                        </div>
                        <span style="color:#666;font-size:0.9rem;">${item.count} 首</span>
                    </div>
                `).join('');
            }
        }
        
        // 7. 渲染"热门流派"列表
        const genresListEl = document.getElementById('top-genres');
        if (genresListEl) {
             if (genres.length === 0) {
                genresListEl.innerHTML = '<div class="no-data" style="padding:20px;text-align:center;color:#999;">暂无数据</div>';
            } else {
                genresListEl.innerHTML = genres.map((item, index) => `
                    <div class="list-item" style="display:flex;justify-content:space-between;padding:12px;border-bottom:1px solid #eee;">
                        <div style="display:flex;align-items:center;gap:10px;">
                            <span style="background:#f3f4f6;width:24px;height:24px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:bold;color:#666;">${index + 1}</span>
                            <span style="font-weight:500;">${escapeHtml(item.name)}</span>
                        </div>
                        <span style="color:#666;font-size:0.9rem;">${item.count} 首</span>
                    </div>
                `).join('');
            }
        }

    } catch (e) {
        console.error('加载仪表盘失败:', e);
    }
}


// 修复音乐库加载
async function loadMusicList(page = 1) {
    currentPage = page;
    const tbody = document.getElementById('music-list');
    
    // 显示加载状态
    tbody.innerHTML = '<tr><td colspan="6" class="loading"><div class="spinner"></div>加载中...</td></tr>';

    try {
        const params = new URLSearchParams({ 
            page: page, 
            page_size: 20 
        });
        const search = document.getElementById('music-search').value;
        if (search) params.append('keyword', search);
        
        const res = await fetch(`${API_BASE}/music/search?${params}`);
        if (!res.ok) throw new Error('Network response was not ok');
        
        const json = await res.json();
        
        // 兼容不同的返回结构
        let list = [];
        let total = 0;
        
        if (json.code === 0) {
            if (Array.isArray(json.data)) {
                list = json.data;
                total = json.total || list.length;
            } else if (json.data && Array.isArray(json.data.list)) {
                list = json.data.list;
                total = json.data.total || json.total || list.length;
            }
        }

        renderMusicList(list);
        
        totalPages = Math.ceil(total / 20) || 1;
        document.getElementById('page-info').textContent = `第 ${page} / ${totalPages} 页`;
        document.getElementById('prev-btn').disabled = page <= 1;
        document.getElementById('next-btn').disabled = page >= totalPages;
        
        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">暂无音乐，请先扫描</td></tr>';
        }
    } catch (e) {
        console.error('加载音乐列表失败:', e);
        tbody.innerHTML = `<tr><td colspan="6" class="loading" style="color:red;">加载失败：${e.message}<br><button class="btn btn-sm" onclick="loadMusicList(${page})" style="margin-top:10px;">重试</button></td></tr>`;
        showToast('加载音乐列表失败', 'error');
    }
}

function renderMusicList(data) {
    const tbody = document.getElementById('music-list');
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">暂无音乐</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(m => `
        <tr>
            <td><input type="checkbox" class="select-checkbox" value="${m.id}" onchange="toggleSelect(${m.id})"></td>
            <td class="col-title">${escapeHtml(m.title || m.file_name || '未知标题')}</td>
            <td class="col-artist">${escapeHtml(m.artist || '-')}</td>
            <td class="col-album">${escapeHtml(m.album || '-')}</td>
            <td class="col-duration">${m.duration_str || '-'}</td>
            <td class="col-actions">
                <div class="music-actions">
                    <button class="btn btn-sm" onclick="viewMusic(${m.id})">查看</button>
                    <button class="btn btn-primary btn-sm" onclick="editMusic(${m.id})">编辑</button>
                    <button class="btn btn-danger btn-sm" onclick="deleteMusic(${m.id})">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

function toggleSelectAll() {
    const checkbox = document.querySelector('.music-table thead .select-checkbox');
    document.querySelectorAll('.music-table tbody .select-checkbox').forEach(cb => {
        cb.checked = checkbox.checked;
        const id = parseInt(cb.value);
        if (checkbox.checked) {
            if (!selectedMusicIds.includes(id)) selectedMusicIds.push(id);
        } else {
            selectedMusicIds = selectedMusicIds.filter(i => i !== id);
        }
    });
    updateBatchToolbar();
}

function toggleSelect(id) {
    if (selectedMusicIds.includes(id)) {
        selectedMusicIds = selectedMusicIds.filter(i => i !== id);
    } else {
        selectedMusicIds.push(id);
    }
    updateBatchToolbar();
}

function updateBatchToolbar() {
    const toolbar = document.getElementById('batch-toolbar');
    const count = document.getElementById('batch-count');
    const countModal = document.getElementById('batch-count-modal');
    if (selectedMusicIds.length > 0) {
        toolbar.classList.add('active');
        count.textContent = selectedMusicIds.length;
        countModal.textContent = selectedMusicIds.length;
    } else {
        toolbar.classList.remove('active');
    }
}

function searchMusic() { loadMusicList(1); }
function prevPage() { if (currentPage > 1) loadMusicList(currentPage - 1); }
function nextPage() { if (currentPage < totalPages) loadMusicList(currentPage + 1); }

async function viewMusic(id) {
    currentMusicId = id;
    try {
        const res = await fetch(`${API_BASE}/music/${id}`);
        const data = await res.json();
        if (data.code === 0) {
            const m = data.data;
            document.getElementById('detail-path').textContent = m.file_path || '-';
            document.getElementById('detail-size').textContent = m.file_size_str || '-';
            document.getElementById('detail-title').textContent = m.title || '-';
            document.getElementById('detail-artist').textContent = m.artist || '-';
            document.getElementById('detail-album').textContent = m.album || '-';
            document.getElementById('detail-duration').textContent = m.duration_str || '-';
            document.getElementById('detail-year').textContent = m.year || '-';
            document.getElementById('detail-genre').textContent = m.genre || '-';
            showViewMode();
            document.getElementById('music-modal').classList.add('active');
        }
    } catch (e) {
        showToast('加载失败', 'error');
    }
}

async function editMusic(id) {
    currentMusicId = id;
    try {
        const res = await fetch(`${API_BASE}/music/${id}`);
        const data = await res.json();
        if (data.code === 0) {
            const m = data.data;
            document.getElementById('edit-id').value = m.id;
            document.getElementById('edit-title').value = m.title || '';
            document.getElementById('edit-artist').value = m.artist || '';
            document.getElementById('edit-album').value = m.album || '';
            document.getElementById('edit-genre').value = m.genre || '';
            document.getElementById('edit-year').value = m.year || '';
            document.getElementById('edit-track').value = m.track_number || '';
            showEditMode();
            document.getElementById('music-modal').classList.add('active');
        }
    } catch (e) {
        showToast('加载失败', 'error');
    }
}

async function saveMusicEdit(e) {
    e.preventDefault();
    const id = document.getElementById('edit-id').value;
    const data = {
        title: document.getElementById('edit-title').value,
        artist: document.getElementById('edit-artist').value,
        album: document.getElementById('edit-album').value,
        genre: document.getElementById('edit-genre').value,
        year: parseInt(document.getElementById('edit-year').value) || 0,
        track_number: parseInt(document.getElementById('edit-track').value) || 0
    };
    try {
        const res = await fetch(`${API_BASE}/music/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const result = await res.json();
        if (result.code === 0) {
            showToast('保存成功', 'success');
            closeModal();
            loadMusicList(currentPage);
        } else {
            showToast('保存失败：' + result.message, 'error');
        }
    } catch (e) {
        showToast('保存失败', 'error');
    }
}

function showViewMode() {
    document.getElementById('view-mode').style.display = 'block';
    document.getElementById('edit-mode').style.display = 'none';
    document.getElementById('modal-title').textContent = '音乐详情';
}

function showEditMode() {
    document.getElementById('view-mode').style.display = 'none';
    document.getElementById('edit-mode').style.display = 'block';
    document.getElementById('modal-title').textContent = '编辑音乐';
}

function closeModal() {
    document.getElementById('music-modal').classList.remove('active');
    currentMusicId = null;
}

async function deleteCurrentMusic() {
    if (!currentMusicId || !confirm('确定删除这首音乐？')) return;
    await deleteMusic(currentMusicId);
    closeModal();
}

async function deleteMusic(id) {
    if (!confirm('确定删除这首音乐？')) return;
    try {
        await fetch(`${API_BASE}/music/${id}`, { method: 'DELETE' });
        showToast('已删除', 'success');
        loadMusicList(currentPage);
        loadDashboard();
    } catch (e) {
        showToast('删除失败', 'error');
    }
}

async function refreshTags() {
    if (!currentMusicId) return;
    try {
        showToast('正在刷新标签...', 'info');
        const res = await fetch(`${API_BASE}/music/${currentMusicId}/refresh`, { method: 'POST' });
        const result = await res.json();
        if (result.code === 0) {
            showToast('标签已更新', 'success');
            viewMusic(currentMusicId);
            loadMusicList(currentPage);
        } else {
            showToast('刷新失败：' + result.message, 'error');
        }
    } catch (e) {
        showToast('刷新失败', 'error');
    }
}

function showBatchModal() {
    document.getElementById('batch-count-modal').textContent = selectedMusicIds.length;
    document.getElementById('batch-modal').classList.add('active');
}

function closeBatchModal() {
    document.getElementById('batch-modal').classList.remove('active');
}

async function saveBatchEdit(e) {
    e.preventDefault();
    const data = {
        ids: selectedMusicIds,
        artist: document.getElementById('batch-artist').value,
        album: document.getElementById('batch-album').value,
        genre: document.getElementById('batch-genre').value,
        year: parseInt(document.getElementById('batch-year').value) || 0
    };
    try {
        const res = await fetch(`${API_BASE}/music/batch`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const result = await res.json();
        if (result.code === 0) {
            showToast(`批量更新完成：成功${result.data.updated}，失败${result.data.failed}`, 'success');
            closeBatchModal();
            selectedMusicIds = [];
            updateBatchToolbar();
            loadMusicList(currentPage);
        } else {
            showToast('批量更新失败', 'error');
        }
    } catch (e) {
        showToast('批量更新失败', 'error');
    }
}

async function batchDelete() {
    if (!confirm(`确定删除选中的 ${selectedMusicIds.length} 首音乐？`)) return;
    let success = 0, failed = 0;
    for (const id of selectedMusicIds) {
        try {
            await fetch(`${API_BASE}/music/${id}`, { method: 'DELETE' });
            success++;
        } catch (e) {
            failed++;
        }
    }
    showToast(`删除完成：成功${success}，失败${failed}`, success > 0 ? 'success' : 'error');
    selectedMusicIds = [];
    updateBatchToolbar();
    loadMusicList(currentPage);
    loadDashboard();
}

function playSelected() {
    if (selectedMusicIds.length === 0) {
        showToast('请先选择音乐', 'warning');
        return;
    }
    loadPlaylistByIds(selectedMusicIds);
    nav('player');
}

async function loadPlaylistByIds(ids) {
    try {
        const promises = ids.map(id => fetch(`${API_BASE}/music/${id}`).then(r => r.json()));
        const results = await Promise.all(promises);
        playlist = results.filter(r => r.code === 0).map(r => r.data);
        currentTrackIndex = 0;
        renderPlaylist();
        playTrack(0);
    } catch (e) {
        showToast('加载播放列表失败', 'error');
    }
}

function playCurrentMusic() {
    if (currentMusicId) {
        loadPlaylistByIds([currentMusicId]);
        nav('player');
    }
}

function initPlayer() {
    if (!audio) return;
    audio.preload = 'metadata';
    
    audio.addEventListener('timeupdate', updateProgress);
    audio.addEventListener('loadedmetadata', () => {
        document.getElementById('duration').textContent = formatTime(audio.duration);
        updatePlayStatus('playing', '已加载');
    });
    audio.addEventListener('waiting', () => {
        isBuffering = true;
        updatePlayStatus('loading', '缓冲中...');
    });
    audio.addEventListener('playing', () => {
        isBuffering = false;
        updatePlayStatus('playing', '播放中');
        retryCount = 0;
    });
    audio.addEventListener('pause', () => {
        isBuffering = false;
        updatePlayStatus('paused', '已暂停');
    });
    audio.addEventListener('error', (e) => {
        handlePlayError(e);
    });
    audio.addEventListener('canplay', () => {
        updatePlayStatus('ready', '可播放');
    });
    audio.addEventListener('progress', () => {
        updateBufferProgress();
    });
}

function updatePlayStatus(status, text) {
    const indicator = document.querySelector('.status-indicator');
    const statusText = document.getElementById('status-text');
    const playBtn = document.getElementById('play-btn');
    
    indicator.className = 'status-indicator ' + status;
    statusText.textContent = text;
    
    if (status === 'loading') {
        playBtn.classList.add('loading');
        document.getElementById('cover-loading').style.display = 'flex';
        document.getElementById('play-error').style.display = 'none';
    } else {
        playBtn.classList.remove('loading');
        document.getElementById('cover-loading').style.display = 'none';
    }
    
    if (status === 'error') {
        document.getElementById('play-error').style.display = 'flex';
    }
}

function updateBufferProgress() {
    if (!audio.buffered || audio.buffered.length === 0) return;
    const buffered = audio.buffered.end(audio.buffered.length - 1);
    const duration = audio.duration;
    if (duration > 0) {
        const percent = (buffered / duration) * 100;
        document.getElementById('progress-buffer').style.width = `${percent}%`;
    }
}

function handlePlayError(e) {
    console.error('播放错误:', e);
    isBuffering = false;
    
    const errorMessages = {
        1: '媒体加载中止',
        2: '网络错误',
        3: '解码错误',
        4: '格式不支持'
    };
    
    const errorCode = audio.error?.code || 0;
    const errorMsg = errorMessages[errorCode] || '播放失败';
    
    document.getElementById('error-text').textContent = errorMsg;
    updatePlayStatus('error', errorMsg);
    showToast('播放失败：' + errorMsg, 'error');
    
    lastErrorTime = Date.now();
    retryCount++;
}

function retryPlay() {
    if (retryCount >= MAX_RETRY) {
        showToast('重试次数过多，请稍后再试', 'error');
        return;
    }
    
    if (Date.now() - lastErrorTime < 3000) {
        showToast('请稍后再试', 'warning');
        return;
    }
    
    document.getElementById('play-error').style.display = 'none';
    
    if (currentTrackIndex >= 0 && playlist[currentTrackIndex]) {
        playTrack(currentTrackIndex);
    }
}

async function loadPlaylist() {
    try {
        const res = await fetch(`${API_BASE}/music/playlist?limit=100`);
        const data = await res.json();
        if (data.code === 0 && data.data) {
            playlist = data.data;
            renderPlaylist();
            document.getElementById('playlist-count').textContent = `${playlist.length} 首`;
        }
    } catch (e) {
        console.error('加载播放列表失败:', e);
    }
}

function renderPlaylist() {
    const container = document.getElementById('playlist');
    if (!playlist || playlist.length === 0) {
        container.innerHTML = '<p class="empty-playlist">暂无播放列表</p>';
        return;
    }
    container.innerHTML = playlist.map((track, index) => `
        <div class="playlist-item ${index === currentTrackIndex ? 'active' : ''}" onclick="playTrack(${index})">
            <span class="playlist-item-icon">${index === currentTrackIndex ? '🎵' : '🎶'}</span>
            <div class="playlist-item-info">
                <div class="playlist-item-title">${escapeHtml(track.title || track.file_name)}</div>
                <div class="playlist-item-artist">${escapeHtml(track.artist || '-')}</div>
            </div>
            <span>${track.duration_str || '-'}</span>
        </div>
    `).join('');
}
// 获取歌词
async function fetchLyrics() {
    if (!currentMusicId) return;
    
    try {
        showToast('正在获取歌词...', 'info');
        const res = await fetch(`${API_BASE}/music/${currentMusicId}/fetch-lyrics`, {
            method: 'POST'
        });
        const result = await res.json();
        if (result.code === 0) {
            showToast('歌词获取成功', 'success');
            // 重新加载歌词
            loadLyrics(currentMusicId);
        } else {
            showToast('获取失败：' + result.message, 'error');
        }
    } catch (e) {
        showToast('获取失败', 'error');
    }
}

// 获取封面
async function fetchCover() {
    if (!currentMusicId) return;
    
    try {
        showToast('正在获取封面...', 'info');
        const res = await fetch(`${API_BASE}/music/${currentMusicId}/fetch-cover`, {
            method: 'POST'
        });
        const result = await res.json();
        if (result.code === 0) {
            showToast('封面获取成功', 'success');
            // 刷新页面
            location.reload();
        } else {
            showToast('获取失败：' + result.message, 'error');
        }
    } catch (e) {
        showToast('获取失败', 'error');
    }
}

// 批量获取
async function fetchAllResources() {
    if (!confirm('确定要批量获取所有音乐的歌词和封面吗？可能需要较长时间。')) return;
    
    try {
        showToast('正在批量获取...', 'info');
        const res = await fetch(`${API_BASE}/music/fetch-all`, {
            method: 'POST'
        });
        const result = await res.json();
        if (result.code === 0) {
            showToast(`完成：成功${result.data.success}，失败${result.data.failed}`, 'success');
        } else {
            showToast('批量获取失败', 'error');
        }
    } catch (e) {
        showToast('批量获取失败', 'error');
    }
}
// 修复播放器加载
async function playTrack(index) {
    if (index < 0 || index >= playlist.length) return;
    
    currentTrackIndex = index;
    const track = playlist[index];
    
    if (!track || !track.id) {
        showToast('无效的音乐文件', 'error');
        return;
    }

    document.getElementById('player-title').textContent = track.title || track.file_name || '未知曲目';
    document.getElementById('player-artist').textContent = track.artist || '-';
    document.getElementById('player-album').textContent = track.album || '-';
    
    updatePlayStatus('loading', '连接中...');
    document.getElementById('play-error').style.display = 'none';
    
    const playUrl = `${API_BASE}/music/${track.id}/play`;
    
    // 重置音频
    audio.pause();
    audio.src = playUrl;
    audio.load();
    
    // 添加超时检测
    const loadTimeout = setTimeout(() => {
        if (audio.readyState < 3) { // HAVE_FUTURE_DATA
            handlePlayError(new Error('加载超时'));
        }
    }, 10000);

    audio.oncanplay = () => {
        clearTimeout(loadTimeout);
        updatePlayStatus('ready', '可播放');
        audio.play().catch(e => {
            console.error('自动播放失败:', e);
            updatePlayStatus('paused', '点击播放');
        });
    };
    
    audio.onerror = (e) => {
        clearTimeout(loadTimeout);
        handlePlayError(e);
    };

    loadLyrics(track.id);
    renderPlaylist();
}

function togglePlay() {
    if (!audio.src) {
        if (playlist.length > 0) playTrack(0);
        return;
    }
    
    if (isBuffering) return;
    
    if (isPlaying) {
        audio.pause();
    } else {
        audio.play().catch(e => {
            console.error('播放失败:', e);
            handlePlayError(e);
        });
    }
}

function prevTrack() {
    if (playlist.length === 0) return;
    let newIndex = currentTrackIndex - 1;
    if (newIndex < 0) {
        newIndex = repeatMode === 2 ? currentTrackIndex : playlist.length - 1;
    }
    playTrack(newIndex);
}

function nextTrack() {
    if (playlist.length === 0) return;
    let newIndex;
    if (isShuffle) {
        newIndex = Math.floor(Math.random() * playlist.length);
    } else {
        newIndex = currentTrackIndex + 1;
        if (newIndex >= playlist.length) {
            newIndex = repeatMode === 2 ? currentTrackIndex : 0;
        }
    }
    playTrack(newIndex);
}

function updateProgress() {
    const progress = (audio.currentTime / audio.duration) * 100;
    document.getElementById('progress-fill').style.width = `${progress}%`;
    document.getElementById('progress-thumb').style.left = `${progress}%`;
    document.getElementById('current-time').textContent = formatTime(audio.currentTime);
    updateLyricsHighlight(audio.currentTime);
}

function seek(event) {
    const bar = document.getElementById('progress-bar');
    const rect = bar.getBoundingClientRect();
    const percent = (event.clientX - rect.left) / rect.width;
    audio.currentTime = percent * audio.duration;
}

function setVolume(value) {
    audio.volume = value / 100;
    updateVolumeIcon(value);
}

function toggleMute() {
    if (audio.muted) {
        audio.muted = false;
        document.getElementById('volume-slider').value = audio.volume * 100;
        updateVolumeIcon(audio.volume * 100);
    } else {
        audio.muted = true;
        updateVolumeIcon(0);
    }
}

function updateVolumeIcon(value) {
    const icon = document.getElementById('volume-icon');
    if (value == 0 || audio.muted) {
        icon.textContent = '🔇';
    } else if (value < 50) {
        icon.textContent = '🔉';
    } else {
        icon.textContent = '🔊';
    }
}

function setRepeatMode() {
    repeatMode = (repeatMode + 1) % 3;
    const icon = document.getElementById('repeat-icon');
    const btn = icon.parentElement;
    const modes = ['🔁', '🔂', '🔀'];
    const titles = ['列表循环', '单曲循环', '不循环'];
    icon.textContent = modes[repeatMode];
    btn.classList.toggle('active', repeatMode !== 0);
    showToast(titles[repeatMode]);
}

function toggleShuffle() {
    isShuffle = !isShuffle;
    const icon = document.getElementById('shuffle-icon');
    const btn = icon.parentElement;
    icon.textContent = isShuffle ? '🔁' : '🔀';
    btn.classList.toggle('active', isShuffle);
    showToast(isShuffle ? '随机播放已开启' : '随机播放已关闭');
}
// 批量获取歌词

let expectedTotal = 0; 
async function batchFetchLyrics() {
    if (!confirm('确定要批量获取所有音乐的歌词吗？这可能需要较长时间。')) return;
    
    openBatchModal('批量获取歌词');
    
    try {
        const res = await fetch(`${API_BASE}/music/batch-fetch-lyrics`, {
            method: 'POST'
        });
        const result = await res.json();
        
        if (result.code === 0) {
            expectedTotal = result.data.total;
            
            console.log('Initial total:', expectedTotal); // 调试日志
            
            updateBatchProgress(0, expectedTotal, `共 ${expectedTotal} 首音乐，开始获取...`);
            updateBatchStats(expectedTotal, 0, 0);
            
            pollBatchStatus();
        } else {
            alert('启动失败：' + result.message);
            closeBatchModal();
        }
    } catch (e) {
        alert('网络错误：' + e.message);
        closeBatchModal();
    }
}

// 批量获取封面
async function batchFetchCovers() {
    if (!confirm('确定要批量获取所有音乐的封面吗？这可能需要较长时间。')) return;
    
    openBatchModal('批量获取封面');
    
    try {
        const res = await fetch(`${API_BASE}/music/batch-fetch-covers`, {
            method: 'POST'
        });
        const result = await res.json();
        
        if (result.code === 0) {
            expectedTotal = result.data.total;
            
            console.log('Initial total:', expectedTotal); // 调试日志
            
            updateBatchProgress(0, expectedTotal, `共 ${expectedTotal} 首音乐，开始获取...`);
            updateBatchStats(expectedTotal, 0, 0);
            
            pollBatchStatus();
        } else {
            alert('启动失败：' + result.message);
            closeBatchModal();
        }
    } catch (e) {
        alert('网络错误：' + e.message);
        closeBatchModal();
    }
}

// 批量获取全部
async function batchFetchAll() {
    if (!confirm('确定要批量获取所有音乐的歌词和封面吗？这可能需要较长时间。')) return;
    
    openBatchModal('批量获取歌词和封面');
    
    try {
        const res = await fetch(`${API_BASE}/music/batch-fetch-all`, {
            method: 'POST'
        });
        const result = await res.json();
        
        if (result.code === 0) {
            const total = result.data.total;
            updateBatchProgress(0, total, `共 ${total} 首音乐，开始获取...`);
            updateBatchStats(total, 0, 0);
            pollBatchStatus();
        } else {
            alert('启动失败：' + result.message);
            closeBatchModal();
        }
    } catch (e) {
        alert('网络错误：' + e.message);
        closeBatchModal();
    }
}

// 轮询批量操作状态
let pollInterval = null;
function pollBatchStatus() {
    if (pollInterval) clearInterval(pollInterval);
    
    pollInterval = setInterval(async () => {
        try {
            const res = await fetch(`${API_BASE}/music/batch-status`);
            const result = await res.json();
            
            if (result.code === 0 && result.data) {
                const status = result.data;
                
                // ✅ 关键修复：始终使用 expectedTotal，忽略后端返回的 status.total
                const displayTotal = expectedTotal > 0 ? expectedTotal : (status.total || 0);
                
                console.log('Polling - Total:', displayTotal, 'Success:', status.success, 'Failed:', status.failed);
                
                updateBatchProgress(status.current, displayTotal, status.message);
                updateBatchStats(displayTotal, status.success, status.failed);
                
                if (!status.running) {
                    clearInterval(pollInterval);
                    pollInterval = null;
                    addBatchLog('✅ 批量获取完成！', 'success');
                    
                    setTimeout(() => {
                        loadMusicList(1);
                        loadDashboard(); // 刷新仪表盘
                    }, 2000);
                }
            }
        } catch (e) {
            console.error('轮询状态失败:', e);
        }
    }, 2000);
}
// 模态框控制
function openBatchModal(title) {
    document.getElementById('batch-title').textContent = title;
    document.getElementById('batch-modal').classList.add('active');
    document.getElementById('batch-log').innerHTML = '';
    addBatchLog('🚀 任务已启动...', 'info');
}

function closeBatchModal() {
    document.getElementById('batch-modal').classList.remove('active');
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

function updateBatchProgress(current, total, text) {
    const percent = total > 0 ? Math.round((current / total) * 100) : 0;
    document.getElementById('progress-fill').style.width = percent + '%';
    document.getElementById('progress-text').textContent = text;
}

function updateBatchStats(total, success, failed) {
    document.getElementById('stat-total').textContent = total;
    document.getElementById('stat-success').textContent = success;
    document.getElementById('stat-failed').textContent = failed;
}

function addBatchLog(message, type) {
    const logEl = document.getElementById('batch-log');
    const p = document.createElement('p');
    p.className = type;
    p.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logEl.appendChild(p);
    logEl.scrollTop = logEl.scrollHeight;
}

// 渲染音乐列表时添加状态列
function renderMusicList(data) {
    const tbody = document.getElementById('music-list');
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="loading">暂无音乐</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(m => `
        <tr>
            <td>
                <input type="checkbox" class="select-checkbox" value="${m.id}" onchange="toggleSelect(${m.id})">
            </td>
            <td class="col-title">${escapeHtml(m.title || m.file_name || '未知标题')}</td>
            <td class="col-artist">${escapeHtml(m.artist || '-')}</td>
            <td class="col-album">${escapeHtml(m.album || '-')}</td>
            <td class="col-duration">${m.duration_str || '-'}</td>
            <td class="col-status">
                ${m.has_lyrics ? '<span class="status-badge has-lyrics">📝 有歌词</span>' : '<span class="status-badge no-lyrics">无歌词</span>'}
                ${m.has_cover ? '<span class="status-badge has-cover">🖼️ 有封面</span>' : '<span class="status-badge no-cover">无封面</span>'}
            </td>
            <td class="col-actions">
                <div class="music-actions">
                    <button class="btn btn-sm" onclick="viewMusic(${m.id})">查看</button>
                    <button class="btn btn-success btn-sm" onclick="fetchMusicResources(${m.id})">获取资源</button>
                    <button class="btn btn-danger btn-sm" onclick="deleteMusic(${m.id})">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

// 获取单首音乐的资源
async function fetchMusicResources(id) {
    if (!confirm('获取这首音乐的歌词和封面？')) return;
    
    try {
        showToast('正在获取...', 'info');
        
        // 获取歌词
        await fetch(`${API_BASE}/music/${id}/fetch-lyrics`, { method: 'POST' });
        
        // 获取封面
        await fetch(`${API_BASE}/music/${id}/fetch-cover`, { method: 'POST' });
        
        showToast('获取成功', 'success');
        loadMusicList(currentPage);
    } catch (e) {
        showToast('获取失败', 'error');
    }
}
async function loadLyrics(musicId) {
    try {
        const res = await fetch(`${API_BASE}/music/${musicId}/lyrics`);
        const data = await res.json();
        const container = document.getElementById('lyrics-container');
        
        console.log('歌词响应:', data); // 添加调试日志
        
        if (data.code === 0 && data.data.has_lyrics && data.data.parsed) {
            lyrics = data.data.parsed;
            console.log('解析后的歌词行数:', lyrics.length); // 检查行数
            
            if (lyrics.length === 0 || (lyrics.length === 1 && lyrics[0].text === '暂无歌词')) {
                 container.innerHTML = '<p class="no-lyrics">暂无歌词</p>';
                 return;
            }

            container.innerHTML = lyrics.map((line, index) => `
                <div class="lyric-line" data-index="${index}" onclick="seekToLyric(${index})">
                    ${escapeHtml(line.text)}
                </div>
            `).join('');
            
            // 重置高亮
            currentLyricIndex = -1;
        } else {
            container.innerHTML = '<p class="no-lyrics">暂无歌词</p>';
            lyrics = [];
        }
    } catch (e) {
        console.error('加载歌词失败:', e);
        document.getElementById('lyrics-container').innerHTML = '<p class="no-lyrics">加载失败</p>';
    }
}

function updateLyricsHighlight(currentTime) {
    if (!lyrics || lyrics.length === 0) return;
    let newIndex = -1;
    for (let i = 0; i < lyrics.length; i++) {
        if (lyrics[i].time <= currentTime) {
            newIndex = i;
        } else {
            break;
        }
    }
    if (newIndex !== currentLyricIndex && newIndex >= 0) {
        currentLyricIndex = newIndex;
        const container = document.getElementById('lyrics-container');
        container.querySelectorAll('.lyric-line').forEach((line, index) => {
            line.classList.toggle('active', index === currentLyricIndex);
        });
        const activeLine = container.querySelector('.lyric-line.active');
        if (activeLine) {
            activeLine.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }
}

function seekToLyric(index) {
    if (lyrics && lyrics[index]) {
        audio.currentTime = lyrics[index].time;
        audio.play();
    }
}

function formatTime(seconds) {
    if (!seconds || isNaN(seconds)) return '0:00';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}

async function loadWebDAVConfig() {
    try {
        const res = await fetch(`${API_BASE}/webdav/config`);
        const data = await res.json();
        if (data.code === 0 && data.data) {
            document.getElementById('webdav-url').value = data.data.url || '';
            document.getElementById('webdav-username').value = data.data.username || '';
            document.getElementById('webdav-password').value = data.data.password || '';
            document.getElementById('webdav-rootpath').value = data.data.root_path || '';
            document.getElementById('webdav-enabled').checked = data.data.enabled;
        }
    } catch (e) {
        console.error(e);
    }
}

async function saveWebDAVConfig(e) {
    e.preventDefault();
    const config = {
        url: document.getElementById('webdav-url').value,
        username: document.getElementById('webdav-username').value,
        password: document.getElementById('webdav-password').value,
        root_path: document.getElementById('webdav-rootpath').value,
        enabled: document.getElementById('webdav-enabled').checked
    };
    try {
        const res = await fetch(`${API_BASE}/webdav/config`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        const data = await res.json();
        showToast(data.code === 0 ? '保存成功' : '保存失败', data.code === 0 ? 'success' : 'error');
        if (data.code === 0) loadWebDAVConfig();
    } catch (e) {
        showToast('保存失败', 'error');
    }
}

async function testWebDAV() {
    const statusDiv = document.getElementById('webdav-status');
    statusDiv.className = 'status-message show';
    statusDiv.textContent = '测试中...';
    try {
        const res = await fetch(`${API_BASE}/webdav/test`, { method: 'POST' });
        const data = await res.json();
        statusDiv.className = `status-message show ${data.code === 0 ? 'success' : 'error'}`;
        statusDiv.textContent = data.code === 0 ? `连接成功！找到 ${data.data.files_found} 个文件` : '连接失败：' + data.message;
    } catch (e) {
        statusDiv.className = 'status-message show error';
        statusDiv.textContent = '连接失败：' + e.message;
    }
}

async function deleteWebDAVConfig() {
    if (!confirm('确定删除配置？')) return;
    try {
        await fetch(`${API_BASE}/webdav/config`, { method: 'DELETE' });
        showToast('已删除', 'success');
        loadWebDAVConfig();
    } catch (e) {
        showToast('删除失败', 'error');
    }
}

async function startScan() {
    const recursive = document.getElementById('scan-recursive').checked;
    try {
        const res = await fetch(`${API_BASE}/scan`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ recursive })
        });
        const data = await res.json();
        if (data.code === 0) {
            showToast('扫描已启动', 'success');
            document.getElementById('scan-btn').disabled = true;
            startScanPolling();
        }
    } catch (e) {
        showToast('启动失败', 'error');
    }
}

function startScanPolling() {
    scanInterval = setInterval(async () => {
        try {
            const res = await fetch(`${API_BASE}/scan/status`);
            const data = await res.json();
            document.getElementById('scan-status').textContent = data.data.last_log || '扫描中...';
            if (!data.data.running) {
                stopScanPolling();
                document.getElementById('scan-btn').disabled = false;
                showToast('扫描完成', 'success');
                loadDashboard();
                loadMusicList();
            }
        } catch (e) {
            console.error(e);
        }
    }, 2000);
}

function stopScanPolling() {
    if (scanInterval) {
        clearInterval(scanInterval);
        scanInterval = null;
    }
}

async function loadScanLogs() {
    try {
        const res = await fetch(`${API_BASE}/scan/logs?page=1&page_size=50`);
        const data = await res.json();
        if (data.code === 0 && data.list) {
            document.getElementById('scan-logs').innerHTML = data.list.map(log => 
                `<p class="${log.level}">[${new Date(log.created_at).toLocaleString()}] ${log.message}</p>`
            ).join('') || '<p class="no-logs">暂无日志</p>';
        }
    } catch (e) {
        console.error(e);
    }
}

function showToast(message, type = 'info') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast ${type} show`;
    setTimeout(() => toast.classList.remove('show'), 3000);
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

document.getElementById('music-modal')?.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal-overlay')) closeModal();
});

document.getElementById('batch-modal')?.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal-overlay')) closeBatchModal();
});