// API base URL - Get the current path to support reverse proxy with subpath
const API_BASE = window.location.origin + window.location.pathname.replace(/\/$/, '');

// Global state
let selectedFile = null;
let selectedFileExtension = ''; // Store detected file extension
let walletConnected = false;
let currentAddress = null;
let maxFileSize = 10485760; // Default 10MB, will be fetched from server
let swaggerBaseUrl = ''; // Swagger base URL from server config

// Helper function to get wallet object
function getWallet() {
    return window.detectedWallet || window.metaidwallet;
}

// Helper function to build contentType
// For text/ types, don't add ;binary suffix
function buildContentType(file) {
    let contentType = file.type || 'application/octet-stream';
    
    // Check if it's a text type
    const isTextType = contentType.startsWith('text/') || 
                      contentType === 'application/json' ||
                      contentType === 'application/javascript' ||
                      contentType === 'application/xml';
    
    // Only add ;binary for non-text types
    if (!isTextType && !contentType.includes(';binary')) {
        contentType = contentType + ';binary';
    }
    
    return contentType;
}

// DOM elements
const connectBtn = document.getElementById('connectBtn');
const disconnectBtn = document.getElementById('disconnectBtn');
const dropZone = document.getElementById('dropZone');
const fileInput = document.getElementById('fileInput');
const fileInfo = document.getElementById('fileInfo');
const uploadBtn = document.getElementById('uploadBtn');
const walletStatus = document.getElementById('walletStatus');
const walletAddress = document.getElementById('walletAddress');
const addressText = document.getElementById('addressText');
const walletAlert = document.getElementById('walletAlert');
const progress = document.getElementById('progress');
const progressFill = document.getElementById('progressFill');
const progressText = document.getElementById('progressText');
const logContainer = document.getElementById('logContainer');
const maxFileSizeText = document.getElementById('maxFileSizeText');

// Buzz related DOM elements
const buzzSection = document.getElementById('buzzSection');
const buzzContent = document.getElementById('buzzContent');
const buzzHost = document.getElementById('buzzHost');
const sendBuzzBtn = document.getElementById('sendBuzzBtn');

// Initialization
window.addEventListener('load', () => {
    console.log('🚀 Page loaded, starting initialization...');
    console.log('🌐 Current URL info:', {
        origin: window.location.origin,
        pathname: window.location.pathname,
        href: window.location.href,
        API_BASE: API_BASE
    });
    loadConfig(); // Load config
    initWalletCheck();
    initDragDrop();
    initEventListeners();
    updateUploadButton(); // Initialize upload button state
    updateBuzzButton(); // Initialize buzz button state
    
    // Test buzz section display
    console.log('🧪 Testing buzz section display...');
    console.log('🧪 buzzSection element:', buzzSection);
    if (buzzSection) {
        console.log('🧪 buzzSection found, testing display...');
        buzzSection.classList.remove('hidden');
        console.log('🧪 buzzSection classes after test:', buzzSection.className);
        // Hide it again for now
        buzzSection.classList.add('hidden');
    }
    
    addLog('Page initialization complete', 'info');
});

// Listen for wallet injection in mobile webview
let walletCheckInterval = null;

// Start monitoring for wallet injection
function startWalletMonitoring() {
    const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const isInApp = window.navigator.standalone || window.matchMedia('(display-mode: standalone)').matches;
    
    if (isMobile || isInApp) {
        console.log('🔍 Starting wallet monitoring for mobile webview...');
        
        walletCheckInterval = setInterval(() => {
            if (typeof window.metaidwallet !== 'undefined' && !window.detectedWallet) {
                console.log('✅ Wallet detected during monitoring!');
                clearInterval(walletCheckInterval);
                walletCheckInterval = null;
                
                const walletObject = detectWallet();
                if (walletObject) {
                    handleWalletDetected(walletObject);
                }
            }
        }, 500); // Check every 500ms
        
        // Stop monitoring after 10 seconds
        setTimeout(() => {
            if (walletCheckInterval) {
                console.log('⏰ Wallet monitoring timeout');
                clearInterval(walletCheckInterval);
                walletCheckInterval = null;
            }
        }, 10000);
    }
}

// Start monitoring after page load
window.addEventListener('load', () => {
    // Start monitoring after a short delay to allow for wallet injection
    setTimeout(startWalletMonitoring, 1000);
});

// Load configuration
async function loadConfig() {
    try {
        addLog('Loading service configuration...', 'info');
        const response = await fetch(`${API_BASE}/api/v1/config`);
        const result = await response.json();
        
        if (result.code === 0 && result.data) {
            maxFileSize = result.data.maxFileSize;
            swaggerBaseUrl = result.data.swaggerBaseUrl || '';
            
            const sizeText = formatFileSize(maxFileSize);
            maxFileSizeText.textContent = sizeText;
            addLog(`File size limit: ${sizeText}`, 'success');
            console.log('✅ Configuration loaded successfully:', result.data);
            
            // Update Swagger link if baseUrl is configured
            updateSwaggerLink();
        } else {
            throw new Error(result.message || 'Failed to load configuration');
        }
    } catch (error) {
        console.error('❌ Failed to load configuration:', error);
        maxFileSizeText.textContent = formatFileSize(maxFileSize) + ' (default)';
        addLog('Failed to load configuration, using default values', 'error');
    }
}

// Update Swagger documentation link
function updateSwaggerLink() {
    console.log('🔍 Updating Swagger link, swaggerBaseUrl:', swaggerBaseUrl);
    
    const swaggerLink = document.getElementById('swaggerLink');
    console.log('🔍 swaggerLink element:', swaggerLink);
    
    if (!swaggerLink) {
        console.warn('⚠️ swaggerLink element not found!');
        return;
    }
    
    if (swaggerBaseUrl) {
        // Build Swagger URL based on baseUrl
        // swaggerBaseUrl format examples:
        // - "localhost:7282" -> use relative path /swagger/index.html
        // - "file.metaid.io/metafile-uploader" -> https://file.metaid.io/metafile-uploader/swagger/index.html
        
        let swaggerUrl = 'swagger/index.html'; // default (relative path)
        
        if (swaggerBaseUrl.includes('/')) {
            // Contains path, likely a proxied URL
            const protocol = window.location.protocol;
            swaggerUrl = `${protocol}//${swaggerBaseUrl}/swagger/index.html`;
            console.log(`📝 URL with path detected, protocol: ${protocol}, result: ${swaggerUrl}`);
        } else if (!swaggerBaseUrl.includes('localhost') && !swaggerBaseUrl.includes('127.0.0.1')) {
            // External domain without path
            swaggerUrl = `https://${swaggerBaseUrl}/swagger/index.html`;
            console.log(`📝 External domain detected, result: ${swaggerUrl}`);
        } else {
            console.log(`📝 Local development, using relative path: ${swaggerUrl}`);
        }
        
        swaggerLink.href = swaggerUrl;
        console.log('📚 Swagger URL updated:', swaggerUrl);
    } else {
        console.log('⚠️ swaggerBaseUrl is empty, keeping default link');
    }
}

// Check Metalet wallet with retry mechanism for mobile webview
function initWalletCheck() {
    // Detect mobile environment
    const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const isInApp = window.navigator.standalone || window.matchMedia('(display-mode: standalone)').matches;
    
    console.log('🔍 Environment detection:', {
        userAgent: navigator.userAgent,
        isMobile: isMobile,
        isInApp: isInApp,
        windowKeys: Object.keys(window).filter(k => k.toLowerCase().includes('meta') || k.toLowerCase().includes('wallet'))
    });
    
    // Try to detect wallet immediately
    const walletObject = detectWallet();
    
    if (walletObject) {
        handleWalletDetected(walletObject);
    } else {
        // If not detected and it's mobile, try with retry mechanism
        if (isMobile || isInApp) {
            addLog('Mobile environment detected, retrying wallet detection...', 'info');
            retryWalletDetection(3, 1000); // Retry 3 times with 1 second intervals
        } else {
            addLog('Metalet wallet extension not detected', 'error');
            walletAlert.classList.remove('hidden');
        }
    }
}

// Detect wallet object
function detectWallet() {
    // Check for standard wallet object (works for both desktop and mobile)
    if (typeof window.metaidwallet !== 'undefined') {
        return {
            object: window.metaidwallet,
            type: 'Metalet Wallet'
        };
    }
    
    // Check for alternative wallet objects
    if (typeof window.MetaletWallet !== 'undefined') {
        return {
            object: window.MetaletWallet,
            type: 'MetaletWallet'
        };
    }
    
    if (typeof window.metalet !== 'undefined') {
        return {
            object: window.metalet,
            type: 'metalet'
        };
    }
    
    if (window.Metaid && typeof window.Metaid.MetaletWalletForMvc !== 'undefined') {
        return {
            object: window.Metaid,
            type: 'MetaID SDK'
        };
    }
    
    return null;
}

// Handle wallet detection success
function handleWalletDetected(walletInfo) {
    const { object: walletObject, type: walletType } = walletInfo;
    
    addLog(`Metalet wallet detected: ${walletType}`, 'info');
    
    // Store wallet object globally for use in other functions
    window.detectedWallet = walletObject;
    window.walletType = walletType;
    
    // Check available methods
    console.log('🔍 Metalet wallet available methods:', Object.keys(walletObject));
    
    // Check signing methods
    const signMethods = [];
    if (typeof walletObject.pay === 'function') {
        signMethods.push('pay (recommended)');
    }
    if (typeof walletObject.signTransactions === 'function') {
        signMethods.push('signTransactions');
    }
    if (typeof walletObject.signTransaction === 'function') {
        signMethods.push('signTransaction');
    }
    if (typeof walletObject.signRawTransaction === 'function') {
        signMethods.push('signRawTransaction');
    }
    if (walletObject.btc && typeof walletObject.btc.signTransaction === 'function') {
        signMethods.push('btc.signTransaction');
    }
    
    if (signMethods.length > 0) {
        console.log('✅ Supported signing methods:', signMethods);
        addLog(`Supported signing methods: ${signMethods.join(', ')}`, 'info');
    } else {
        console.warn('⚠️ No standard signing method detected');
    }
    
    // Hide wallet alert
    walletAlert.classList.add('hidden');
}

// Retry wallet detection for mobile webview
function retryWalletDetection(attempts, delay) {
    if (attempts <= 0) {
        addLog('Mobile Metalet app detected but wallet interface not found after retries', 'warning');
        addLog('Please ensure you are using the latest version of Metalet app', 'info');
        addLog('Try refreshing the page or restarting the Metalet app', 'info');
        walletAlert.classList.add('hidden');
        return;
    }
    
    addLog(`Retrying wallet detection... (${attempts} attempts left)`, 'info');
    
    setTimeout(() => {
        const walletObject = detectWallet();
        
        if (walletObject) {
            handleWalletDetected(walletObject);
        } else {
            retryWalletDetection(attempts - 1, delay);
        }
    }, delay);
}

// Initialize drag and drop upload
function initDragDrop() {
    console.log('📁 Initializing drag and drop upload...');
    console.log('dropZone:', dropZone);
    console.log('fileInput:', fileInput);
    
    // Click to select file
    dropZone.addEventListener('click', () => {
        console.log('🔘 Drop zone clicked');
        fileInput.click();
    });

    // File selection
    fileInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            handleFile(e.target.files[0]);
        }
    });

    // Drag events
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('dragover');
    });

    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('dragover');
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('dragover');
        
        if (e.dataTransfer.files.length > 0) {
            handleFile(e.dataTransfer.files[0]);
        }
    });
}

