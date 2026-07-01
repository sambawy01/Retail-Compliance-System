import '@testing-library/jest-dom/vitest'

// Polyfill matchMedia (not available in jsdom)
if (!window.matchMedia) {
  window.matchMedia = () => ({
    matches: false,
    media: '',
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}

// Stub WebSocket (not available in jsdom)
global.WebSocket = class {
  constructor() {}
  close() {}
  send() {}
  set onopen(_) {}
  set onmessage(_) {}
  set onclose(_) {}
  set onerror(_) {}
}

// Stub RTCPeerConnection (not available in jsdom)
global.RTCPeerConnection = class {
  constructor() {}
  close() {}
  createOffer() { return Promise.resolve({ sdp: '', type: 'offer' }) }
  setLocalDescription() { return Promise.resolve() }
  setRemoteDescription() { return Promise.resolve() }
  addTransceiver() {}
  set ontrack(_) {}
}