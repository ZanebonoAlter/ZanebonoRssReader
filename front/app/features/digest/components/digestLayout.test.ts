import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const digestListView = readFileSync(resolve('app/features/digest/components/DigestListView.vue'), 'utf-8')
const digestDetail = readFileSync(resolve('app/features/digest/components/DigestDetail.vue'), 'utf-8')

describe('digest layout structure', () => {
  it('uses a bounded desktop grid with independent left and center scroll areas', () => {
    expect(digestListView).toContain('class="digest-main-grid')
    expect(digestListView).toContain('xl:h-[calc(100vh-18rem)]')
    expect(digestListView).toContain('class="digest-column digest-column--meta')
    expect(digestListView).toContain('class="digest-column digest-column--list')
    expect(digestListView).toContain('class="digest-column__scroll')
    expect(digestListView).toContain('overflow-y-auto')
  })

  it('keeps the detail pane bounded with separate internal scroll regions', () => {
    expect(digestDetail).toContain('class="digest-detail-shell min-h-0')
    expect(digestDetail).toContain('class="digest-detail-main')
    expect(digestDetail).toContain('class="digest-detail-scroll digest-detail-scroll--summary')
    expect(digestDetail).toContain('class="digest-detail-scroll digest-detail-scroll--articles')
    expect(digestDetail).toContain('overflow-y-auto')
  })
})