// Initialize event listeners
function initEventListeners() {
    console.log('📌 Initializing event listeners...');
    console.log('connectBtn:', connectBtn);
    console.log('disconnectBtn:', disconnectBtn);
    console.log('uploadBtn:', uploadBtn);
    
    // Connect wallet button
    if (connectBtn) {
        connectBtn.addEventListener('click', () => {
            console.log('🔘 Connect wallet button clicked');
            connectWallet();
        });
    } else {
        console.error('❌ connectBtn element not found!');
    }
    
    // Disconnect wallet button
    if (disconnectBtn) {
        disconnectBtn.addEventListener('click', () => {
            console.log('🔘 Disconnect wallet button clicked');
            disconnectWallet();
        });
    } else {
        console.error('❌ disconnectBtn element not found!');
    }
    
    // Upload button
    if (uploadBtn) {
        // uploadBtn.addEventListener('click', startUpload);
        uploadBtn.addEventListener('click', startUpload3);
    } else {
        console.error('❌ uploadBtn element not found!');
    }
    
    // Refresh balance button
    const refreshBalanceBtn = document.getElementById('refreshBalanceBtn');
    if (refreshBalanceBtn) {
        refreshBalanceBtn.addEventListener('click', () => {
            console.log('🔄 Refresh balance button clicked');
            if (walletConnected) {
                fetchAndDisplayBalance();
            } else {
                showNotification('Please connect wallet first', 'warning');
            }
        });
    }
    
    // Send buzz button
    if (sendBuzzBtn) {
        sendBuzzBtn.addEventListener('click', () => {
            console.log('📝 Send buzz button clicked');
            sendBuzz();
        });
    }
    
    // Buzz content input listener (removed - no longer needed for button state)
    
    // ShowNow button for file host input
    const showNowBtn = document.getElementById('showNowBtn');
    if (showNowBtn) {
        showNowBtn.addEventListener('click', () => {
            console.log('📱 ShowNow button clicked');
            const fileHostInput = document.getElementById('fileHostInput');
            if (fileHostInput) {
                const showNowAddress = 'bc1p20k3x2c4mglfxr5wa5sgtgechwstpld80kru2cg4gmm4urvuaqqsvapxu0';
                fileHostInput.value = showNowAddress;
                addLog(`📱 ShowNow address filled: ${showNowAddress}`, 'info');
                showNotification('ShowNow address filled', 'success');
            }
        });
    }
    
    // ShowNow button for buzz host input
    const buzzShowNowBtn = document.getElementById('buzzShowNowBtn');
    if (buzzShowNowBtn) {
        buzzShowNowBtn.addEventListener('click', () => {
            console.log('📱 Buzz ShowNow button clicked');
            const buzzHostInput = document.getElementById('buzzHost');
            if (buzzHostInput) {
                const showNowAddress = 'bc1p20k3x2c4mglfxr5wa5sgtgechwstpld80kru2cg4gmm4urvuaqqsvapxu0';
                buzzHostInput.value = showNowAddress;
                addLog(`📱 Buzz ShowNow address filled: ${showNowAddress}`, 'info');
                showNotification('Buzz ShowNow address filled', 'success');
            }
        });
    }
}

// Handle file
function handleFile(file) {
    // Validate file size
    if (file.size > maxFileSize) {
        const maxSizeText = formatFileSize(maxFileSize);
        const fileSizeText = formatFileSize(file.size);
        showNotification(`File too large! File size: ${fileSizeText}, Max limit: ${maxSizeText}`, 'error');
        addLog(`File size exceeds limit: ${fileSizeText} > ${maxSizeText}`, 'error');
        return;
    }
    
    selectedFile = file;
    
    // Build contentType (text types don't need ;binary)
    const contentType = buildContentType(file);
    
    // Detect file extension and store it globally
    const detectedExtension = getFileExtension(file);
    selectedFileExtension = detectedExtension;
    
    // Display file information
    document.getElementById('fileName').textContent = file.name;
    document.getElementById('fileSize').textContent = formatFileSize(file.size);
    document.getElementById('fileType').textContent = file.type || 'Unknown';
    document.getElementById('fileExtension').textContent = detectedExtension || '(none)';
    document.getElementById('fileContentType').textContent = contentType;
    
    fileInfo.classList.add('show');
    dropZone.classList.add('has-file');
    
    // Auto-fill path
    const pathInput = document.getElementById('pathInput');
    if (pathInput.value === '/file') {
        pathInput.value = '/file';
    }
    
    // Enable upload button
    updateUploadButton();
    
    addLog(`File selected: ${file.name} (${formatFileSize(file.size)})`, 'info');
    addLog(`📄 ContentType: ${contentType}`, 'info');
    addLog(`📎 Detected Extension: ${detectedExtension || '(none)'}`, 'info');
    showNotification(`File selected: ${file.name}`, 'success');
}

// Connect wallet
async function connectWallet() {
    console.log('🔵 Starting to connect wallet...');
    console.log('detectedWallet:', window.detectedWallet);
    console.log('walletType:', window.walletType);
    
    // Use detected wallet object or fallback to window.metaidwallet
    const wallet = getWallet();
    
    if (!wallet) {
        const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        const isInApp = window.navigator.standalone || window.matchMedia('(display-mode: standalone)').matches;
        
        if (isMobile || isInApp) {
            showNotification('Please ensure you are using the latest version of Metalet app', 'warning');
            addLog('❌ Metalet wallet not detected in mobile app', 'error');
        } else {
            showNotification('Please install Metalet wallet extension first!', 'error');
            addLog('❌ Metalet wallet not detected', 'error');
            window.open('https://www.metalet.space/', '_blank');
        }
        return;
    }

    try {
        connectBtn.disabled = true;
        connectBtn.textContent = 'Connecting...';
        
        addLog(`Requesting to connect Metalet wallet (${window.walletType || 'Unknown'})...`, 'info');
        console.log('📡 Calling wallet.connect()...');
        
        // Connect wallet
        const account = await wallet.connect();
        
        console.log('✅ Wallet response:', account);
        
        // Compatible with different wallet API versions
        // New version may return address, old version returns mvcAddress
        const address = account.address || account.mvcAddress || account.btcAddress;
        
        if (account && address) {
            currentAddress = address;
            walletConnected = true;
            
            walletStatus.textContent = 'Connected';
            walletStatus.style.color = '#28a745';
            
            // Display address
            addressText.textContent = currentAddress;
            
            // Calculate and display MetaID
            const metaId = await calculateMetaID(currentAddress);
            const metaidText = document.getElementById('metaidText');
            if (metaidText) {
                metaidText.textContent = metaId;
            }
            
            walletAddress.classList.remove('hidden');
            walletAlert.classList.add('hidden');
            
            connectBtn.textContent = '✓ Connected';
            connectBtn.classList.remove('btn-primary');
            connectBtn.classList.add('btn-success');
            connectBtn.classList.add('hidden');
            
            // show disconnect button
            disconnectBtn.classList.remove('hidden');
            
            updateUploadButton();
            updateBuzzButton();
            
            addLog(`✅ Wallet connected successfully`, 'success');
            addLog(`📍 Address: ${currentAddress}`, 'info');
            addLog(`🔑 MetaID: ${metaId}`, 'info');
            showNotification('Wallet connected successfully!', 'success');
            
            // fetch and display balance
            await fetchAndDisplayBalance();
        } else {
            console.error('❌ wallet returned data format error:', account);
            throw new Error('wallet returned data format error,valid address field not found');
        }
    } catch (error) {
        console.error('❌ failed to connect wallet:', error);
        console.error('error details:', {
            name: error.name,
            message: error.message,
            stack: error.stack
        });
        
        // handle user cancellation
        if (error.message && (
            error.message.includes('User canceled') || 
            error.message.includes('user cancelled') ||
            error.message.includes('User rejected') ||
            error.message.includes('canceled')
        )) {
            addLog('⚠️ user cancelled connection', 'error');
            showNotification('wallet connection cancelled', 'warning');
        } else {
            addLog(`❌ failed to connect wallet: ${error.message}`, 'error');
            showNotification('failed to connect wallet: ' + error.message, 'error');
        }
        
        connectBtn.disabled = false;
        connectBtn.textContent = 'Connect Metalet Wallet';
    }
}

// Disconnect walletconnect
function disconnectWallet() {
    console.log('🔌 disconnect wallet');
    
    // reset status
    walletConnected = false;
    currentAddress = null;
    
    // update UI
    walletStatus.textContent = 'Not Connected';
    walletStatus.style.color = '#999';
    walletAddress.classList.add('hidden');
    
    // hide balance info
    const balanceInfo = document.getElementById('balanceInfo');
    if (balanceInfo) {
        balanceInfo.classList.add('hidden');
    }
    
    // show connect button, hide disconnect button
    connectBtn.textContent = 'Connect Metalet Wallet';
    connectBtn.classList.remove('btn-success', 'hidden');
    connectBtn.classList.add('btn-primary');
    connectBtn.disabled = false;
    
    disconnectBtn.classList.add('hidden');
    
    // Update upload button state
    updateUploadButton();
    updateBuzzButton();
    
    addLog('🔌 Wallet disconnected', 'info');
    showNotification('Wallet disconnected', 'info');
}

// Update upload button state
function updateUploadButton() {
    const canUpload = selectedFile && walletConnected;
    uploadBtn.disabled = !canUpload;
    
    // update button text hint
    if (!walletConnected && !selectedFile) {
        uploadBtn.textContent = '🚀 Please connect wallet firstand select file';
    } else if (!walletConnected) {
        uploadBtn.textContent = '🚀 Please connect wallet first';
    } else if (!selectedFile) {
        uploadBtn.textContent = '🚀 Please select file first';
    } else {
        uploadBtn.textContent = '🚀 Start Upload to Chain';
    }
    
    console.log('🔄 update upload button status:', {
        walletConnected,
        selectedFile: !!selectedFile,
        canUpload
    });
}

// fetch and display balance
async function fetchAndDisplayBalance() {
    try {
        addLog('Fetching wallet balance...', 'info');
        console.log('💰 call wallet.getBalance()...');
        
        const wallet = getWallet();
        if (!wallet) {
            throw new Error('Wallet not available');
        }
        
        const balance = await wallet.getBalance();
        console.log('✅ balance return result:', balance);
        
        // display balance info
        const balanceInfo = document.getElementById('balanceInfo');
        if (balanceInfo) {
            // compatible with different return formats
            let totalBalance = balance.total || balance.confirmed || balance.balance || 0;
            let confirmedBalance = balance.confirmed || balance.total || 0;
            let unconfirmedBalance = balance.unconfirmed || 0;
            
            document.getElementById('totalBalance').textContent = formatSatoshis(totalBalance);
            document.getElementById('confirmedBalance').textContent = formatSatoshis(confirmedBalance);
            document.getElementById('unconfirmedBalance').textContent = formatSatoshis(unconfirmedBalance);
            
            balanceInfo.classList.remove('hidden');
            
            addLog(`💰 Balance: ${formatSatoshis(totalBalance)} SPACE`, 'success');
            
            // check if balance is sufficient
            if (totalBalance < 1000) {
                showNotification('⚠️ balance insufficient.*satoshis, may not complete upload to chain', 'warning');
                addLog('⚠️ balance insufficient, recommend recharge', 'error');
            }
        }
    } catch (error) {
        console.error('❌ failed to get balance:', error);
        addLog(`⚠️ failed to get balance: ${error.message}`, 'error');
        // dont block flow, just warning
    }
}

// format satoshis to readable format
function formatSatoshis(satoshis) {
    const space = satoshis / 100000000;
    return `${space.toFixed(8)} SPACE (${satoshis.toLocaleString()} sats)`;
}

// calculate MetaID（address SHA256 hash）
async function calculateMetaID(address) {
    try {
        // convert address string to Uint8Array
        const encoder = new TextEncoder();
        const data = encoder.encode(address);
        
        // calculate SHA256
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        
        // convert to hex string
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        
        console.log(`🔑 MetaID calculate: SHA256("${address}") = ${hashHex}`);
        return hashHex;
    } catch (error) {
        console.error('❌ calculate MetaID failed:', error);
        return 'calculatefailed';
    }
}

