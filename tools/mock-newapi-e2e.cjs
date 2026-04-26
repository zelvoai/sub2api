const http = require('http')

const port = Number(process.env.MOCK_NEWAPI_PORT || 18089)
const host = process.env.MOCK_NEWAPI_HOST || '127.0.0.1'

const models = [
  { id: 'gpt-4o-mini-e2e', object: 'model', owned_by: 'openai' },
  { id: 'claude-3-5-haiku-e2e', object: 'model', owned_by: 'anthropic' },
]

const server = http.createServer((req, res) => {
  if (req.url === '/v1/models') {
    res.writeHead(200, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ object: 'list', data: models }))
    return
  }
  res.writeHead(404, { 'content-type': 'application/json' })
  res.end(JSON.stringify({ error: 'not found' }))
})

server.listen(port, host)
