// API base URL
const API_BASE = window.location.origin + window.location.pathname.replace(/indexer\.html$/, '').replace(/\/$/, '');

// Global state
let walletConnected = false;
let currentAddress = null;
let currentMetaID = null;
let filesCursor = 0;
let hasMoreFiles = true;
let isLoadingFiles = false;
let usersCursor = 0;
let hasMoreUsers = true;
let isLoadingUsers = false;
let currentTab = 'myFiles';

// DOM elements
const connectBtn = document.getElementById('connectBtn');
const disconnectBtn = document.getElementById('disconnectBtn');
const walletStatus = document.getElementById('walletStatus');
const walletAddress = document.getElementById('walletAddress');
const addressText = document.getElementById('addressText');
const metaidText = document.getElementById('metaidText');
const walletAlert = document.getElementById('walletAlert');
const fileListSection = document.getElementById('fileListSection');
const fileListContainer = document.getElementById('fileListContainer');
const loadMoreBtn = document.getElementById('loadMoreBtn');
const refreshStatusBtn = document.getElementById('refreshStatusBtn');
const refreshFilesBtn = document.getElementById('refreshFilesBtn');
const userAvatarContainer = document.getElementById('userAvatarContainer');
const userAvatar = document.getElementById('userAvatar');
const avatarPlaceholder = document.getElementById('avatarPlaceholder');
const userListContainer = document.getElementById('userListContainer');
const loadMoreUsersBtn = document.getElementById('loadMoreUsersBtn');
const refreshUsersBtn = document.getElementById('refreshUsersBtn');

// Status elements
const totalFilesEl = document.getElementById('totalFiles');
const chainStatusContainer = document.getElementById('chainStatusContainer');

// Initialization
window.addEventListener('load', () => {
    console.log('🚀 Indexer page loaded');
    initWalletCheck();
    initEventListeners();
    loadIndexerStatus();
    initTabs();
    
    // Auto refresh status every 30 seconds
    setInterval(loadIndexerStatus, 30000);
});

// Check Metalet wallet
function initWalletCheck() {
    const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const isInApp = window.navigator.standalone || window.matchMedia('(display-mode: standalone)').matches;
    
    const walletObject = detectWallet();
    
    if (walletObject) {
        handleWalletDetected(walletObject);
    } else if (isMobile || isInApp) {
        console.log('Mobile environment, retrying wallet detection...');
        retryWalletDetection(3, 1000);
    }
}

function detectWallet() {
    if (typeof window.metaidwallet !== 'undefined') {
        return { object: window.metaidwallet, type: 'Metalet Wallet' };
    }
    return null;
}

function handleWalletDetected(walletInfo) {
    window.detectedWallet = walletInfo.object;
    window.walletType = walletInfo.type;
    walletAlert.classList.add('hidden');
}

function retryWalletDetection(attempts, delay) {
    if (attempts <= 0) {
        walletAlert.classList.remove('hidden');
        return;
    }
    
    setTimeout(() => {
        const walletObject = detectWallet();
        if (walletObject) {
            handleWalletDetected(walletObject);
        } else {
            retryWalletDetection(attempts - 1, delay);
        }
    }, delay);
}

function getWallet() {
    return window.detectedWallet || window.metaidwallet;
}

// Initialize event listeners
function initEventListeners() {
    if (connectBtn) {
        connectBtn.addEventListener('click', connectWallet);
    }
    
    if (disconnectBtn) {
        disconnectBtn.addEventListener('click', disconnectWallet);
    }
    
    if (refreshStatusBtn) {
        refreshStatusBtn.addEventListener('click', loadIndexerStatus);
    }
    
    if (refreshFilesBtn) {
        refreshFilesBtn.addEventListener('click', refreshFileList);
    }
    
    if (loadMoreBtn) {
        loadMoreBtn.addEventListener('click', loadMoreFiles);
    }
    
    if (refreshUsersBtn) {
        refreshUsersBtn.addEventListener('click', refreshUserList);
    }
    
    if (loadMoreUsersBtn) {
        loadMoreUsersBtn.addEventListener('click', loadMoreUsers);
    }
}

// Initialize tabs
function initTabs() {
    const tabButtons = document.querySelectorAll('.tab-btn');
    tabButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const tabName = btn.getAttribute('data-tab');
            switchTab(tabName);
        });
    });
    
    // Load users list on page load
    loadUserList();
}

