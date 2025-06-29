// public/sw.js
// 这是 StreamSaver.js 所需的 Service Worker 文件。
// 它的作用是作为一个中间人，接收来自主页面的数据流，并将其写入文件系统。

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', event => {
  event.waitUntil(self.clients.claim())
})

const map = new Map()

// The main message handler for the service worker.
self.onmessage = event => {
  const data = event.data
  const port = event.ports[0]

  switch (data.type) {
    case 'CREATE_WRITABLE_STREAM':
      createWritableStream(data, port)
      break
    case 'PONG':
      // Heartbeat received
      break
    default:
      console.warn(`Unknown message type received: ${data.type}`)
      port.postMessage({ type: 'ERROR', message: `Unknown message type: ${data.type}` })
      port.close()
  }
}

function createWritableStream (data, port) {
  const ws = new WritableStream({
    write (chunk) {
      // Forward every chunk to the main thread
      port.postMessage({ type: 'WRITE_CHUNK', chunk: chunk })
    },
    close () {
      port.postMessage({ type: 'STREAM_CLOSED' })
      port.close()
    },
    abort (err) {
      port.postMessage({ type: 'STREAM_ABORTED', error: err.toString() })
      port.close()
    }
  })

  // Expose the WritableStream to the main thread
  port.postMessage({ type: 'WRITABLE_STREAM_CREATED', writableStream: ws }, [ws])
}