// Start uploadflow
async function startUpload() {
    // validate wallet connection
    if (!walletConnected) {
        showNotification('⚠️ please connect Metalet Wallet', 'warning');
        addLog('❌ walletNot Connected', 'error');
        return;
    }
    
    // validate file selection
    if (!selectedFile) {
        showNotification('⚠️ please select file to upload to chain', 'warning');
        addLog('❌ no file selected', 'error');
        return;
    }
    
    console.log('✅ validation passed: wallet connected, file selected');
    addLog('Start Upload to Chainflow...', 'info');

    try {
        uploadBtn.disabled = true;
        uploadBtn.textContent = 'Uploading to chain...';
        progress.classList.add('show');
        
        showNotification('Start Upload to Chainflow...', 'info');
        
        // step 1: pre-upload
        updateProgress(10, 'step 1/4: build transaction...');
        const preUploadResult = await preUpload();
        
        // check if file is alreadyupload to chain successful
        if (preUploadResult.status === 'success') {
            updateProgress(100, 'file already exists!');
            addLog(`✅ this file has been uploaded to chain successfully!`, 'success');
            
            // show link
            showUploadSuccessLinks(preUploadResult.txId, preUploadResult.pinId);
            
            // show buzz section for existing file
            console.log('📝 File already exists, showing buzz section with pinId:', preUploadResult.pinId);
            showBuzzSection(preUploadResult.pinId);
            
            uploadBtn.disabled = false;
            uploadBtn.textContent = '🚀 Start Upload to Chain';
            progress.classList.remove('show');
            return;
        }
        
        // step 2: get UTXO andverify balance
        updateProgress(30, 'step 2/4: verify balance...');
        const utxoData = await prepareUTXO(preUploadResult);
        
        // step 3: sign transaction
        updateProgress(50, 'step 3/4: sign transaction...');
        showNotification('please confirm signature in wallet...', 'info');
        const signedTx = await signTransaction(preUploadResult.preTxRaw, utxoData);
        
        // step 4: commit upload to chain
        updateProgress(80, 'step 4/4: broadcast transaction...');
        const commitResult = await commitUpload(preUploadResult.fileId, signedTx);
        
        // completed
        updateProgress(100, 'upload to chain completed!');
        addLog(`✅ fileupload to chain successful! TxID: ${commitResult.txId}`, 'success');
        showNotification(`🎉 fileupload to chain successful!`, 'success');
        
        // show link
        showUploadSuccessLinks(commitResult.txId, commitResult.pinId);
        
        // show buzz section after successful upload
        console.log('📝 About to show buzz section with pinId:', commitResult.pinId);
        console.log('📝 commitResult:', commitResult);
        showBuzzSection(commitResult.pinId);
        
        // setTimeout(() => {
        //     resetForm();
        // }, 5000);
        
    } catch (error) {
        console.error('upload to chain failed:', error);
        addLog(`❌ upload to chain failed: ${error.message}`, 'error');
        
        // show different hints based on error type
        if (error.message && error.message.includes('user cancelled')) {
            showNotification('upload to chain operation cancelled', 'warning');
        } else {
            showNotification('upload to chain failed: ' + error.message, 'error');
        }
        
        uploadBtn.disabled = false;
        uploadBtn.textContent = '🚀 Start Upload to Chain';
        progress.classList.remove('show');
    }
}

// Start upload flow (Smart Upload - Auto choose chunked or direct based on file size)
async function startUpload3() {
    // Validate wallet connection
    if (!walletConnected) {
        showNotification('⚠️ Please connect Metalet Wallet', 'warning');
        addLog('❌ Wallet not connected', 'error');
        return;
    }
    
    // Validate file selection
    if (!selectedFile) {
        showNotification('⚠️ Please select file to upload', 'warning');
        addLog('❌ No file selected', 'error');
        return;
    }
    
    // Check file size (10MB = 10485760 bytes)
    const fileSizeLimit = 10 * 1024 * 1024; // 10MB
    const useChunkedUpload = selectedFile.size > fileSizeLimit;
    
    if (useChunkedUpload) {
        addLog(`📦 Large file detected (${formatFileSize(selectedFile.size)}), using chunked upload...`, 'info');
        await startChunkedUpload();
    } else {
        addLog(`📄 Small file (${formatFileSize(selectedFile.size)}), using direct upload...`, 'info');
        await startUpload2();
    }
}

// Start chunked upload flow
async function startChunkedUpload() {
    try {
        uploadBtn.disabled = true;
        uploadBtn.textContent = 'Preparing...';
        progress.classList.add('show');
        
        showNotification('Starting Chunked Upload...', 'info');
        
        // Step 1: Read file and convert to base64
        updateProgress(10, 'Step 1/6: Reading file...');
        const fileContent = await readFileAsBase64(selectedFile);
        addLog(`✅ File read successfully (${formatFileSize(selectedFile.size)})`, 'success');
        
        // Step 2: Estimate chunked upload fee
        updateProgress(20, 'Step 2/6: Estimating chunked upload fee...');
        const estimateResult = await estimateChunkedUploadFee(fileContent);
        addLog(`✅ Fee estimation completed`, 'success');
        addLog(`📊 Chunks: ${estimateResult.chunkNumber}, Total fee: ${formatSatoshis(estimateResult.totalFee)}`, 'info');
        
        // Step 3: Show confirmation dialog with chunk details
        updateProgress(30, 'Step 3/6: Showing confirmation...');
        const confirmed = await showChunkedUploadConfirmation(estimateResult);
        if (!confirmed) {
            addLog('⚠️ User cancelled chunked upload', 'warning');
            uploadBtn.disabled = false;
            uploadBtn.textContent = '🚀 Start Upload to Chain';
            progress.classList.remove('show');
            return;
        }
        
        // Step 4: Get UTXOs for merge transaction
        updateProgress(40, 'Step 4/7: Getting UTXOs...');
        
        // Calculate fees for building PreTx transactions
        // PreTx size estimation: base + inputs + outputs (approximate)
        const preTxBaseSize = 200; // Base transaction overhead
        const preTxInputSize = 150; // Per input with signature
        const preTxOutputSize = 34; // Per output
        const feeRate = Number(document.getElementById('feeRateInput').value) || 1;
        
        // Estimate chunk PreTx size (1 input, no outputs yet - backend will add)
        const chunkPreTxSize = preTxBaseSize + preTxInputSize;
        const chunkPreTxBuildFee = Math.ceil(chunkPreTxSize * feeRate);
        
        // Estimate index PreTx size (1 input, no outputs yet - backend will add)
        const indexPreTxSize = preTxBaseSize + preTxInputSize;
        const indexPreTxBuildFee = Math.ceil(indexPreTxSize * feeRate);
        
        // Calculate total required amount for merge transaction
        // chunkPreTxOutput = chunkPreTxFee + chunkPreTxBuildFee
        // indexPreTxOutput = indexPreTxFee + indexPreTxBuildFee
        // mergeTxFee = merge transaction fee
        const chunkPreTxOutputAmount = estimateResult.chunkPreTxFee + chunkPreTxBuildFee;
        const indexPreTxOutputAmount = estimateResult.indexPreTxFee + indexPreTxBuildFee;
        
        // Estimate merge transaction fee
        const mergeTxBaseSize = 200;
        const mergeTxInputSize = 150;
        const mergeTxOutputSize = 34;
        const estimatedMergeTxInputs = 2; // Assume 2 inputs
        const mergeTxSize = mergeTxBaseSize + (mergeTxInputSize * estimatedMergeTxInputs) + (mergeTxOutputSize * 2); // 2 outputs
        const mergeTxFee = Math.ceil(mergeTxSize * feeRate);
        
        const totalRequiredAmount = chunkPreTxOutputAmount + indexPreTxOutputAmount + mergeTxFee;
        
        const allUtxos = await getWalletUTXOs(totalRequiredAmount);
        addLog(`✅ Got ${allUtxos.utxos.length} UTXO(s), total: ${allUtxos.totalAmount} satoshis`, 'success');
        addLog(`💰 Chunk PreTx output: ${formatSatoshis(chunkPreTxOutputAmount)}`, 'info');
        addLog(`💰 Index PreTx output: ${formatSatoshis(indexPreTxOutputAmount)}`, 'info');
        addLog(`💰 Merge tx fee: ${formatSatoshis(mergeTxFee)}`, 'info');
        
        // Step 5: Build merge transaction with two outputs
        updateProgress(50, 'Step 5/7: Building merge transaction...');
        showNotification('Please confirm merge transaction in wallet...', 'info');
        
        const mergeResult = await buildChunkedUploadMergeTx(
            allUtxos,
            chunkPreTxOutputAmount,
            indexPreTxOutputAmount,
            mergeTxFee
        );
        
        addLog(`✅ Merge transaction built: ${mergeResult.mergeTxId}`, 'success');
        addLog(`📦 Chunk PreTx UTXO: output ${mergeResult.chunkPreTxOutputIndex}, ${formatSatoshis(chunkPreTxOutputAmount)}`, 'success');
        addLog(`📦 Index PreTx UTXO: output ${mergeResult.indexPreTxOutputIndex}, ${formatSatoshis(indexPreTxOutputAmount)}`, 'success');
        
        // Step 6: Build and sign pre-transactions using merge tx outputs
        updateProgress(60, 'Step 6/7: Building and signing pre-transactions...');
        showNotification('Please confirm pre-transactions in wallet...', 'info');
        
        // Build chunk funding pre-tx using merge tx output
        const chunkPreTxUtxo = {
            utxos: [{
                txId: mergeResult.mergeTxId,
                outputIndex: mergeResult.chunkPreTxOutputIndex,
                script: mergeResult.chunkPreTxScript,
                satoshis: chunkPreTxOutputAmount
            }],
            totalAmount: chunkPreTxOutputAmount
        };
        const chunkPreTxHex = await buildChunkFundingPreTx(chunkPreTxUtxo, estimateResult.chunkPreTxFee);
        addLog(`✅ Chunk funding pre-tx signed`, 'success');
        
        // Build index pre-tx using merge tx output
        const indexPreTxUtxo = {
            utxos: [{
                txId: mergeResult.mergeTxId,
                outputIndex: mergeResult.indexPreTxOutputIndex,
                script: mergeResult.indexPreTxScript,
                satoshis: indexPreTxOutputAmount
            }],
            totalAmount: indexPreTxOutputAmount
        };
        const indexPreTxHex = await buildIndexPreTx(indexPreTxUtxo, estimateResult.indexPreTxFee);
        addLog(`✅ Index pre-tx signed`, 'success');
        
        // Step 7: Chunked upload (build chunk transactions and index transaction)
        updateProgress(80, 'Step 7/7: Uploading chunks to chain...');
        const uploadResult = await chunkedUpload(fileContent, chunkPreTxHex, indexPreTxHex, mergeResult.mergeTxHex);
        
        // Completed
        updateProgress(100, 'Upload completed!');
        addLog(`✅ File uploaded successfully! Index TxID: ${uploadResult.indexTxId}`, 'success');
        showNotification(`🎉 File uploaded successfully!`, 'success');
        
        // Show links
        showUploadSuccessLinks(uploadResult.indexTxId, uploadResult.indexTxId + 'i0');
        
        // Show buzz section after successful upload
        console.log('📝 About to show buzz section with pinId:', uploadResult.indexTxId + 'i0');
        showBuzzSection(uploadResult.indexTxId + 'i0');
        
        // Reset button state on success
        uploadBtn.disabled = false;
        uploadBtn.textContent = '🚀 Start Upload to Chain';
        progress.classList.remove('show');
        
    } catch (error) {
        console.error('❌ Chunked upload failed:', error);
        addLog(`❌ Chunked upload failed: ${error.message}`, 'error');
        
        // Show different hints based on error type
        if (error.message && error.message.includes('user cancelled')) {
            showNotification('Upload operation cancelled', 'warning');
        } else {
            showNotification('Upload failed: ' + error.message, 'error');
        }
        
        // Reset button state on error
        uploadBtn.disabled = false;
        uploadBtn.textContent = '🚀 Start Upload to Chain';
        progress.classList.remove('show');
    }
}

// Read file as base64
function readFileAsBase64(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
            // Remove data URL prefix (e.g., "data:image/jpeg;base64,")
            const base64 = reader.result.split(',')[1] || reader.result;
            resolve(base64);
        };
        reader.onerror = reject;
        reader.readAsDataURL(file);
    });
}

