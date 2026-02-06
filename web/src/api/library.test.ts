import { describe, expect, it } from 'vitest'
import { toCollectionsResponse } from './library'

describe('toCollectionsResponse', () => {
  it('returns an empty collections array when server returns null', () => {
    const result = toCollectionsResponse({
      enabled: true,
      count: 0,
      collections: null,
    })

    expect(result.enabled).toBe(true)
    expect(result.count).toBe(0)
    expect(result.collections).toEqual([])
  })

  it('maps server collections into UI collections', () => {
    const result = toCollectionsResponse({
      enabled: true,
      count: 1,
      collections: [
        {
          id: 'docs',
          name: 'Documentation',
          source_type: 'folder',
          source: '/tmp/docs',
          page_count: 3,
        },
      ],
    })

    expect(result.collections).toEqual([
      {
        id: 'docs',
        name: 'Documentation',
        description: 'folder: /tmp/docs',
        page_count: 3,
        item_count: 3,
      },
    ])
  })
})