// Switch tab
function switchTab(tabName) {
    currentTab = tabName;
    
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => {
        if (btn.getAttribute('data-tab') === tabName) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });
    
    // Update tab contents
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    if (tabName === 'myFiles') {
        document.getElementById('myFilesTab').classList.add('active');
    } else if (tabName === 'allUsers') {
        document.getElementById('allUsersTab').classList.add('active');
    }
}

// Connect wallet
async function connectWallet() {
    console.log('🔵 Connecting wallet...');
    
    const wallet = getWallet();
    if (!wallet) {
        showNotification('Please install Metalet wallet extension first!', 'error');
        return;
    }

    try {
        connectBtn.disabled = true;
        connectBtn.textContent = 'Connecting...';
        
        const account = await wallet.connect();
        console.log('📱 Wallet account:', account);
        
        const address = account.address || account.mvcAddress || account.btcAddress;
        console.log('📍 Address extracted:', address);
        
        if (account && address) {
            currentAddress = address;
            walletConnected = true;
            
            walletStatus.textContent = 'Connected';
            walletStatus.style.color = '#28a745';
            
            addressText.textContent = currentAddress;
            
            // Calculate MetaID
            console.log('🔐 Calculating MetaID for address:', currentAddress);
            currentMetaID = await calculateMetaID(currentAddress);
            console.log('🆔 MetaID calculated:', currentMetaID);
            
            if (!currentMetaID) {
                console.error('❌ Failed to calculate MetaID!');
                showNotification('Failed to calculate MetaID. Please use localhost or HTTPS.', 'error');
                // Don't return, still show wallet info
            }
            
            metaidText.textContent = currentMetaID || 'Calculation failed';
            
            walletAddress.classList.remove('hidden');
            walletAlert.classList.add('hidden');
            
            connectBtn.classList.add('hidden');
            disconnectBtn.classList.remove('hidden');
            
            showNotification('Wallet connected successfully!', 'success');
            
            console.log('✅ Wallet connected:', {
                address: currentAddress,
                metaID: currentMetaID,
                metaIDAvailable: !!currentMetaID
            });
            
            // Switch to My Files tab
            switchTab('myFiles');
            
            // Load user avatar (only if MetaID is available)
            if (currentMetaID) {
                loadUserAvatar();
                loadUserFiles();
            } else {
                console.warn('⚠️ Skipping file/avatar load - MetaID not available');
                showNotification('MetaID calculation failed. Files cannot be loaded.', 'warning');
            }
        }
    } catch (error) {
        console.error('Failed to connect wallet:', error);
        showNotification('Failed to connect wallet: ' + error.message, 'error');
        connectBtn.disabled = false;
        connectBtn.textContent = 'Connect Metalet Wallet';
    }
}

// Disconnect wallet
function disconnectWallet() {
    walletConnected = false;
    currentAddress = null;
    currentMetaID = null;
    
    walletStatus.textContent = 'Not Connected';
    walletStatus.style.color = '#999';
    walletAddress.classList.add('hidden');
    
    connectBtn.classList.remove('hidden');
    connectBtn.textContent = 'Connect Metalet Wallet';
    connectBtn.disabled = false;
    
    disconnectBtn.classList.add('hidden');
    
    // Hide avatar
    userAvatarContainer.classList.add('hidden');
    userAvatar.style.display = 'none';
    avatarPlaceholder.style.display = 'flex';
    userAvatar.src = '';
    
    fileListSection.classList.add('hidden');
    fileListContainer.innerHTML = '';
    
    showNotification('Wallet disconnected', 'info');
}