// Estimate chunked upload fee
async function estimateChunkedUploadFee(fileContentBase64) {
    try {
        addLog('Calling EstimateChunkedUpload API...', 'info');
        
        const path = document.getElementById('pathInput').value;
        const contentType = buildContentType(selectedFile);
        
        const requestBody = {
            fileName: selectedFile.name,
            content: fileContentBase64,
            path: path,
            contentType: contentType,
            feeRate: Number(document.getElementById('feeRateInput').value) || 1
        };
        
        const response = await fetch(`${API_BASE}/api/v1/files/estimate-chunked-upload`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP Error: ${response.status}`);
        }
        
        const result = await response.json();
        
        if (result.code !== 0) {
            throw new Error(result.message);
        }
        
        addLog(`✅ Estimate successful`, 'success');
        return result.data;
    } catch (error) {
        console.error('❌ Failed to estimate chunked upload fee:', error);
        throw new Error(`Failed to estimate fee: ${error.message}`);
    }
}

// Show chunked upload confirmation dialog
function showChunkedUploadConfirmation(estimateResult) {
    return new Promise((resolve) => {
        // Create modal dialog
        const modal = document.createElement('div');
        modal.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.5);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 10000;
        `;
        
        const dialog = document.createElement('div');
        dialog.style.cssText = `
            background: white;
            padding: 30px;
            border-radius: 12px;
            max-width: 600px;
            width: 90%;
            max-height: 80vh;
            overflow-y: auto;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        `;
        
        dialog.innerHTML = `
            <h2 style="margin-top: 0; color: #333;">📦 Chunked Upload Confirmation</h2>
            <div style="margin: 20px 0;">
                <div style="margin: 10px 0;">
                    <strong>📊 File Information:</strong>
                    <ul style="margin: 5px 0; padding-left: 20px;">
                        <li>File Name: ${selectedFile.name}</li>
                        <li>File Size: ${formatFileSize(selectedFile.size)}</li>
                        <li>Chunk Size: ${formatFileSize(estimateResult.chunkSize)}</li>
                        <li>Number of Chunks: ${estimateResult.chunkNumber}</li>
                    </ul>
                </div>
                <div style="margin: 10px 0;">
                    <strong>💰 Fee Information:</strong>
                    <ul style="margin: 5px 0; padding-left: 20px;">
                        <li>Chunk Funding Fee: ${formatSatoshis(estimateResult.chunkPreTxFee)}</li>
                        <li>Index Transaction Fee: ${formatSatoshis(estimateResult.indexPreTxFee)}</li>
                        <li>Total Fee: ${formatSatoshis(estimateResult.totalFee)}</li>
                        <li>Per Chunk Fee: ${formatSatoshis(estimateResult.perChunkFee)}</li>
                    </ul>
                </div>
                <div style="margin: 10px 0; padding: 10px; background: #f0f0f0; border-radius: 6px;">
                    <strong>ℹ️ Note:</strong>
                    <p style="margin: 5px 0; font-size: 14px; color: #666;">
                        This large file will be split into ${estimateResult.chunkNumber} chunks. 
                        You need to confirm ${estimateResult.chunkNumber + 2} transactions:
                        <br>1. Chunk funding transaction (${estimateResult.chunkNumber} outputs)
                        <br>2. ${estimateResult.chunkNumber} chunk transactions
                        <br>3. Index transaction
                    </p>
                </div>
            </div>
            <div style="display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px;">
                <button id="cancelChunkedUpload" style="
                    padding: 10px 20px;
                    background: #ccc;
                    color: #333;
                    border: none;
                    border-radius: 6px;
                    cursor: pointer;
                    font-size: 14px;
                ">Cancel</button>
                <button id="confirmChunkedUpload" style="
                    padding: 10px 20px;
                    background: #28a745;
                    color: white;
                    border: none;
                    border-radius: 6px;
                    cursor: pointer;
                    font-size: 14px;
                    font-weight: bold;
                ">Confirm & Upload</button>
            </div>
        `;
        
        modal.appendChild(dialog);
        document.body.appendChild(modal);
        
        // Handle button clicks
        document.getElementById('confirmChunkedUpload').addEventListener('click', () => {
            document.body.removeChild(modal);
            resolve(true);
        });
        
        document.getElementById('cancelChunkedUpload').addEventListener('click', () => {
            document.body.removeChild(modal);
            resolve(false);
        });
        
        // Close on background click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                document.body.removeChild(modal);
                resolve(false);
            }
        });
    });
}

// Build merge transaction for chunked upload (creates two outputs for chunkPreTx and indexPreTx)
async function buildChunkedUploadMergeTx(utxoData, chunkPreTxOutputAmount, indexPreTxOutputAmount, mergeTxFee) {
    try {
        addLog('Building merge transaction for chunked upload...', 'info');
        
        const metaContract = window.metaContract;
        if (!metaContract) {
            throw new Error('meta-contract library not loaded');
        }
        
        const mvc = metaContract.mvc;
        const TxComposer = metaContract.TxComposer;
        
        // Create merge transaction with two outputs
        const mergeTx = new mvc.Transaction();
        mergeTx.version = 10;
        
        // Add inputs from UTXOs
        for (const utxo of utxoData.utxos) {
            mergeTx.from({
                txId: utxo.txId,
                outputIndex: utxo.outputIndex,
                script: utxo.script,
                satoshis: utxo.satoshis
            });
        }
        
        // Add two outputs: one for chunk PreTx, one for index PreTx
        mergeTx.to(currentAddress, chunkPreTxOutputAmount);
        mergeTx.to(currentAddress, indexPreTxOutputAmount);
        
        addLog(`📤 Merge transaction: ${chunkPreTxOutputAmount} sats for chunk, ${indexPreTxOutputAmount} sats for index`, 'info');
        
        // Use pay method to sign and broadcast (or just sign)
        const wallet = getWallet();
        if (!wallet || typeof wallet.pay !== 'function') {
            throw new Error('Wallet does not support pay method');
        }
        
        // Create TxComposer
        const txComposer = new TxComposer(mergeTx);
        const txComposerSerialize = txComposer.serialize();
        
        // Build pay params
        const feeRate = Number(document.getElementById('feeRateInput').value) || 1;
        const payParams = {
            transactions: [
                {
                    txComposer: txComposerSerialize,
                    message: 'Merge UTXOs for chunked upload',
                }
            ],
            feeb: feeRate,
        };
        
        addLog('📡 Calling pay method to merge UTXOs...', 'info');
        const payResult = await wallet.pay(payParams);
        
        if (!payResult || !payResult.payedTransactions || payResult.payedTransactions.length === 0) {
            throw new Error('Pay method returned invalid result');
        }
        
        // Deserialize the payed transaction
        const payedTxComposerStr = payResult.payedTransactions[0];
        const payedTxComposer = TxComposer.deserialize(payedTxComposerStr);
        
        // Get signed transaction hex
        const signedMergeTxHex = payedTxComposer.getRawHex();
        const mergeTxId = payedTxComposer.getTxId();
        
        addLog('✅ Merge transaction created and signed', 'success');
        
        // Parse the transaction to get output info
        const parsedMergeTx = new mvc.Transaction(signedMergeTxHex);
        
        // Find outputs by matching amounts (pay method may add change output, so order may vary)
        let chunkPreTxOutputIndex = -1;
        let indexPreTxOutputIndex = -1;
        let chunkPreTxScript = null;
        let indexPreTxScript = null;
        
        // Tolerance for amount matching (due to fee calculation differences)
        const amountTolerance = 1000; // 1000 satoshis tolerance
        
        // First pass: find outputs by exact or near amount match
        for (let i = 0; i < parsedMergeTx.outputs.length; i++) {
            const output = parsedMergeTx.outputs[i];
            const outputScript = output.script.toHex();
            const outputAmount = output.satoshis;
            
            // Check if this output is to our address
            try {
                const addr = output.script.toAddress(mvc.Networks.livenet);
                if (addr && addr.toString() === currentAddress) {
                    // Match chunk PreTx output by amount
                    if (chunkPreTxOutputIndex === -1 && 
                        Math.abs(outputAmount - chunkPreTxOutputAmount) <= amountTolerance) {
                        chunkPreTxOutputIndex = i;
                        chunkPreTxScript = outputScript;
                        addLog(`✅ Found chunk PreTx output at index ${i}: ${outputAmount} sats (expected: ${chunkPreTxOutputAmount})`, 'success');
                    }
                    // Match index PreTx output by amount
                    else if (indexPreTxOutputIndex === -1 && 
                             Math.abs(outputAmount - indexPreTxOutputAmount) <= amountTolerance) {
                        indexPreTxOutputIndex = i;
                        indexPreTxScript = outputScript;
                        addLog(`✅ Found index PreTx output at index ${i}: ${outputAmount} sats (expected: ${indexPreTxOutputAmount})`, 'success');
                    }
                }
            } catch (e) {
                // Not a standard P2PKH output, skip
                continue;
            }
        }
        
        // Fallback: if we couldn't find by amount, use first two outputs to our address
        if (chunkPreTxOutputIndex === -1 || indexPreTxOutputIndex === -1) {
            addLog('⚠️ Could not match outputs by amount, trying fallback...', 'warning');
            let foundOutputs = [];
            for (let i = 0; i < parsedMergeTx.outputs.length; i++) {
                const output = parsedMergeTx.outputs[i];
                try {
                    const addr = output.script.toAddress(mvc.Networks.livenet);
                    if (addr && addr.toString() === currentAddress) {
                        foundOutputs.push({ index: i, script: output.script.toHex(), amount: output.satoshis });
                    }
                } catch (e) {
                    continue;
                }
            }
            
            if (foundOutputs.length >= 2) {
                // Use first two outputs to our address
                chunkPreTxOutputIndex = foundOutputs[0].index;
                chunkPreTxScript = foundOutputs[0].script;
                indexPreTxOutputIndex = foundOutputs[1].index;
                indexPreTxScript = foundOutputs[1].script;
                addLog(`⚠️ Using first two outputs: chunk=${chunkPreTxOutputIndex}, index=${indexPreTxOutputIndex}`, 'warning');
            } else {
                throw new Error(`Merge transaction does not have enough outputs to our address. Found: ${foundOutputs.length}, need: 2`);
            }
        }
        
        addLog(`✅ Chunk PreTx output: index ${chunkPreTxOutputIndex}, amount ${parsedMergeTx.outputs[chunkPreTxOutputIndex].satoshis}`, 'success');
        addLog(`✅ Index PreTx output: index ${indexPreTxOutputIndex}, amount ${parsedMergeTx.outputs[indexPreTxOutputIndex].satoshis}`, 'success');
        
        return {
            mergeTxId: mergeTxId,
            mergeTxHex: signedMergeTxHex,
            chunkPreTxOutputIndex: chunkPreTxOutputIndex,
            indexPreTxOutputIndex: indexPreTxOutputIndex,
            chunkPreTxScript: chunkPreTxScript,
            indexPreTxScript: indexPreTxScript
        };
        
    } catch (error) {
        console.error('❌ Failed to build merge transaction:', error);
        throw new Error(`Failed to build merge transaction: ${error.message}`);
    }
}

// Build chunk funding pre-tx (multiple outputs to assistent address)
async function buildChunkFundingPreTx(utxoData, totalChunkFee) {
    try {
        addLog('Building chunk funding pre-transaction...', 'info');
        
        // Build a transaction with inputs signed with signNull
        // Backend will add multiple outputs to assistent address
        const metaContract = window.metaContract;
        if (!metaContract) {
            throw new Error('meta-contract library not loaded');
        }
        
        const mvc = metaContract.mvc;
        
        // Create transaction with inputs only (no outputs - backend will add them)
        const tx = new mvc.Transaction();
        tx.version = 10;
        
        // Add inputs from UTXOs
        for (const utxo of utxoData.utxos) {
            tx.from({
                txId: utxo.txId,
                outputIndex: utxo.outputIndex,
                script: utxo.script,
                satoshis: utxo.satoshis
            });
        }
        
        // Sign with signNull (SIGHASH_NONE | SIGHASH_ANYONECANPAY = 0x80 | 0x40 = 0xC0)
        const wallet = getWallet();
        if (!wallet || typeof wallet.signTransaction !== 'function') {
            throw new Error('Wallet does not support signTransaction');
        }
        
        // Sign each input with signNull
        for (let i = 0; i < utxoData.utxos.length; i++) {
            const utxo = utxoData.utxos[i];
            const signResult = await wallet.signTransaction({
                transaction: {
                    txHex: tx.toString(),
                    address: currentAddress,
                    inputIndex: i,
                    scriptHex: utxo.script,
                    satoshis: utxo.satoshis,
                    sigtype: 0x2 | 0x40 // SIGHASH_NONE | SIGHASH_ForK_id
                }
            });
            
            const sig = signResult.signature.sig;
            const publicKey = signResult.signature.publicKey;
            const unlockingScript = mvc.Script.buildPublicKeyHashIn(
                publicKey,
                mvc.crypto.Signature.fromTxFormat(Buffer.from(sig, 'hex')).toDER(),
                0x2 | 0x40
            );
            tx.inputs[i].setScript(unlockingScript);
        }
        
        const signedTxHex = tx.toString();
        addLog('✅ Chunk funding pre-tx signed (signNull)', 'success');
        return signedTxHex;
        
    } catch (error) {
        console.error('❌ Failed to build chunk funding pre-tx:', error);
        throw new Error(`Failed to build chunk funding pre-tx: ${error.message}`);
    }
}

