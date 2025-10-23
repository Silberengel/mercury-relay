const WebSocket = require('ws');

// Create a simple test event
const testEvent = {
  id: "test123456789abcdef",
  pubkey: "test_pubkey_123456789abcdef",
  created_at: Math.floor(Date.now() / 1000),
  kind: 1,
  tags: [],
  content: "Hello Mercury Relay! This is a test note from Node.js",
  sig: "test_signature_123456789abcdef"
};

console.log('Test event:', JSON.stringify(testEvent, null, 2));

// Connect to the relay
const ws = new WebSocket('ws://192.168.222.109:8080');

ws.on('open', function open() {
  console.log('Connected to Mercury Relay!');
  
  // Send a REQ message to subscribe to events
  const reqMessage = ["REQ", "test-sub", {"kinds": [1]}];
  console.log('Sending REQ:', JSON.stringify(reqMessage));
  ws.send(JSON.stringify(reqMessage));
  
  // Send an EVENT message
  const eventMessage = ["EVENT", testEvent];
  console.log('Sending EVENT:', JSON.stringify(eventMessage));
  ws.send(JSON.stringify(eventMessage));
});

ws.on('message', function message(data) {
  console.log('Received:', data.toString());
});

ws.on('error', function error(err) {
  console.error('WebSocket error:', err);
});

ws.on('close', function close() {
  console.log('Connection closed');
});

// Close after 5 seconds
setTimeout(() => {
  console.log('Closing connection...');
  ws.close();
}, 5000);