// Calculate MetaID (SHA256 of address)
async function calculateMetaID(address) {
    console.log('🔐 Starting MetaID calculation...');
    console.log('   Address:', address);
    console.log('   crypto.subtle available:', typeof crypto !== 'undefined' && typeof crypto.subtle !== 'undefined');
    console.log('   Current protocol:', window.location.protocol);
    console.log('   Current host:', window.location.host);
    
    if (!address) {
        console.error('❌ Address is empty!');
        return '';
    }
    
    // Check if crypto.subtle is available
    if (typeof crypto === 'undefined' || typeof crypto.subtle === 'undefined') {
        console.error('❌ crypto.subtle not available!');
        console.error('   This usually means:');
        console.error('   1. Not using HTTPS or localhost');
        console.error('   2. Using IP address (127.0.0.1) instead of localhost');
        console.error('   Solution: Access via https:// or http://localhost:7281');
        return '';
    }
    
    try {
        console.log('🔨 Encoding address...');
        const encoder = new TextEncoder();
        const data = encoder.encode(address);
        console.log('   Encoded bytes:', data.length);
        
        console.log('🔨 Computing SHA-256 hash...');
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        console.log('   Hash buffer length:', hashBuffer.byteLength);
        
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        console.log('✅ MetaID calculated:', hashHex);
        
        return hashHex;
    } catch (error) {
        console.error('❌ Failed to calculate MetaID:', error);
        console.error('   Error name:', error.name);
        console.error('   Error message:', error.message);
        console.error('   Error stack:', error.stack);
        return '';
    }
}

// Load user avatar
async function loadUserAvatar() {
    if (!currentMetaID) {
        console.log('MetaID not available, cannot load avatar');
        return;
    }
    
    try {
        console.log('Loading avatar for MetaID:', currentMetaID);
        
        // Show avatar container
        userAvatarContainer.classList.remove('hidden');
        
        // Try to get avatar by MetaID
        const response = await fetch(`${API_BASE}/api/v1/avatars/metaid/${currentMetaID}`);
        const data = await response.json();
        
        if (data.code === 0 && data.data) {
            const avatar = data.data;
            const avatarContentUrl = `${API_BASE}/api/v1/avatars/content/${avatar.pin_id}`;
            
            console.log('✅ Avatar found:', avatar);
            
            // Load avatar image
            userAvatar.src = avatarContentUrl;
            userAvatar.style.display = 'block';
            avatarPlaceholder.style.display = 'none';
            
            // Handle image load error
            userAvatar.onerror = () => {
                console.warn('Failed to load avatar image, showing placeholder');
                userAvatar.style.display = 'none';
                avatarPlaceholder.style.display = 'flex';
            };
            
            // Handle image load success
            userAvatar.onload = () => {
                console.log('✅ Avatar image loaded successfully');
            };
        } else {
            // No avatar found, show placeholder
            console.log('No avatar found for MetaID:', currentMetaID);
            userAvatar.style.display = 'none';
            avatarPlaceholder.style.display = 'flex';
        }
    } catch (error) {
        console.error('Failed to load avatar:', error);
        // On error, show placeholder
        userAvatar.style.display = 'none';
        avatarPlaceholder.style.display = 'flex';
    }
}

// Load indexer status
async function loadIndexerStatus() {
    try {
        // Get sync status from API (multi-chain format)
        const statusResponse = await fetch(`${API_BASE}/api/v1/status`);
        const statusData = await statusResponse.json();
        
        // Get statistics (total files count and per-chain breakdown)
        const statsResponse = await fetch(`${API_BASE}/api/v1/stats`);
        const statsData = await statsResponse.json();
        
        if (statusData.code === 0 && statusData.data && statsData.code === 0 && statsData.data) {
            const chains = statusData.data.chains || [];
            const stats = statsData.data;
            
            // Update total files
            totalFilesEl.textContent = stats.total_files.toLocaleString();
            console.log('📊 Total files:', stats.total_files);
            
            // Clear chain status container
            chainStatusContainer.innerHTML = '';
            
            // Render each chain's status
            chains.forEach(chain => {
                const chainCard = createChainStatusCard(chain, stats.chain_stats);
                chainStatusContainer.appendChild(chainCard);
            });
            
            console.log('✅ Multi-chain status loaded:', chains);
        } else {
            throw new Error(statusData.message || statsData.message || 'Failed to load status');
        }
    } catch (error) {
        console.error('Failed to load indexer status:', error);
        totalFilesEl.textContent = 'Error';
        chainStatusContainer.innerHTML = '<div class="empty-state"><p style="color: #dc3545;">Failed to load status</p></div>';
    }
}