// Build index pre-tx
async function buildIndexPreTx(utxoData, indexFee) {
    try {
        addLog('Building index pre-transaction...', 'info');
        
        // Build a transaction with inputs signed with signNull
        // Backend will add OP_RETURN output and change output
        const metaContract = window.metaContract;
        if (!metaContract) {
            throw new Error('meta-contract library not loaded');
        }
        
        const mvc = metaContract.mvc;
        
        const tx = new mvc.Transaction();
        tx.version = 10;
        
        // Add inputs from UTXOs
        for (const utxo of utxoData.utxos) {
            tx.from({
                txId: utxo.txId,
                outputIndex: utxo.outputIndex,
                script: utxo.script,
                satoshis: utxo.satoshis
            });
        }
        
        // No outputs yet - backend will add OP_RETURN and change output
        const wallet = getWallet();
        if (!wallet || typeof wallet.signTransaction !== 'function') {
            throw new Error('Wallet does not support signTransaction');
        }
        
        // Sign each input with signNull
        for (let i = 0; i < utxoData.utxos.length; i++) {
            const utxo = utxoData.utxos[i];
            const signResult = await wallet.signTransaction({
                transaction: {
                    txHex: tx.toString(),
                    address: currentAddress,
                    inputIndex: i,
                    scriptHex: utxo.script,
                    satoshis: utxo.satoshis,
                    sigtype: 0x2 | 0x40 // SIGHASH_NONE | SIGHASH_ForK_id | SIGHASH_ANYONECANPAY
                }
            });
            
            const sig = signResult.signature.sig;
            const publicKey = signResult.signature.publicKey;
            const unlockingScript = mvc.Script.buildPublicKeyHashIn(
                publicKey,
                mvc.crypto.Signature.fromTxFormat(Buffer.from(sig, 'hex')).toDER(),
                0x2 | 0x40
            );
            tx.inputs[i].setScript(unlockingScript);
        }
        
        const signedTxHex = tx.toString();
        addLog('✅ Index pre-tx signed (signNull)', 'success');
        return signedTxHex;
        
    } catch (error) {
        console.error('❌ Failed to build index pre-tx:', error);
        throw new Error(`Failed to build index pre-tx: ${error.message}`);
    }
}

