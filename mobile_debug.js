// Mobile browser WebSocket debugging script
console.log('=== Mobile Browser WebSocket Debug ===');

// Check WebSocket support
console.log('WebSocket support:', typeof WebSocket !== 'undefined');

// Check if we're on mobile
const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
console.log('Mobile device detected:', isMobile);

// Check user agent
console.log('User Agent:', navigator.userAgent);

// Test WebSocket connection
function testWebSocket() {
    console.log('Testing WebSocket connection to ws://192.168.222.109:8080');
    
    try {
        const ws = new WebSocket('ws://192.168.222.109:8080');
        
        ws.onopen = function() {
            console.log('✅ WebSocket connection opened successfully');
            
            // Send a test message
            const testMsg = JSON.stringify(["REQ", "mobile-debug", {"kinds": [1], "limit": 5}]);
            ws.send(testMsg);
            console.log('📤 Sent test subscription message');
        };
        
        ws.onmessage = function(event) {
            console.log('📥 Received message:', event.data);
        };
        
        ws.onerror = function(error) {
            console.error('❌ WebSocket error:', error);
        };
        
        ws.onclose = function(event) {
            console.log('🔌 WebSocket closed. Code:', event.code, 'Reason:', event.reason);
        };
        
    } catch (error) {
        console.error('❌ Failed to create WebSocket:', error);
    }
}

// Check for common mobile browser issues
function checkMobileIssues() {
    console.log('=== Mobile Browser Issues Check ===');
    
    // Check if we're on HTTPS
    const isHTTPS = location.protocol === 'https:';
    console.log('On HTTPS:', isHTTPS);
    
    // Check if WebSocket is blocked by mixed content policy
    if (isHTTPS && !location.hostname.includes('localhost') && !location.hostname.includes('127.0.0.1')) {
        console.warn('⚠️ HTTPS page trying to connect to HTTP WebSocket - this may be blocked');
    }
    
    // Check for common mobile browser restrictions
    console.log('Navigator online:', navigator.onLine);
    console.log('Connection type:', navigator.connection ? navigator.connection.effectiveType : 'unknown');
    
    // Check if we can make basic HTTP requests
    fetch('http://192.168.222.109:8080')
        .then(response => {
            console.log('✅ HTTP request successful:', response.status);
            return response.text();
        })
        .then(text => {
            console.log('HTTP response:', text);
        })
        .catch(error => {
            console.error('❌ HTTP request failed:', error);
        });
}

// Run checks
checkMobileIssues();
setTimeout(testWebSocket, 1000);