// Create chain status card
function createChainStatusCard(chain, chainStats) {
    const card = document.createElement('div');
    card.className = 'chain-status-section';
    
    const chainName = chain.chain_name.toUpperCase();
    const chainClass = chain.chain_name.toLowerCase();
    const currentHeight = chain.current_sync_height;
    const latestHeight = chain.latest_block_height;
    const fileCount = chainStats[chain.chain_name] || 0;
    
    // Calculate progress
    let progressPercent = 0;
    let progressText = '✅ Synced';
    let behindBlocks = 0;
    
    if (latestHeight > 0) {
        if (currentHeight >= latestHeight) {
            progressPercent = 100;
            progressText = '✅ Synced';
        } else {
            progressPercent = (currentHeight / latestHeight * 100).toFixed(2);
            behindBlocks = latestHeight - currentHeight;
            progressText = `⏳ Syncing (${progressPercent}%, ${behindBlocks.toLocaleString()} blocks behind)`;
        }
    } else {
        progressText = '✅ Running';
        progressPercent = 100;
    }
    
    card.innerHTML = `
        <div class="chain-status-header">
            <div style="display: flex; align-items: center; gap: 10px;">
                <span class="chain-name-badge ${chainClass}">${chainName}</span>
                <span style="font-size: 13px; color: #666;">${progressText}</span>
            </div>
            <div style="font-size: 14px; font-weight: 600; color: #667eea;">
                ${fileCount.toLocaleString()} files
            </div>
        </div>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin-bottom: 10px;">
            <div>
                <div style="font-size: 12px; color: #999; margin-bottom: 4px;">Current Height</div>
                <div style="font-size: 16px; font-weight: bold; color: #333;">${currentHeight.toLocaleString()}</div>
            </div>
            <div>
                <div style="font-size: 12px; color: #999; margin-bottom: 4px;">Latest Height</div>
                <div style="font-size: 16px; font-weight: bold; color: #333;">${latestHeight > 0 ? latestHeight.toLocaleString() : '-'}</div>
            </div>
            ${behindBlocks > 0 ? `
            <div>
                <div style="font-size: 12px; color: #999; margin-bottom: 4px;">Behind</div>
                <div style="font-size: 16px; font-weight: bold; color: #ffc107;">${behindBlocks.toLocaleString()} blocks</div>
            </div>
            ` : ''}
        </div>
        ${latestHeight > 0 ? `
        <div class="chain-progress-bar">
            <div class="chain-progress-fill ${chainClass}" style="width: ${progressPercent}%"></div>
        </div>
        ` : ''}
        <div style="margin-top: 8px; font-size: 11px; color: #999; text-align: right;">
            Updated: ${new Date(chain.updated_at).toLocaleString()}
        </div>
    `;
    
    return card;
}