// Chunked upload
async function chunkedUpload(fileContentBase64, chunkPreTxHex, indexPreTxHex, mergeTxHex) {
    try {
        addLog('Calling ChunkedUpload API...', 'info');
        
        const path = document.getElementById('pathInput').value;
        const contentType = buildContentType(selectedFile);
        const metaId = await calculateMetaID(currentAddress);
        
        const requestBody = {
            metaId: metaId,
            address: currentAddress,
            fileName: selectedFile.name,
            content: fileContentBase64,
            path: path,
            operation: document.getElementById('operationSelect').value || 'create',
            contentType: contentType,
            chunkPreTxHex: chunkPreTxHex,
            indexPreTxHex: indexPreTxHex,
            mergeTxHex: mergeTxHex, // 合并交易 hex，用于广播
            feeRate: Number(document.getElementById('feeRateInput').value) || 1,
            isBroadcast: true // Auto broadcast
        };
        
        const response = await fetch(`${API_BASE}/api/v1/files/chunked-upload`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP Error: ${response.status}`);
        }
        
        const result = await response.json();
        
        if (result.code !== 0) {
            throw new Error(result.message);
        }
        
        addLog(`✅ ChunkedUpload success!`, 'success');
        addLog(`📝 Index TxID: ${result.data.indexTxId}`, 'success');
        addLog(`📊 Status: ${result.data.status}`, 'success');
        
        return result.data;
    } catch (error) {
        console.error('❌ ChunkedUpload failed:', error);
        throw new Error(`ChunkedUpload failed: ${error.message}`);
    }
}

// Start upload flow (DirectUpload - One-step method)
async function startUpload2() {
    // Validate wallet connection
    if (!walletConnected) {
        showNotification('⚠️ Please connect Metalet Wallet', 'warning');
        addLog('❌ Wallet not connected', 'error');
        return;
    }
    
    // Validate file selection
    if (!selectedFile) {
        showNotification('⚠️ Please select file to upload', 'warning');
        addLog('❌ No file selected', 'error');
        return;
    }
    
    console.log('✅ Validation passed: wallet connected, file selected');
    addLog('Starting Direct Upload flow...', 'info');

    try {
        uploadBtn.disabled = true;
        uploadBtn.textContent = 'Uploading...';
        progress.classList.add('show');
        
        showNotification('Starting Direct Upload...', 'info');
        
        // Step 1: Estimate file upload fee
        updateProgress(10, 'Step 1/5: Estimating upload fee...');
        const estimatedFee = await estimateUploadFee();
        addLog(`💰 Estimated fee: ${estimatedFee} satoshis`, 'info');
        
        // Step 2: Get UTXOs
        updateProgress(30, 'Step 2/5: Getting UTXOs...');
        const utxos = await getWalletUTXOs(estimatedFee);
        addLog(`✅ Got ${utxos.utxos.length} UTXO(s), total: ${utxos.totalAmount} satoshis`, 'success');
        
        // Step 3: Merge UTXOs if needed (for SIGHASH_SINGLE compatibility)
        let finalUtxo = null;
        let mergeTxHex = ''; // Merge transaction hex (if UTXOs were merged)
        
        if (utxos.utxos.length > 1) {
            updateProgress(40, 'Step 3/5: Merging UTXOs...');
            addLog(`⚠️ Multiple UTXOs detected, merging into one UTXO...`, 'info');
            showNotification('Please confirm UTXO merge transaction...', 'info');
            const mergeResult = await mergeUTXOs(utxos, estimatedFee);
            finalUtxo = {
                utxos: mergeResult.utxos,
                totalAmount: mergeResult.totalAmount
            };
            mergeTxHex = mergeResult.mergeTxHex || '';
            addLog(`✅ UTXOs merged successfully`, 'success');
            addLog(`📦 Merged UTXO: ${finalUtxo.totalAmount} satoshis`, 'info');
        } else {
            finalUtxo = {
                utxos: utxos.utxos,
                totalAmount: utxos.totalAmount
            };
            addLog(`✅ Single UTXO, no merge needed`, 'success');
        }
        
        // Step 4: Build and sign base transaction
        updateProgress(60, 'Step 4/5: Building and signing transaction...');
        showNotification('Please confirm signature in wallet...', 'info');
        const preTxHex = await buildAndSignBaseTx(finalUtxo);
        addLog(`✅ Base transaction signed`, 'success');
        
        // Step 5: Direct upload (one-step: add OP_RETURN + calculate change + broadcast)
        updateProgress(80, 'Step 5/5: Uploading to chain...');
        const uploadResult = await directUpload(preTxHex, finalUtxo.totalAmount, mergeTxHex);
        
        // Completed
        updateProgress(100, 'Upload completed!');
        addLog(`✅ File uploaded successfully! TxID: ${uploadResult.txId}`, 'success');
        showNotification(`🎉 File uploaded successfully!`, 'success');
        
        // Show links
        showUploadSuccessLinks(uploadResult.txId, uploadResult.pinId);
        
        // Show buzz section after successful upload
        console.log('📝 About to show buzz section with pinId:', uploadResult.pinId);
        showBuzzSection(uploadResult.pinId);
        
        // Reset button state on success
        uploadBtn.disabled = false;
        uploadBtn.textContent = '🚀 Start Upload to Chain';
        progress.classList.remove('show');
        
    } catch (error) {
        console.error('❌ Direct upload failed:', error);
        addLog(`❌ Direct upload failed: ${error.message}`, 'error');
        
        // Show different hints based on error type
        if (error.message && error.message.includes('user cancelled')) {
            showNotification('Upload operation cancelled', 'warning');
        } else {
            showNotification('Upload failed: ' + error.message, 'error');
        }
        
        // Reset button state on error
        uploadBtn.disabled = false;
        uploadBtn.textContent = '🚀 Start Upload to Chain';
        progress.classList.remove('show');
    }
}

// Estimate file upload fee
async function estimateUploadFee() {
    try {
        // Base transaction size estimation
        const baseSize = 200; // Basic transaction overhead
        const inputSize = 150; // Per input size (with signature)
        const outputSize = 34; // Per output size
        const opReturnOverhead = 50; // OP_RETURN script overhead
        
        // File size
        const fileSize = selectedFile.size;
        
        // Calculate OP_RETURN output size
        // MetaID protocol: metaid + operation + path + encryption + version + contentType + content
        const path = document.getElementById('pathInput').value;
        const fileHost = document.getElementById('fileHostInput').value.trim();
        const finalPath = fileHost ? fileHost + ':' + path : path;
        
        const metadataSize = 6 + 10 + finalPath.length + 10 + 10 + 50; // Rough estimate
        const opReturnSize = opReturnOverhead + metadataSize + fileSize;
        
        // Total transaction size estimation (1 input, 2 outputs: change + OP_RETURN)
        const estimatedTxSize = baseSize + inputSize + outputSize * 2 + opReturnSize;
        
        // Get fee rate
        const feeRate = Number(document.getElementById('feeRateInput').value) || 1;
        
        // Calculate fee
        const estimatedFee = Math.ceil(estimatedTxSize * feeRate);
        
        // Add safety margin (20%)
        const feeWithMargin = Math.ceil(estimatedFee * 1.2);
        
        addLog(`📏 Estimated tx size: ${estimatedTxSize} bytes`, 'info');
        addLog(`💸 Fee rate: ${feeRate} sat/byte`, 'info');
        addLog(`💰 Estimated fee (with 20% margin): ${feeWithMargin} satoshis`, 'info');
        
        return feeWithMargin;
    } catch (error) {
        console.error('❌ Failed to estimate fee:', error);
        throw new Error(`Failed to estimate fee: ${error.message}`);
    }
}

// Get wallet UTXOs
async function getWalletUTXOs(requiredAmount) {
    try {
        addLog('Getting wallet UTXOs...', 'info');
        
        // Get UTXOs from wallet
        const wallet = getWallet();
        if (!wallet) {
            throw new Error('Wallet not available');
        }
        
        const utxos = await wallet.getUtxos();
        
        console.log('📦 Wallet UTXOs:', utxos);
        
        if (!utxos || utxos.length === 0) {
            throw new Error('No available UTXOs in wallet');
        }
 
        // Filter UTXOs: only select UTXOs > 600 satoshis (to ensure change output is possible)
        const filler = 600;
        const fillerUtxos = utxos.filter(utxo => utxo.value > filler);
        
        if (!fillerUtxos || fillerUtxos.length === 0) {
            throw new Error('No UTXOs larger than 600 satoshis available in wallet');
        }
    
        // Sort UTXOs by amount (descending)
        const sortedUtxos = fillerUtxos.sort((a, b) => b.value - a.value);
        
        // Get meta-contract library for address conversion
        const metaContract = window.metaContract;
        if (!metaContract || !metaContract.mvc) {
            throw new Error('meta-contract library not loaded');
        }
        const mvc = metaContract.mvc;
        
        // Select UTXOs to meet required amount
        let selectedUtxos = [];
        let totalAmount = 0;
        
        for (const utxo of sortedUtxos) {
            // Convert address to script
            let scriptHex = mvc.Script.buildPublicKeyHashOut(utxo.address).toHex();
            selectedUtxos.push({
                txId: utxo.txid,
                outputIndex: utxo.outIndex,
                script: scriptHex,
                satoshis: utxo.value
            });
            totalAmount += utxo.value;
            
            // Add buffer for change output ( 1 satoshi for receiver)
            if (totalAmount >= requiredAmount + 1) {
                break;
            }
        }

        
        if (totalAmount < requiredAmount + 1) {
            throw new Error(`Insufficient balance! Need ${requiredAmount + 1} satoshis, but only have ${totalAmount} satoshis`);
        }
        console.log('📝 Selected UTXO:', selectedUtxos);
        
        addLog(`✅ Selected ${selectedUtxos.length} UTXO(s), total: ${totalAmount} satoshis`, 'success');
        
        return {
            utxos: selectedUtxos,
            totalAmount: totalAmount
        };
    } catch (error) {
        console.error('❌ Failed to get UTXOs:', error);
        throw new Error(`Failed to get UTXOs: ${error.message}`);
    }
}

// Merge multiple UTXOs into one using pay method
async function mergeUTXOs(utxoData, estimatedFee) {
    try {
        addLog('Merging multiple UTXOs into one...', 'info');
        addLog(`📦 Merging ${utxoData.utxos.length} UTXOs (total: ${utxoData.totalAmount} sats)`, 'info');
        
        // Check if pay method is available
        const wallet = getWallet();
        if (!wallet) {
            throw new Error('Wallet not available');
        }
        
        if (typeof wallet.pay !== 'function') {
            throw new Error('Wallet does not support pay method');
        }
        
        // Get meta-contract library for TxComposer
        const metaContract = window.metaContract;
        if (!metaContract) {
            throw new Error('meta-contract library not loaded, please refresh page');
        }
        
        const mvc = metaContract.mvc;
        const TxComposer = metaContract.TxComposer;
        
        if (!mvc || !TxComposer) {
            throw new Error('mvc or TxComposer not available');
        }
        
        // Create merge transaction - we only specify the output
        // pay method will automatically select inputs, add change, and sign
        const mergeTx = new mvc.Transaction();
        mergeTx.version = 10;
        
        // Add single output to ourselves (this will merge all UTXOs into one)
        // We use a large amount that will consume most UTXOs
        // The pay method will handle input selection and change calculation
        mergeTx.to(currentAddress, estimatedFee); 
        
        addLog(`📤 Merge target: consolidate to ${currentAddress}`, 'info');
        console.log('📦 Merge transaction template:', mergeTx);
        
        // Create TxComposer for pay method
        const txComposer = new TxComposer(mergeTx);
        const txComposerSerialize = txComposer.serialize();
        
        // Build pay params
        const feeRate = Number(document.getElementById('feeRateInput').value) || 1;
        const payParams = {
            transactions: [
                {
                    txComposer: txComposerSerialize,
                    message: 'Merge UTXOs',
                }
            ],
            // broadcast: false, // Don't broadcast, we just need the signed tx
            feeb: feeRate,
        };
        
        addLog('📡 Calling pay method to merge and sign...', 'info');
        console.log('📡 Pay params:', payParams);
        
        // Call pay method - it will auto select inputs, add change, and sign
        const payResult = await wallet.pay(payParams);
        console.log('✅ Pay result:', payResult);
        
        if (!payResult || !payResult.payedTransactions || payResult.payedTransactions.length === 0) {
            throw new Error('Pay method returned invalid result');
        }
        
        // Deserialize the payed transaction
        const payedTxComposerStr = payResult.payedTransactions[0];
        const payedTxComposer = TxComposer.deserialize(payedTxComposerStr);
        
        // Get signed transaction hex
        const signedMergeTxHex = payedTxComposer.getRawHex();
        const mergeTxId = payedTxComposer.getTxId();
        
        addLog('✅ Merge transaction created and signed via pay', 'success');
        console.log('📝 Signed merge tx:', signedMergeTxHex.substring(0, 100) + '...');
        console.log('📝 Merge txId:', mergeTxId);
        
        // Parse the transaction to get output info
        const parsedMergeTx = new mvc.Transaction(signedMergeTxHex);
        console.log('📦 Parsed merge transaction:', parsedMergeTx);
        
        // Find the output that goes to our address (the merged UTXO)
        // pay method will create outputs: [our output, possibly change output]
        let mergedOutputIndex = -1;
        let mergedOutputAmount = 0;
        
        for (let i = 0; i < parsedMergeTx.outputs.length; i++) {
            const output = parsedMergeTx.outputs[i];
            const outputScript = output.script.toHex();
            
            // Check if this output is to our address
            // We can identify it by checking if it's a P2PKH to our address
            try {
                const addr = output.script.toAddress(mvc.Networks.livenet);
                if (addr && addr.toString() === currentAddress) {
                    mergedOutputIndex = i;
                    mergedOutputAmount = output.satoshis;
                    break;
                }
            } catch (e) {
                // Not a standard P2PKH output, skip
                continue;
            }
        }
        
        if (mergedOutputIndex === -1) {
            // Fallback: use the first output (usually the main output)
            mergedOutputIndex = 0;
            mergedOutputAmount = parsedMergeTx.outputs[0].satoshis;
            addLog('⚠️ Could not identify output, using first output', 'warning');
        }
        
        // Create new UTXO info from merge transaction
        const newUtxo = {
            txId: mergeTxId,
            outputIndex: mergedOutputIndex,
            script: parsedMergeTx.outputs[mergedOutputIndex].script.toHex(),
            satoshis: mergedOutputAmount
        };
        
        addLog(`✅ New merged UTXO: ${newUtxo.satoshis} satoshis at output ${mergedOutputIndex}`, 'success');
        console.log('📦 New UTXO:', newUtxo);
        
        // Return merged UTXO info (no broadcast needed)
        return {
            utxos: [newUtxo],
            totalAmount: newUtxo.satoshis,
            mergeTxId: mergeTxId,
            mergeTxHex: signedMergeTxHex
        };
        
    } catch (error) {
        console.error('❌ Failed to merge UTXOs:', error);
        throw new Error(`Failed to merge UTXOs: ${error.message}`);
    }
}

// Build and sign base transaction (SIGHASH_SINGLE compatible - requires single UTXO)
async function buildAndSignBaseTx(utxoData) {
    try {
        addLog('Building base MVC transaction...', 'info');
        
        // Validate: must have exactly one UTXO for SIGHASH_SINGLE
        if (!utxoData.utxos || utxoData.utxos.length !== 1) {
            throw new Error(`SIGHASH_SINGLE requires exactly 1 UTXO, got ${utxoData.utxos ? utxoData.utxos.length : 0}`);
        }
        
        // Get meta-contract library
        const metaContract = window.metaContract;
        if (!metaContract) {
            throw new Error('meta-contract library not loaded, please refresh page');
        }
        
        const mvc = metaContract.mvc;
        if (!mvc) {
            throw new Error('mvc library not available');
        }
        
        const utxo = utxoData.utxos[0]; // Single UTXO
        addLog(`📦 Using single UTXO: ${utxo.satoshis} satoshis`, 'info');
        
        // Create new transaction
        const tx = new mvc.Transaction();
        tx.version = 10; // MVC version
        
        // Add single input
        tx.from({
            txId: utxo.txId,
            outputIndex: utxo.outputIndex,
            script: utxo.script,
            satoshis: utxo.satoshis
        });
        
        // Add receiver output (1 satoshi)
        tx.to(currentAddress, 1);
        
        addLog('✅ Base transaction built (unsigned)', 'success');
        console.log('📦 Unsigned transaction:', tx);
        
        // Serialize to hex
        const txHex = tx.toString();
        
        addLog('Requesting wallet to sign MVC transaction...', 'info');
        console.log('📝 Transaction hex for signing:', txHex.substring(0, 100) + '...');
        
        // Check if signTransaction method exists
        const wallet = getWallet();
        if (!wallet) {
            throw new Error('Wallet not available');
        }
        
        if (typeof wallet.signTransaction !== 'function') {
            throw new Error('Wallet does not support signTransaction method');
        }
        
        // Sign the single input with SIGHASH_SINGLE
        addLog('Signing input with SIGHASH_SINGLE...', 'info');
        
        console.log('📝 Signing input 0...', {
            inputIndex: 0,
            scriptHex: utxo.script,
            satoshis: utxo.satoshis
        });
        
        // Call signTransaction for this input
        const signResult = await wallet.signTransaction({
            transaction: {
                txHex: tx.toString(),
                address: currentAddress,
                inputIndex: 0,
                scriptHex: utxo.script,
                satoshis: utxo.satoshis,
                sigtype: 0x3 | 0x80 | 0x40// SIGHASH_SINGLE | ANYONE_CAN_PAY (0x80 | 0x03 = 0x83) | SIGHASH_ANYONECANPAY (0x01 = 0x01)
            }
        });
        
        console.log('✅ Input signature:', signResult);
        
        if (!signResult || !signResult.signature || !signResult.signature.sig) {
            throw new Error('Failed to get signature');
        }
        
        // Build unlocking script (scriptSig) from signature
        const sig = signResult.signature.sig;
        const publicKey = signResult.signature.publicKey;
        console.log('📝 Public key:', publicKey);
        
        // Build P2PKH unlocking script: <sig> <pubkey>
        const unlockingScript = mvc.Script.buildPublicKeyHashIn(
            publicKey,
            mvc.crypto.Signature.fromTxFormat(Buffer.from(sig, 'hex')).toDER(),
            0x3 | 0x80 | 0x40// SIGHASH_SINGLE | ANYONE_CAN_PAY (0x80 | 0x03 = 0x83) | SIGHASH_ANYONECANPAY (0x01 = 0x01)
        );
        console.log('📝 Unlocking script:', unlockingScript);

        // Set the unlocking script for this input
        tx.inputs[0].setScript(unlockingScript);
        
        addLog('✅ Input signed and script filled', 'success');
        
        // Get final signed transaction hex
        const signedTxHex = tx.toString();
        
        addLog('✅ Transaction signed successfully with SIGHASH_SINGLE', 'success');
        console.log('📝 Final signed txHex:', signedTxHex);
        console.log('📝 Transaction:', tx);
        
        return signedTxHex;
        
    } catch (error) {
        console.error('❌ Failed to build/sign MVC transaction:', error);
        throw new Error(`Failed to build/sign MVC transaction: ${error.message}`);
    }
}

// Direct upload (one-step)
async function directUpload(preTxHex, totalInputAmount, mergeTxHex) {
    try {
        addLog('Calling DirectUpload API...', 'info');
        
        // Build contentType (text types don't need ;binary)
        const contentType = buildContentType(selectedFile);
        
        const path = document.getElementById('pathInput').value;
        
        // Add host information to path if provided
        const fileHost = document.getElementById('fileHostInput').value.trim();
        let finalPath = path;
        if (fileHost) {
            finalPath = fileHost + ':' + path;
            addLog(`🏠 File Host: ${fileHost}`, 'info');
        }
        
        addLog(`📁 Path: ${finalPath}`, 'info');
        addLog(`📄 ContentType: ${contentType}`, 'info');
        addLog(`💰 Total input amount: ${totalInputAmount} satoshis`, 'info');
        
        if (mergeTxHex) {
            addLog(`🔗 Merge transaction will be broadcasted first`, 'info');
        }
        
        const formData = new FormData();
        formData.append('file', selectedFile);
        formData.append('path', finalPath);
        if (mergeTxHex) {
            formData.append('mergeTxHex', mergeTxHex);
        }
        formData.append('preTxHex', preTxHex);
        formData.append('operation', document.getElementById('operationSelect').value);
        formData.append('contentType', contentType);
        formData.append('metaId', await calculateMetaID(currentAddress));
        formData.append('address', currentAddress);
        formData.append('changeAddress', currentAddress);
        formData.append('feeRate', document.getElementById('feeRateInput').value);
        formData.append('totalInputAmount', totalInputAmount.toString());
        
        const response = await fetch(`${API_BASE}/api/v1/files/direct-upload`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            throw new Error(`HTTP Error: ${response.status}`);
        }
        
        const result = await response.json();
        
        if (result.code !== 0) {
            throw new Error(result.message);
        }
        
        addLog(`✅ DirectUpload success!`, 'success');
        addLog(`📝 TxID: ${result.data.txId}`, 'success');
        addLog(`📊 Status: ${result.data.status}`, 'success');
        
        return result.data;
    } catch (error) {
        console.error('❌ DirectUpload failed:', error);
        throw new Error(`DirectUpload failed: ${error.message}`);
    }
}

// pre-upload
async function preUpload() {
    addLog('call PreUpload API...', 'info');
    
    // Build contentType (text types don't need ;binary)
    const contentType = buildContentType(selectedFile);
    
    // build outputs（use wallet address）
    const outputs = [{address: currentAddress, amount: 1}];
    
    addLog(`📄 ContentType: ${contentType}`, 'info');
    addLog(`💰 Outputs: ${currentAddress} (1 satoshi)`, 'info');
    const path = document.getElementById('pathInput').value;
    
    // Add host information to path if provided
    const fileHost = document.getElementById('fileHostInput').value.trim();
    let finalPath = path;
    if (fileHost) {
        finalPath = fileHost + ':' + path;
        addLog(`🏠 File Host: ${fileHost}`, 'info');
        addLog(`📁 Final Path: ${finalPath}`, 'info');
    }
    
    const formData = new FormData();
    formData.append('file', selectedFile);
    formData.append('path', finalPath);
    formData.append('operation', document.getElementById('operationSelect').value);
    formData.append('contentType', contentType);
    // formData.append('changeAddress', currentAddress);
    formData.append('feeRate', document.getElementById('feeRateInput').value);
    formData.append('outputs', JSON.stringify(outputs));
    formData.append('otherOutputs', '[]');
    formData.append('metaId', await calculateMetaID(currentAddress));
    formData.append('address', currentAddress);
    
    const response = await fetch(`${API_BASE}/api/v1/files/pre-upload`, {
        method: 'POST',
        body: formData
    });
    
    if (!response.ok) {
        throw new Error(`HTTP Error: ${response.status}`);
    }
    
    const result = await response.json();
    
    if (result.code !== 0) {
        throw new Error(result.message);
    }
    
    addLog(`PreUpload success, FileID: ${result.data.fileId}, MD5: ${result.data.fileMd5.substring(0, 8)}...`, 'success');
    return result.data;
}

// prepare UTXO and verify balance
async function prepareUTXO(preUploadData) {
    addLog('get wallet balance...', 'info');
    
    // get balance
    const wallet = getWallet();
    if (!wallet) {
        throw new Error('Wallet not available');
    }
    
    const balance = await wallet.getBalance();
    const totalBalance = balance.total || balance.confirmed || balance.balance || 0;
    
    addLog(`💰 current balance: ${totalBalance} satoshis`, 'info');
    addLog(`💸 estimated transaction fee: ${preUploadData.calTxFee} satoshis`, 'info');
    addLog(`📏 transaction size: ${preUploadData.calTxSize} bytes`, 'info');
    
    // verify balance：80% of balance must be >= transaction fee
    const requiredBalance = preUploadData.calTxFee;
    const availableBalance = Math.floor(totalBalance * 0.8);
    
    addLog(`✓ available balance (80%): ${availableBalance} satoshis`, 'info');
    
    if (availableBalance < requiredBalance) {
        const message = `insufficient balance! need ${requiredBalance} satoshis,but available balance（80%）only ${availableBalance} satoshis。please recharge at least ${Math.ceil(requiredBalance / 0.8)} satoshis。`;
        addLog(`❌ ${message}`, 'error');
        throw new Error(message);
    }
    
    addLog(`✅ balance validation passed`, 'success');
    
    return {
        balance: balance,
        changeAddress: currentAddress
    };
}

// sign transaction
async function signTransaction(preTxRaw, utxoData) {
    addLog('opening Metalet wallet for signing...', 'info');
    console.log('📝 transaction prepared for signing:', {
        preTxRaw: preTxRaw.substring(0, 100) + '...',
        preTxRawLength: preTxRaw.length,
        address: currentAddress
    });
    
    try {
        // check wallet methods
        const wallet = getWallet();
        if (!wallet) {
            throw new Error('Wallet not available');
        }
        
        console.log('🔍 check available wallet methods:', Object.keys(wallet));
        
        // use pay method（TxComposer mode）
        if (typeof wallet.pay === 'function') {
            console.log('📡 use pay method...');
            addLog('use pay method to sign and pay...', 'info');
            
            // get meta-contract library（reference MetaID SDK）
            // library exported as window.metaContract (note lowercase c)
            const metaContract = window.metaContract;
            
            console.log('🔍 check library loading:', {
                metaContract: !!metaContract,
                metaContractKeys: metaContract ? Object.keys(metaContract) : [],
                allWindowKeys: Object.keys(window).filter(k => k.toLowerCase().includes('meta') || k.toLowerCase().includes('mvc') || k.toLowerCase().includes('tx'))
            });
            
            if (!metaContract) {
                throw new Error('meta-contract library not loaded, please refresh page');
            }
            
            // get from metaContract mvc and TxComposer
            const mvc = metaContract.mvc;
            const TxComposer = metaContract.TxComposer;
            
            if (!mvc || !TxComposer) {
                throw new Error('meta-contract library missing mvc or TxComposer');
            }
            
            console.log('✅ meta-contract library loaded');
            console.log('📦 mvc:', mvc);
            console.log('📦 TxComposer:', TxComposer);
            
            // from hex create Transaction（reference MetaID SDK）
            addLog('parse transaction hex...', 'info');
            const tx = new mvc.Transaction(preTxRaw);
            console.log('📦 parse transaction success:', tx);
            
            // create TxComposer
            addLog('create TxComposer...', 'info');
            const txComposer = new TxComposer(tx);
            console.log('🔧 TxComposer createsuccess:', txComposer);
            
            // serialize TxComposer
            addLog('serialize TxComposer...', 'info');
            const txComposerSerialize = txComposer.serialize();
            // console.log('📝 TxComposer serialize result:', txComposerSerialize);
            
            // build pay params
            const payParams = {
                transactions: [
                    {
                        txComposer: txComposerSerialize,
                        message: 'Upload File to MetaID',
                    }
                ],
                feeb: Number(document.getElementById('feeRateInput').value),
            };
            
            console.log('📡 call pay,params:', payParams);
            
            const payResult = await wallet.pay(payParams);
            console.log('✅ pay return result:', payResult);
            
            // handle various possible return formats
            if (payResult) {
                // format 1: payedTransactions array（serialized TxComposer）
                if (payResult.payedTransactions && payResult.payedTransactions.length > 0) {
                    addLog('✅ transaction signed and paid successfully (payedTransactions)', 'success');
                    
                    // deserialize first transaction
                    const payedTxComposerStr = payResult.payedTransactions[0];
                    console.log('📦 payedTransactions[0]:', payedTxComposerStr.substring(0, 100) + '...');
                    
                    // use TxComposer.deserialize() to parse
                    const payedTxComposer = TxComposer.deserialize(payedTxComposerStr);
                    console.log('🔧 deserialize TxComposer:', payedTxComposer);
                    
                    // get signed transaction hex
                    const signedHex = payedTxComposer.getRawHex();
                    console.log('📝 signed transaction hex:', signedHex.substring(0, 100) + '...');
                    
                    addLog(`📝 transaction signed`, 'success');
                    return signedHex;
                } else {
                    throw new Error('signature failed');
                }
            }
        } else {
            throw new Error('signature failed');
        }
    } catch (error) {
        console.error('❌ signature error details:', error);
        throw new Error(`signature failed: ${error.message}`);
    }
}

// commit upload
async function commitUpload(fileId, signedRawTx) {
    addLog('call CommitUpload API, broadcast transaction to blockchain...', 'info');
    
    const response = await fetch(`${API_BASE}/api/v1/files/commit-upload`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            fileId: fileId,
            signedRawTx: signedRawTx
        })
    });
    
    if (!response.ok) {
        throw new Error(`HTTP Error: ${response.status}`);
    }
    
    const result = await response.json();
    
    if (result.code !== 0) {
        throw new Error(result.message);
    }
    
    addLog(`✅ CommitUpload success!`, 'success');
    addLog(`📝 TxID: ${result.data.txId}`, 'success');
    addLog(`📊 status: ${result.data.status}`, 'success');
    
    return result.data;
}

// Update progress
function updateProgress(percent, text) {
    progressFill.style.width = percent + '%';
    progressText.textContent = text;
    addLog(`📊 progress: ${percent}% ${text}`, 'info');
}

// Add log
function addLog(message, type = 'info') {
    const logItem = document.createElement('div');
    logItem.className = `log-item log-${type}`;
    logItem.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logContainer.appendChild(logItem);
    logContainer.scrollTop = logContainer.scrollHeight;
}

// Format file size
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

// Show notificationpopup
function showNotification(message, type = 'info') {
    // create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    
    // set icon based on type
    let icon = '💡';
    if (type === 'success') icon = '✅';
    if (type === 'error') icon = '❌';
    if (type === 'warning') icon = '⚠️';
    
    notification.innerHTML = `
        <span class="notification-icon">${icon}</span>
        <span class="notification-message">${message}</span>
        <button class="notification-close" onclick="this.parentElement.remove()">×</button>
    `;
    
    // add to page
    document.body.appendChild(notification);
    
    // 3seconds auto dismiss
    setTimeout(() => {
        notification.classList.add('notification-fade-out');
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 300);
    }, 3000);
    
    console.log(`📢 notification [${type}]: ${message}`);
}

// show buzz section after successful file upload
function showBuzzSection(filePinId) {
    console.log('📝 Showing buzz section for file pinId:', filePinId);
    console.log('📝 buzzSection element:', buzzSection);
    console.log('📝 buzzSection element exists:', !!buzzSection);
    
    // Store file transaction ID for buzz attachment
    window.uploadedFilePinId = filePinId;
    
    // Show buzz section
    if (buzzSection) {
        console.log('📝 buzzSection element found, removing hidden class');
        console.log('📝 buzzSection classes before removal:', buzzSection.className);
        buzzSection.classList.remove('hidden');
        console.log('📝 buzzSection classes after removal:', buzzSection.className);
        console.log('📝 buzzSection style.display:', buzzSection.style.display);
        addLog('📝 Buzz section displayed - you can now send a buzz with file attachment', 'info');
        showNotification('File uploaded successfully! You can now send a buzz with attachment', 'success');
        
        // Update buzz button state
        updateBuzzButton();
    } else {
        console.error('❌ buzzSection element not found!');
        addLog('❌ Cannot find buzz section element', 'error');
    }
}

// update buzz button state
function updateBuzzButton() {
    if (!sendBuzzBtn) return;
    
    const canSendBuzz = walletConnected;
    
    sendBuzzBtn.disabled = !canSendBuzz;
    
    if (!walletConnected) {
        sendBuzzBtn.textContent = '📝 Please connect wallet first';
    } else {
        sendBuzzBtn.textContent = '📝 Send Buzz';
    }
}

// send buzz function
async function sendBuzz() {
    console.log('📝 Starting to send buzz...');
    
    // Validate wallet connection
    if (!walletConnected) {
        showNotification('⚠️ Please connect wallet first', 'warning');
        addLog('❌ Wallet not connected', 'error');
        return;
    }
    
    // Validate content
    const content = buzzContent ? buzzContent.value.trim() : '';
    if (!content) {
        showNotification('⚠️ Please enter buzz content', 'warning');
        addLog('❌ No buzz content entered', 'error');
        return;
    }
    
    const host = buzzHost ? buzzHost.value.trim() : '';
    
    console.log('✅ Validation passed: wallet connected, content provided');
    addLog('Starting to send Buzz...', 'info');
    
    try {
        sendBuzzBtn.disabled = true;
        sendBuzzBtn.textContent = 'Sending...';
        
        showNotification('Starting to send Buzz...', 'info');
        
        // Build buzz body
        const buzzBody = {
            content: content,
            contentType: "application/json;utf-8"
        };
        
        // Add file attachment if available
        if (window.uploadedFilePinId) {
            // Use the stored file extension from when file was selected
            const attachmentUri = `metafile://${window.uploadedFilePinId}${selectedFileExtension}`;
            buzzBody.attachments = [attachmentUri];
            addLog(`📎 Adding file attachment: ${attachmentUri}`, 'info');
            console.log('📎 Attachment URI:', attachmentUri);
            console.log('📎 File extension used:', selectedFileExtension);
        }
        
        
        console.log('📝 Buzz body:', buzzBody);
        addLog(`📝 Buzz content: ${content}`, 'info');
        
        // Create buzz using MetaID SDK
        const buzzResult = await createBuzz(buzzBody, host);
        
        addLog(`✅ Buzz sent successfully! TxID: ${buzzResult.txid}`, 'success');
        showNotification(`🎉 Buzz sent successfully!`, 'success');
        
        // Show buzz success links
        showBuzzSuccessLinks(buzzResult.txid, buzzResult.pinId);
        
        // Reset buzz form
        // resetBuzzForm();
        
    } catch (error) {
        console.error('❌ Failed to send Buzz:', error);
        addLog(`❌ Failed to send Buzz: ${error.message}`, 'error');
        
        if (error.message && error.message.includes('user cancelled')) {
            showNotification('Buzz sending operation cancelled', 'warning');
        } else {
            showNotification('Failed to send Buzz: ' + error.message, 'error');
        }
        
        sendBuzzBtn.disabled = false;
        sendBuzzBtn.textContent = '📝 Send Buzz';
    }
}

