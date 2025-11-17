// API base URL
const API_BASE = window.location.origin + window.location.pathname.replace(/indexer\.html$/, '').replace(/\/$/, '');

// Global state
let walletConnected = false;
let currentAddress = null;
let currentMetaID = null;
let filesCursor = 0;
let hasMoreFiles = true;
let isLoadingFiles = false;

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

// Status elements
const currentBlockEl = document.getElementById('currentBlock');
const latestBlockEl = document.getElementById('latestBlock');
const totalFilesEl = document.getElementById('totalFiles');
const syncProgressEl = document.getElementById('syncProgress');

// Initialization
window.addEventListener('load', () => {
    console.log('🚀 Indexer page loaded');
    initWalletCheck();
    initEventListeners();
    loadIndexerStatus();
    
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
        const address = account.address || account.mvcAddress || account.btcAddress;
        
        if (account && address) {
            currentAddress = address;
            walletConnected = true;
            
            walletStatus.textContent = 'Connected';
            walletStatus.style.color = '#28a745';
            
            addressText.textContent = currentAddress;
            
            // Calculate MetaID
            currentMetaID = await calculateMetaID(currentAddress);
            metaidText.textContent = currentMetaID;
            
            walletAddress.classList.remove('hidden');
            walletAlert.classList.add('hidden');
            
            connectBtn.classList.add('hidden');
            disconnectBtn.classList.remove('hidden');
            
            showNotification('Wallet connected successfully!', 'success');
            
            // Load user avatar
            loadUserAvatar();
            
            // Load user files
            loadUserFiles();
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
    try {
        const encoder = new TextEncoder();
        const data = encoder.encode(address);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        return hashHex;
    } catch (error) {
        console.error('Failed to calculate MetaID:', error);
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
        // Get sync status from API
        const statusResponse = await fetch(`${API_BASE}/api/v1/status`);
        const statusData = await statusResponse.json();
        
        if (statusData.code === 0 && statusData.data) {
            const status = statusData.data;
            
            // Update current sync height
            currentBlockEl.textContent = status.current_sync_height.toLocaleString();
            
            // Update latest block height from node
            if (status.latest_block_height && status.latest_block_height > 0) {
                latestBlockEl.textContent = status.latest_block_height.toLocaleString();
                
                // Calculate sync progress
                if (status.current_sync_height >= status.latest_block_height) {
                    syncProgressEl.textContent = '✅ Synced';
                } else {
                    const progress = ((status.current_sync_height / status.latest_block_height) * 100).toFixed(2);
                    const behind = status.latest_block_height - status.current_sync_height;
                    syncProgressEl.textContent = `⏳ Syncing (${progress}%, ${behind.toLocaleString()} blocks behind)`;
                }
            } else {
                latestBlockEl.textContent = '-';
                syncProgressEl.textContent = '✅ Running';
            }
            
            console.log('✅ Indexer status loaded:', status);
            console.log('📊 Current sync height:', status.current_sync_height);
            console.log('📊 Latest block height:', status.latest_block_height);
            
            // Get statistics (total files count)
            const statsResponse = await fetch(`${API_BASE}/api/v1/stats`);
            const statsData = await statsResponse.json();
            
            if (statsData.code === 0 && statsData.data) {
                totalFilesEl.textContent = statsData.data.total_files.toLocaleString();
                console.log('📊 Total files:', statsData.data.total_files);
            } else {
                totalFilesEl.textContent = '-';
            }
        } else {
            throw new Error(statusData.message || 'Failed to load status');
        }
    } catch (error) {
        console.error('Failed to load indexer status:', error);
        currentBlockEl.textContent = 'Error';
        latestBlockEl.textContent = 'Error';
        totalFilesEl.textContent = 'Error';
        syncProgressEl.textContent = 'Error';
    }
}

// Load user files
async function loadUserFiles() {
    if (!currentMetaID) {
        console.error('MetaID not available');
        return;
    }
    
    fileListSection.classList.remove('hidden');
    fileListContainer.innerHTML = '<div class="loading"><div class="spinner"></div><p style="margin-top: 10px;">Loading your files...</p></div>';
    
    try {
        filesCursor = 0;
        hasMoreFiles = true;
        
        const response = await fetch(`${API_BASE}/api/v1/files/metaid/${currentMetaID}?cursor=0&size=20`);
        const data = await response.json();
        
        if (data.code === 0) {
            const files = data.data.files || [];
            const nextCursor = data.data.next_cursor || 0;
            hasMoreFiles = data.data.has_more || false;
            
            if (files.length === 0) {
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
                renderFiles(files, true);
                
                if (hasMoreFiles) {
                    loadMoreBtn.classList.remove('hidden');
                } else {
                    loadMoreBtn.classList.add('hidden');
                }
            }
        } else {
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
    const createdAt = new Date(file.created_at).toLocaleString();
    
    let pinIds = file.pin_id.split('i');
    let txId = pinIds[0];
    
    // Build view links
    const txUrl = `https://www.mvcscan.com/tx/${txId}`;
    const pinUrl = `https://man.metaid.io/pin/${file.pin_id}`;
    const contentUrl = `${API_BASE}/api/v1/files/content/${file.pin_id}`;
    
    // Check if file is an image
    const isImage = file.file_type === 'image' || (file.content_type && file.content_type.startsWith('image/'));
    
    // Build image preview HTML
    let imagePreviewHtml = '';
    if (isImage) {
        imagePreviewHtml = `
            <div style="margin: 15px 0; text-align: center; background: #f0f0f0; border-radius: 8px; padding: 10px;">
                <img src="${contentUrl}" 
                     alt="${file.file_name || 'Image'}" 
                     style="max-width: 100%; max-height: 300px; border-radius: 8px; cursor: pointer;"
                     onclick="window.open('${contentUrl}', '_blank')"
                     onerror="this.parentElement.innerHTML='<p style=\\'color: #999; padding: 20px;\\'>Failed to load image preview</p>'">
            </div>
        `;
    }
    
    card.innerHTML = `
        <div class="file-card-header">
            <div class="file-name">${isImage ? '🖼️' : '📄'} ${file.file_name || 'Unnamed File'}</div>
            <span class="file-badge ${chainBadgeClass}">${chainName}</span>
        </div>
        ${imagePreviewHtml}
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
    navigator.clipboard.writeText(text).then(() => {
        showNotification('Copied to clipboard!', 'success');
    }).catch(err => {
        console.error('Failed to copy:', err);
        showNotification('Failed to copy', 'error');
    });
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