// Load user files
async function loadUserFiles() {
    console.log('📂 loadUserFiles called, currentMetaID:', currentMetaID);
    
    if (!currentMetaID) {
        console.error('❌ MetaID not available, cannot load files');
        showNotification('MetaID not available', 'error');
        return;
    }
    
    console.log('🔍 Showing file list section...');
    fileListSection.classList.remove('hidden');
    fileListContainer.innerHTML = '<div class="loading"><div class="spinner"></div><p style="margin-top: 10px;">Loading your files...</p></div>';
    
    try {
        filesCursor = 0;
        hasMoreFiles = true;
        
        const apiUrl = `${API_BASE}/api/v1/files/metaid/${currentMetaID}?cursor=0&size=20`;
        console.log('📡 Fetching files from:', apiUrl);
        
        const response = await fetch(apiUrl);
        console.log('📥 Response status:', response.status);
        
        const data = await response.json();
        console.log('📊 Response data:', data);
        
        if (data.code === 0) {
            const files = data.data?.files || [];
            const nextCursor = data.data?.next_cursor || 0;
            hasMoreFiles = data.data?.has_more || false;
            
            console.log(`✅ Files loaded: ${files.length} files, hasMore: ${hasMoreFiles}`);
            
            if (files.length === 0) {
                console.log('📭 No files found for this MetaID');
                fileListContainer.innerHTML = `
                    <div class="empty-state">
                        <div class="empty-state-icon">📭</div>
                        <p>No files found</p>
                        <p style="font-size: 14px; margin-top: 10px;">Upload your first file to get started!</p>
                    </div>
                `;
                loadMoreBtn.classList.add('hidden');
            } else {
                filesCursor = nextCursor;
                console.log(`📋 Rendering ${files.length} files...`);
                renderFiles(files, true);
                
                if (hasMoreFiles) {
                    loadMoreBtn.classList.remove('hidden');
                } else {
                    loadMoreBtn.classList.add('hidden');
                }
                console.log('✅ Files rendered successfully');
            }
        } else {
            console.error('❌ API returned error:', data);
            throw new Error(data.message || 'Failed to load files');
        }
    } catch (error) {
        console.error('Failed to load user files:', error);
        fileListContainer.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load files</p>
                <p style="font-size: 14px; margin-top: 10px;">${error.message}</p>
            </div>
        `;
        showNotification('Failed to load files: ' + error.message, 'error');
    }
}

// Load more files
async function loadMoreFiles() {
    if (!currentMetaID || isLoadingFiles || !hasMoreFiles) {
        return;
    }
    
    isLoadingFiles = true;
    loadMoreBtn.disabled = true;
    loadMoreBtn.textContent = 'Loading...';
    
    try {
        const response = await fetch(`${API_BASE}/api/v1/files/metaid/${currentMetaID}?cursor=${filesCursor}&size=20`);
        const data = await response.json();
        
        if (data.code === 0) {
            const files = data.data.files || [];
            const nextCursor = data.data.next_cursor || 0;
            hasMoreFiles = data.data.has_more || false;
            
            if (files.length > 0) {
                filesCursor = nextCursor;
                renderFiles(files, false);
            }
            
            if (!hasMoreFiles) {
                loadMoreBtn.classList.add('hidden');
            }
        } else {
            throw new Error(data.message || 'Failed to load more files');
        }
    } catch (error) {
        console.error('Failed to load more files:', error);
        showNotification('Failed to load more files: ' + error.message, 'error');
    } finally {
        isLoadingFiles = false;
        loadMoreBtn.disabled = false;
        loadMoreBtn.textContent = 'Load More';
    }
}

// Refresh file list
function refreshFileList() {
    if (currentMetaID) {
        loadUserFiles();
    }
}

// Render files
function renderFiles(files, clearFirst) {
    if (clearFirst) {
        fileListContainer.innerHTML = '';
    }
    
    files.forEach(file => {
        const fileCard = createFileCard(file);
        fileListContainer.appendChild(fileCard);
    });
}

// Create file card
function createFileCard(file) {
    const card = document.createElement('div');
    card.className = 'file-card';
    
    const chainBadgeClass = file.chain_name === 'btc' ? 'badge-btc' : 'badge-mvc';
    const chainName = file.chain_name.toUpperCase();
    
    const fileSize = formatFileSize(file.file_size);
    const createdAt = new Date(file.timestamp).toLocaleString();
    
    let pinIds = file.pin_id.split('i');
    let txId = pinIds[0];
    
    // Build view links
    const txUrl = `https://www.mvcscan.com/tx/${txId}`;
    const pinUrl = `https://man.metaid.io/pin/${file.pin_id}`;
    const contentUrl = `${API_BASE}/api/v1/files/content/${file.pin_id}`;
    
    // Check if file is an image or video
    const isImage = file.file_type === 'image' || (file.content_type && file.content_type.startsWith('image/'));
    const isVideo = file.file_type === 'video' || (file.content_type && file.content_type.startsWith('video/'));
    
    // Build preview HTML
    let previewHtml = '';
    if (isImage) {
        previewHtml = `
            <div style="margin: 15px 0; text-align: center; background: #f0f0f0; border-radius: 8px; padding: 10px;">
                <img src="${contentUrl}" 
                     alt="${file.file_name || 'Image'}" 
                     style="max-width: 100%; max-height: 300px; border-radius: 8px; cursor: pointer;"
                     onclick="window.open('${contentUrl}', '_blank')"
                     onerror="this.parentElement.innerHTML='<p style=\\'color: #999; padding: 20px;\\'>Failed to load image preview</p>'">
            </div>
        `;
    } else if (isVideo) {
        // Get video preview thumbnail URL
        const videoPreviewUrl = `${API_BASE}/api/v1/files/accelerate/content/${file.pin_id}?process=video`;
        const videoContainerId = `video-preview-${file.pin_id.replace(/[^a-zA-Z0-9]/g, '-')}`;
        previewHtml = `
            <div class="video-preview-container" id="${videoContainerId}" onclick="playVideo('${contentUrl}', '${file.file_name || 'Video'}')">
                <img src="${videoPreviewUrl}" 
                     alt="${file.file_name || 'Video'}" 
                     class="video-preview-image"
                     onerror="handleVideoPreviewError('${videoContainerId}')">
                <div class="video-preview-placeholder">
                    <div style="font-size: 48px; margin-bottom: 10px;">🎬</div>
                    <div style="font-size: 14px;">点击播放视频</div>
                </div>
                <div class="video-play-overlay">
                    <div class="video-play-icon"></div>
                </div>
            </div>
        `;
    }
    
    // Get file icon
    let fileIcon = '📄';
    if (isImage) fileIcon = '🖼️';
    else if (isVideo) fileIcon = '🎬';
    
    card.innerHTML = `
        <div class="file-card-header">
            <div class="file-name">${fileIcon} ${file.file_name || 'Unnamed File'}</div>
            <span class="file-badge ${chainBadgeClass}">${chainName}</span>
        </div>
        ${previewHtml}
        <div class="file-info-grid">
            <div class="file-info-item">
                <strong>Size:</strong> ${fileSize}
            </div>
            <div class="file-info-item">
                <strong>Type:</strong> ${file.content_type || 'Unknown'}
            </div>
            <div class="file-info-item">
                <strong>Block:</strong> ${file.block_height.toLocaleString()}
            </div>
            <div class="file-info-item">
                <strong>Operation:</strong> ${file.operation}
            </div>
        </div>
        <div style="margin-top: 10px;">
            <div class="file-info-item" style="word-break: break-all;">
                <strong>Path:</strong> ${file.path}
            </div>
            <div class="file-info-item" style="word-break: break-all;">
                <strong>PIN ID:</strong> <span style="font-family: monospace; font-size: 12px;">${file.pin_id}</span>
            </div>
            <div class="file-info-item">
                <strong>Created:</strong> ${createdAt}
            </div>
        </div>
        <div class="file-actions">
            ${isVideo ? `
            <button onclick="playVideo('${contentUrl}', '${file.file_name || 'Video'}')" class="btn btn-primary btn-small" style="background: #dc3545;">
                ▶️ Play
            </button>
            ` : ''}
            <button onclick="window.open('${contentUrl}', '_blank')" class="btn btn-primary btn-small">
                📥 Download
            </button>
            <button onclick="window.open('${pinUrl}', '_blank')" class="btn btn-primary btn-small">
                🔗 View Pin
            </button>
            <button onclick="window.open('${txUrl}', '_blank')" class="btn btn-primary btn-small">
                📝 View TX
            </button>
            <button onclick="copyToClipboard('${file.pin_id}')" class="btn btn-primary btn-small">
                📋 Copy PIN ID
            </button>
        </div>
    `;
    
    return card;
}

// Format file size
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

// Copy to clipboard
function copyToClipboard(text) {
    // Try modern clipboard API first (requires HTTPS or localhost)
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
            showNotification('Copied to clipboard!', 'success');
        }).catch(err => {
            console.error('Failed to copy:', err);
            // Fallback to legacy method
            fallbackCopyToClipboard(text);
        });
    } else {
        // Use fallback method for non-HTTPS environments
        fallbackCopyToClipboard(text);
    }
}