// create buzz using MetaID SDK
async function createBuzz(buzzBody, host) {
    addLog('Calling MetaID SDK to create Buzz...', 'info');
    
    try {
        // Check if MetaID SDK is available
        if (!window.Metaid) {
            throw new Error('MetaID SDK not loaded, please refresh the page');
        }
        
        console.log('✅ MetaID SDK loaded');
        console.log('🔍 window.Metaid:', window.Metaid);
        console.log('🔍 window.Metaid methods:', window.Metaid ? Object.keys(window.Metaid) : 'null');
        addLog('✅ MetaID SDK loaded', 'info');
        
        // Try to use MetaID SDK's createPin method
        addLog('Using MetaID SDK to create Buzz...', 'info');
        
        // Check available MetaID SDK methods
        console.log('🔍 Checking MetaID SDK available methods...');
        const availableMethods = Object.keys(window.Metaid || {});
        console.log('🔍 Available MetaID methods:', availableMethods);
        
        // Try to use MetaID SDK's correct API
        if (window.Metaid && typeof window.Metaid === 'function') {
            console.log('🔍 MetaID is a function, trying to initialize...');
        }
        
        // Create MVC connector
        const { MetaletWalletForMvc, mvcConnect } = window.Metaid;
        
        if (!MetaletWalletForMvc || !mvcConnect) {
            throw new Error('MetaID SDK missing required methods');
        }
        
        addLog('Creating MVC wallet connection...', 'info');
        
        // Create wallet instance
        const wallet = await MetaletWalletForMvc.create();
        
        // Connect to MVC network
        const mvcConnector = await mvcConnect({ 
            wallet: wallet, 
            network: 'livenet'
        });
        
        console.log('🔍 mvcConnector:', mvcConnector);
        console.log('🔍 mvcConnector methods:', mvcConnector ? Object.keys(mvcConnector) : 'null');
        
        if (!mvcConnector) {
            throw new Error('Cannot create MVC connector');
        }
        
        // Check if createPin method exists
        if (typeof mvcConnector.createPin !== 'function') {
            throw new Error('mvcConnector does not have createPin method');
        }
        
        addLog('Using createPin method to create Buzz...', 'info');
        console.log('📝 Buzz data:', buzzBody);
        
        // Build createPin data
        const pinData = {
            body: JSON.stringify(buzzBody),
            contentType: 'application/json;utf-8',
            path: host ? host + ':/protocols/simplebuzz' : '/protocols/simplebuzz',
            operation: 'create'
        };
        
        const pinOptions = {
            network: 'livenet',
            feeRate: Number(document.getElementById('feeRateInput').value) || 1,
        };
        
        console.log('📡 Calling createPin, data:', pinData);
        console.log('📡 Calling createPin, options:', pinOptions);
        
        // Use createPin method to create buzz
        const result = await mvcConnector.createPin(pinData, pinOptions);
        
        console.log('✅ createPin result:', result);
        
        if (result && result.txid) {
            addLog(`✅ Buzz created successfully! TxID: ${result.txid}`, 'success');
            // Try to get pinId from result, if not available use txid
            const pinId = result.pinId || result.txid+'i0';
            if (result.pinId) {
                addLog(`📌 PinID: ${result.pinId}`, 'success');
            }
            return { txid: result.txid, pinId: pinId };
        } else {
            throw new Error('Buzz creation failed, no transaction ID returned');
        }
        
    } catch (error) {
        console.error('❌ Failed to create Buzz:', error);
        throw new Error(`Failed to create Buzz: ${error.message}`);
    }
}

