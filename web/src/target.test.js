import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parseTarget, buildURL } from './target.js'

test('a bare host gets the ClickHouse HTTP port, not port 80', () => {
  assert.deepEqual(parseTarget('101.46.52.153'), {
    scheme: 'http', host: '101.46.52.153', port: '8123',
  })
})

test('a pasted URL is split back into its parts', () => {
  assert.deepEqual(parseTarget('https://ch.example.com:9440'), {
    scheme: 'https', host: 'ch.example.com', port: '9440',
  })
  assert.deepEqual(parseTarget('http://ch.internal'), {
    scheme: 'http', host: 'ch.internal', port: '8123',
  })
  assert.deepEqual(parseTarget('ch.internal:8443'), {
    scheme: 'http', host: 'ch.internal', port: '8443',
  })
})

test('an IPv6 literal is not split on its own colons', () => {
  assert.deepEqual(parseTarget('[2001:db8::1]:8123'), {
    scheme: 'http', host: '[2001:db8::1]', port: '8123',
  })
  assert.deepEqual(parseTarget('http://[::1]'), {
    scheme: 'http', host: '[::1]', port: '8123',
  })
})

test('trailing path or query is ignored rather than smuggled into the host', () => {
  assert.deepEqual(parseTarget('http://ch:8123/play?x=1'), {
    scheme: 'http', host: 'ch', port: '8123',
  })
})

test('nothing sensible in, nothing out', () => {
  for (const bad of ['', '   ', null, undefined, '://']) {
    assert.equal(parseTarget(bad), null, `parseTarget(${JSON.stringify(bad)})`)
  }
})

test('round trips', () => {
  for (const raw of ['https://ch.example.com:9440', 'http://[::1]:8123']) {
    assert.equal(buildURL(parseTarget(raw)), raw)
  }
  assert.equal(buildURL(parseTarget('101.46.52.153')), 'http://101.46.52.153:8123')
})