// Fallback copy method for non-HTTPS environments
function fallbackCopyToClipboard(text) {
    // Create a temporary textarea
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.top = '-9999px';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);
    
    try {
        // Select and copy
        textarea.select();
        textarea.setSelectionRange(0, text.length);
        const successful = document.execCommand('copy');
        
        if (successful) {
            showNotification('Copied to clipboard!', 'success');
        } else {
            showNotification('Failed to copy', 'error');
        }
    } catch (err) {
        console.error('Failed to copy:', err);
        showNotification('Failed to copy: ' + err.message, 'error');
    } finally {
        document.body.removeChild(textarea);
    }
}

// Show notification
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    
    let icon = '💡';
    if (type === 'success') icon = '✅';
    if (type === 'error') icon = '❌';
    if (type === 'warning') icon = '⚠️';
    
    notification.innerHTML = `
        <span class="notification-icon">${icon}</span>
        <span class="notification-message">${message}</span>
        <button class="notification-close" onclick="this.parentElement.remove()">×</button>
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('notification-fade-out');
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 300);
    }, 3000);
}

// Listen for wallet monitoring
let walletCheckInterval = null;

function startWalletMonitoring() {
    const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const isInApp = window.navigator.standalone || window.matchMedia('(display-mode: standalone)').matches;
    
    if (isMobile || isInApp) {
        walletCheckInterval = setInterval(() => {
            if (typeof window.metaidwallet !== 'undefined' && !window.detectedWallet) {
                clearInterval(walletCheckInterval);
                walletCheckInterval = null;
                
                const walletObject = detectWallet();
                if (walletObject) {
                    handleWalletDetected(walletObject);
                }
            }
        }, 500);
        
        setTimeout(() => {
            if (walletCheckInterval) {
                clearInterval(walletCheckInterval);
                walletCheckInterval = null;
            }
        }, 10000);
    }
}

window.addEventListener('load', () => {
    setTimeout(startWalletMonitoring, 1000);
});

// Video playback functions
let videoModalInitialized = false;

function initVideoModal() {
    if (videoModalInitialized) return;
    
    const modal = document.getElementById('videoModal');
    if (!modal) return;
    
    // Close modal when clicking outside video
    modal.addEventListener('click', function(e) {
        if (e.target === modal) {
            closeVideoModal();
        }
    });
    
    // Close modal with Escape key
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && modal.classList.contains('show')) {
            closeVideoModal();
        }
    });
    
    videoModalInitialized = true;
}

function playVideo(videoUrl, videoName) {
    const modal = document.getElementById('videoModal');
    const videoPlayer = document.getElementById('videoPlayer');
    
    if (!modal || !videoPlayer) {
        // Fallback: open in new window if modal doesn't exist
        window.open(videoUrl, '_blank');
        return;
    }
    
    // Initialize modal event listeners if not already done
    initVideoModal();
    
    videoPlayer.src = videoUrl;
    videoPlayer.load();
    modal.classList.add('show');
}

function closeVideoModal() {
    const modal = document.getElementById('videoModal');
    const videoPlayer = document.getElementById('videoPlayer');
    
    if (modal && videoPlayer) {
        modal.classList.remove('show');
        videoPlayer.pause();
        videoPlayer.src = '';
    }
}

// Handle video preview image load error
function handleVideoPreviewError(containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;
    
    const img = container.querySelector('.video-preview-image');
    const placeholder = container.querySelector('.video-preview-placeholder');
    
    if (img) img.style.display = 'none';
    if (placeholder) placeholder.style.display = 'flex';
}

// Initialize video modal on page load
window.addEventListener('load', () => {
    setTimeout(() => {
        initVideoModal();
    }, 500);
});

// ============================================================
// User List Functions
// ============================================================

// Load user list
async function loadUserList() {
    userListContainer.innerHTML = '<div class="loading"><div class="spinner"></div><p style="margin-top: 10px;">Loading users...</p></div>';
    
    try {
        usersCursor = 0;
        hasMoreUsers = true;
        
        const response = await fetch(`${API_BASE}/api/v1/users?cursor=0&size=20`);
        const data = await response.json();
        
        if (data.code === 0) {
            const users = data.data.users || [];
            const nextCursor = data.data.next_cursor || 0;
            const total = data.data.total || 0;
            hasMoreUsers = data.data.has_more || false;
            
            // Update total users count display
            const totalUsersCountEl = document.getElementById('totalUsersCount');
            if (totalUsersCountEl) {
                totalUsersCountEl.textContent = `(Total: ${total.toLocaleString()})`;
            }
            
            if (users.length === 0) {
                userListContainer.innerHTML = `
                    <div class="empty-state">
                        <div class="empty-state-icon">👥</div>
                        <p>No users found</p>
                        <p style="font-size: 14px; margin-top: 10px;">Users will appear here as they register on MetaID</p>
                    </div>
                `;
                loadMoreUsersBtn.classList.add('hidden');
            } else {
                usersCursor = nextCursor;
                renderUsers(users, true);
                
                if (hasMoreUsers) {
                    loadMoreUsersBtn.classList.remove('hidden');
                } else {
                    loadMoreUsersBtn.classList.add('hidden');
                }
            }
        } else {
            throw new Error(data.message || 'Failed to load users');
        }
    } catch (error) {
        console.error('Failed to load user list:', error);
        userListContainer.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load users</p>
                <p style="font-size: 14px; margin-top: 10px;">${error.message}</p>
            </div>
        `;
        showNotification('Failed to load users: ' + error.message, 'error');
    }
}