// show buzz success links
function showBuzzSuccessLinks(txId, pinId) {
    // Build links HTML
    let linksHtml = '<div style="margin-top: 15px; padding: 15px; background: #e3f2fd; border-radius: 8px; border-left: 4px solid #1976d2;">';
    linksHtml += '<div style="font-weight: bold; margin-bottom: 10px; color: #1976d2;">🎉 Buzz sent successfully! View details:</div>';
    
    if (txId) {
        const txUrl = `https://www.mvcscan.com/tx/${txId}`;
        linksHtml += `
            <div style="margin: 8px 0;">
                <strong>📝 Buzz Transaction ID:</strong> 
                <a href="${txUrl}" target="_blank" style="color: #1976d2; text-decoration: none; word-break: break-all; font-family: monospace;">
                    ${txId}
                </a>
                <button onclick="window.open('${txUrl}', '_blank')" style="margin-left: 10px; padding: 4px 12px; background: #1976d2; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">
                    View Transaction 🔗
                </button>
            </div>
        `;
        addLog(`🔗 Buzz transaction link: ${txUrl}`, 'success');
    }
    
    if (pinId) {
        // Show Buzz链接
        const showBuzzUrl = `https://www.show.now/buzz/${pinId}`;
        linksHtml += `
            <div style="margin: 8px 0;">
                <strong>📱 Show Buzz:</strong> 
                <a href="${showBuzzUrl}" target="_blank" style="color: #28a745; text-decoration: none; word-break: break-all; font-family: monospace;">
                    ${pinId}
                </a>
                <button onclick="window.open('${showBuzzUrl}', '_blank')" style="margin-left: 10px; padding: 4px 12px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">
                    View Buzz 📱
                </button>
            </div>
        `;
        addLog(`🔗 Show Buzz link: ${showBuzzUrl}`, 'success');
        
        // MetaID Manager链接
        const metaidManagerUrl = `https://man.metaid.io/pin/${pinId}`;
        linksHtml += `
            <div style="margin: 8px 0;">
                <strong>🔧 Buzz PinID:</strong> 
                <a href="${metaidManagerUrl}" target="_blank" style="color: #6f42c1; text-decoration: none; word-break: break-all; font-family: monospace;">
                    ${pinId}
                </a>
                <button onclick="window.open('${metaidManagerUrl}', '_blank')" style="margin-left: 10px; padding: 4px 12px; background: #6f42c1; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">
                    View Buzz Pin 🔧
                </button>
            </div>
        `;
        addLog(`🔗 Buzz PinID link: ${metaidManagerUrl}`, 'success');
    }
    
    linksHtml += '</div>';
    
    // Show link above log area
    const container = document.querySelector('.container');
    const logSection = document.querySelector('.log-section');
    
    // Remove old buzz link display (if exists)
    const oldBuzzLinks = document.getElementById('buzzSuccessLinks');
    if (oldBuzzLinks) {
        oldBuzzLinks.remove();
    }
    
    // Add new buzz link display
    const linksDiv = document.createElement('div');
    linksDiv.id = 'buzzSuccessLinks';
    linksDiv.innerHTML = linksHtml;
    container.insertBefore(linksDiv, logSection);
    
    // Show notification
    showNotification('Buzz sent successfully! Click to view details', 'success');
}

// Get file extension with improved detection
function getFileExtension(file) {
    if (!file || !file.name) {
        return '';
    }
    
    const fileName = file.name;
    const mimeType = file.type;
    console.log('🔍 file name:', fileName);
    console.log('🔍 mime type:', mimeType);
    console.log('🔍 file extension:', file.extension);
    
    // Method 1: Use MIME type to determine extension (most reliable)
    if (mimeType) {
        const mimeToExt = {
            'image/jpeg': '.jpg',
            'image/jpg': '.jpg',
            'image/png': '.png',
            'image/gif': '.gif',
            'image/webp': '.webp',
            'image/svg+xml': '.svg',
            'text/plain': '.txt',
            'text/html': '.html',
            'text/css': '.css',
            'text/javascript': '.js',
            'application/javascript': '.js',
            'application/json': '.json',
            'application/pdf': '.pdf',
            'application/zip': '.zip',
            'application/x-tar': '.tar',
            'application/gzip': '.gz',
            'application/x-7z-compressed': '.7z',
            'application/x-rar-compressed': '.rar',
            'video/mp4': '.mp4',
            'video/avi': '.avi',
            'video/mov': '.mov',
            'audio/mp3': '.mp3',
            'audio/wav': '.wav',
            'audio/ogg': '.ogg'
        };
        
        if (mimeToExt[mimeType]) {
            addLog(`🔍 Detected file type from MIME: ${mimeType} → ${mimeToExt[mimeType]}`, 'info');
            return mimeToExt[mimeType];
        }
    }
    
    // Method 2: Smart filename parsing for complex extensions
    const parts = fileName.split('.');
    if (parts.length >= 2) {
        // Handle common multi-part extensions
        const lastPart = parts[parts.length - 1].toLowerCase();
        const secondLastPart = parts[parts.length - 2].toLowerCase();
        
        // Common compressed file patterns
        if (lastPart === 'gz' && (secondLastPart === 'tar' || secondLastPart === 'tgz')) {
            return '.tar.gz';
        }
        if (lastPart === 'bz2' && secondLastPart === 'tar') {
            return '.tar.bz2';
        }
        if (lastPart === 'xz' && secondLastPart === 'tar') {
            return '.tar.xz';
        }
        
        // Handle versioned files (e.g., file.min.js, style.css.map)
        if (lastPart === 'map' && parts.length >= 3) {
            const thirdLastPart = parts[parts.length - 3].toLowerCase();
            if (thirdLastPart === 'css' || thirdLastPart === 'js') {
                return `.${thirdLastPart}.${secondLastPart}.${lastPart}`;
            }
        }
        
        // Handle minified files (e.g., script.min.js, style.min.css)
        if (secondLastPart === 'min' && (lastPart === 'js' || lastPart === 'css')) {
            return `.${secondLastPart}.${lastPart}`;
        }
        
        // Default: use the last extension
        return '.' + lastPart;
    }
    
    // Method 3: No extension found
    addLog(`⚠️ No file extension detected for: ${fileName}`, 'warning');
    return '';
}

// reset buzz form
function resetBuzzForm() {
    if (buzzContent) buzzContent.value = '';
    if (buzzHost) buzzHost.value = '';
    updateBuzzButton();
}

// show successful upload links
function showUploadSuccessLinks(txId, pinId) {
    // build links HTML
    let linksHtml = '<div style="margin-top: 15px; padding: 15px; background: #e8f5e9; border-radius: 8px; border-left: 4px solid #28a745;">';
    linksHtml += '<div style="font-weight: bold; margin-bottom: 10px; color: #28a745;">🎉 upload to chain successful!view details：</div>';
    
    if (txId) {
        const txUrl = `https://www.mvcscan.com/tx/${txId}`;
        linksHtml += `
            <div style="margin: 8px 0;">
                <strong>📝 Transaction ID:</strong> 
                <a href="${txUrl}" target="_blank" style="color: #667eea; text-decoration: none; word-break: break-all; font-family: monospace;">
                    ${txId}
                </a>
                <button onclick="window.open('${txUrl}', '_blank')" style="margin-left: 10px; padding: 4px 12px; background: #667eea; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">
                    View Transaction 🔗
                </button>
            </div>
        `;
        addLog(`🔗 Transaction link: ${txUrl}`, 'success');
    }
    
    if (pinId) {
        const pinUrl = `https://man.metaid.io/pin/${pinId}`;
        linksHtml += `
            <div style="margin: 8px 0;">
                <strong>📌 PinID:</strong> 
                <a href="${pinUrl}" target="_blank" style="color: #667eea; text-decoration: none; word-break: break-all; font-family: monospace;">
                    ${pinId}
                </a>
                <button onclick="window.open('${pinUrl}', '_blank')" style="margin-left: 10px; padding: 4px 12px; background: #764ba2; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">
                    View Pin 🔗
                </button>
            </div>
        `;
        addLog(`🔗 PinLink: ${pinUrl}`, 'success');
    }
    
    linksHtml += '</div>';
    
    // show link above log area
    const container = document.querySelector('.container');
    const logSection = document.querySelector('.log-section');
    
    // remove old link display（if exists）
    const oldLinks = document.getElementById('successLinks');
    if (oldLinks) {
        oldLinks.remove();
    }
    
    // add new link display
    const linksDiv = document.createElement('div');
    linksDiv.id = 'successLinks';
    linksDiv.innerHTML = linksHtml;
    container.insertBefore(linksDiv, logSection);
    
    // Show notification
    showNotification('file uploaded to chain!click to view details', 'success');
}

// reset form
function resetForm() {
    selectedFile = null;
    selectedFileExtension = '';
    fileInput.value = '';
    fileInfo.classList.remove('show');
    dropZone.classList.remove('has-file');
    uploadBtn.disabled = true;
    uploadBtn.textContent = '🚀 Start Upload to Chain';
    progress.classList.remove('show');
    updateProgress(0, '');
    
    // remove success links display
    const successLinks = document.getElementById('successLinks');
    if (successLinks) {
        successLinks.remove();
    }
}

// listen to wallet account change
const wallet = getWallet();
if (wallet && typeof wallet.on === 'function') {
    wallet.on('accountsChanged', async (account) => {
        console.log('📢 wallet account changed:', account);
        
        // compatible with different wallet API versions
        const address = account?.address || account?.mvcAddress || account?.btcAddress;
        
        if (account && address) {
            currentAddress = address;
            addressText.textContent = currentAddress;
            
            // re-calculate and display MetaID
            const metaId = await calculateMetaID(currentAddress);
            const metaidText = document.getElementById('metaidText');
            if (metaidText) {
                metaidText.textContent = metaId;
            }
            
            addLog(`account switched: ${currentAddress}`, 'info');
            addLog(`🔑 MetaID: ${metaId}`, 'info');
            showNotification(`account switched: ${currentAddress.substring(0, 10)}...`, 'info');
            
            // refresh balance
            await fetchAndDisplayBalance();
        } else {
            walletConnected = false;
            walletStatus.textContent = 'Not Connected';
            walletStatus.style.color = '#999';
            walletAddress.classList.add('hidden');
            
            // hide balance info
            const balanceInfo = document.getElementById('balanceInfo');
            if (balanceInfo) {
                balanceInfo.classList.add('hidden');
            }
            
            // show connect button, hide disconnect button
            connectBtn.textContent = 'Connect Metalet Wallet';
            connectBtn.classList.remove('btn-success', 'hidden');
            connectBtn.classList.add('btn-primary');
            connectBtn.disabled = false;
            
            disconnectBtn.classList.add('hidden');
            
            updateUploadButton();
            updateBuzzButton();
            addLog('Wallet disconnected', 'error');
            showNotification('Wallet disconnected', 'warning');
        }
    });
}