// Load more users
async function loadMoreUsers() {
    if (isLoadingUsers || !hasMoreUsers) {
        return;
    }
    
    isLoadingUsers = true;
    loadMoreUsersBtn.disabled = true;
    loadMoreUsersBtn.textContent = 'Loading...';
    
    try {
        const response = await fetch(`${API_BASE}/api/v1/users?cursor=${usersCursor}&size=20`);
        const data = await response.json();
        
        if (data.code === 0) {
            const users = data.data.users || [];
            const nextCursor = data.data.next_cursor || 0;
            hasMoreUsers = data.data.has_more || false;
            
            if (users.length > 0) {
                usersCursor = nextCursor;
                renderUsers(users, false);
            }
            
            if (!hasMoreUsers) {
                loadMoreUsersBtn.classList.add('hidden');
            }
        } else {
            throw new Error(data.message || 'Failed to load more users');
        }
    } catch (error) {
        console.error('Failed to load more users:', error);
        showNotification('Failed to load more users: ' + error.message, 'error');
    } finally {
        isLoadingUsers = false;
        loadMoreUsersBtn.disabled = false;
        loadMoreUsersBtn.textContent = 'Load More';
    }
}

// Refresh user list
function refreshUserList() {
    loadUserList();
}

// Render users
function renderUsers(users, clearFirst) {
    if (clearFirst) {
        userListContainer.innerHTML = '';
    }
    
    users.forEach(user => {
        const userCard = createUserCard(user);
        userListContainer.appendChild(userCard);
    });
}

// Create user card
function createUserCard(user) {
    const card = document.createElement('div');
    card.className = 'user-card';
    
    const chainBadgeClass = user.chainName === 'btc' ? 'badge-btc' : 'badge-mvc';
    const chainName = (user.chainName || 'mvc').toUpperCase();
    
    // Avatar URL
    const avatarUrl = user.metaId 
        ? `${API_BASE}/api/v1/users/metaid/${user.metaId}/avatar`
        : '';
    
    // User name
    const userName = user.name || 'Anonymous';
    
    // Short MetaID for display
    const shortMetaId = user.metaId 
        ? `${user.metaId.substring(0, 8)}...${user.metaId.substring(user.metaId.length - 8)}`
        : '';
    
    const timestamp = user.timestamp ? new Date(user.timestamp).toLocaleString() : '-';
    
    card.innerHTML = `
        <div class="user-avatar-large">
            ${avatarUrl ? `
                <img src="${avatarUrl}" 
                     alt="${userName}" 
                     onerror="this.style.display='none'; this.nextElementSibling.style.display='flex';">
                <div class="user-avatar-placeholder" style="display: none;">👤</div>
            ` : `
                <div class="user-avatar-placeholder">👤</div>
            `}
        </div>
        <div class="user-info-content">
            <div class="user-name">${userName}</div>
            <div class="user-metaid" title="${user.metaId || ''}">
                MetaID: ${shortMetaId}
            </div>
            <div class="user-meta-info">
                <span class="file-badge ${chainBadgeClass}">${chainName}</span>
                <span>📅 ${timestamp}</span>
                ${user.blockHeight ? `<span>📦 Block ${user.blockHeight.toLocaleString()}</span>` : ''}
            </div>
        </div>
        <div class="user-actions">
            <button class="btn btn-primary btn-small copy-metaid-btn" data-metaid="${user.metaId || ''}" title="Copy MetaID">
                📋 Copy MetaID
            </button>
            ${user.address ? `
            <button class="btn btn-primary btn-small copy-address-btn" data-address="${user.address}" title="Copy Address">
                📋 Copy Address
            </button>
            ` : ''}
        </div>
    `;
    
    // Add event listeners for copy buttons
    const copyMetaIdBtn = card.querySelector('.copy-metaid-btn');
    if (copyMetaIdBtn) {
        copyMetaIdBtn.addEventListener('click', function() {
            const metaId = this.getAttribute('data-metaid');
            if (metaId) {
                copyToClipboard(metaId);
            }
        });
    }
    
    const copyAddressBtn = card.querySelector('.copy-address-btn');
    if (copyAddressBtn) {
        copyAddressBtn.addEventListener('click', function() {
            const address = this.getAttribute('data-address');
            if (address) {
                copyToClipboard(address);
            }
        });
    }
    
    return card;
}

