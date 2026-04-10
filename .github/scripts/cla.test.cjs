const test = require('node:test')
const assert = require('node:assert/strict')

const {
  MARKER,
  collectContributorIdentities,
  evaluateContributors,
  findManagedComment,
  findNewSignatureRecords,
  isMatchingPhrase,
  isRecheckCommand,
  mergeSignatures,
  normalizeSignaturesPayload,
  renderComment
} = require('./cla.cjs')

const config = {
  signingPhrase: 'I have read the CLA Document and I hereby sign the CLA',
  botAllowlist: ['dependabot[bot]', 'github-actions[bot]'],
  botOverrideLabel: 'cla-approved-for-bot'
}

test('matching phrase ignores whitespace and case', () => {
  assert.equal(
    isMatchingPhrase(
      '  I   have read the cla document and i hereby sign the cla  ',
      config.signingPhrase
    ),
    true
  )
})

test('recheck command ignores whitespace and case', () => {
  assert.equal(isRecheckCommand('  ReCheck  '), true)
})

test('legacy signatures payload normalizes to signedContributors', () => {
  const normalized = normalizeSignaturesPayload({
    signatures: [
      {
        id: 7,
        name: 'alice',
        created_at: '2026-04-10T00:00:00Z',
        comment_id: 42,
        pullRequestNo: 5
      }
    ]
  })

  assert.equal(normalized.changed, true)
  assert.deepEqual(normalized.payload, {
    signedContributors: [
      {
        id: 7,
        login: 'alice',
        signedAt: '2026-04-10T00:00:00Z',
        commentId: 42,
        pullRequestNumber: 5
      }
    ]
  })
})

test('collectContributorIdentities dedupes users and reports unresolved identities', () => {
  const result = collectContributorIdentities([
    {
      author: { id: 1, login: 'alice', type: 'User' },
      committer: { id: 1, login: 'alice', type: 'User' },
      commit: {
        author: { name: 'Alice', email: 'alice@example.com' },
        committer: { name: 'Alice', email: 'alice@example.com' }
      }
    },
    {
      author: null,
      committer: { id: 2, login: 'dependabot[bot]', type: 'Bot' },
      commit: {
        author: { name: 'Unknown', email: 'unknown@example.com' },
        committer: { name: 'dependabot[bot]', email: 'bot@example.com' }
      }
    }
  ])

  assert.deepEqual(result.contributors, [
    { id: 1, login: 'alice', type: 'User' },
    { id: 2, login: 'dependabot[bot]', type: 'Bot' }
  ])
  assert.deepEqual(result.unresolved, [
    {
      role: 'author',
      name: 'Unknown',
      email: 'unknown@example.com'
    }
  ])
})

test('findNewSignatureRecords only records current human contributors', () => {
  const records = findNewSignatureRecords({
    comments: [
      {
        id: 10,
        body: config.signingPhrase,
        created_at: '2026-04-10T00:00:00Z',
        user: { id: 1, login: 'alice' }
      },
      {
        id: 11,
        body: config.signingPhrase,
        created_at: '2026-04-10T00:00:01Z',
        user: { id: 2, login: 'not-a-contributor' }
      }
    ],
    contributors: [
      { id: 1, login: 'alice', type: 'User' },
      { id: 3, login: 'dependabot[bot]', type: 'Bot' }
    ],
    signatures: [],
    signingPhrase: config.signingPhrase,
    pullRequestNumber: 99
  })

  assert.deepEqual(records, [
    {
      id: 1,
      login: 'alice',
      signedAt: '2026-04-10T00:00:00Z',
      commentId: 10,
      pullRequestNumber: 99
    }
  ])
})

test('evaluateContributors requires humans to sign even with allowlisted bots', () => {
  const result = evaluateContributors({
    contributors: [
      { id: 1, login: 'alice', type: 'User' },
      { id: 2, login: 'dependabot[bot]', type: 'Bot' }
    ],
    signatures: [],
    labels: [],
    config
  })

  assert.deepEqual(result.humansMissing, [
    { id: 1, login: 'alice', type: 'User' }
  ])
  assert.deepEqual(result.botsNeedingOverride, [])
  assert.equal(result.allSatisfied, false)
})

test('evaluateContributors passes non-allowlisted bot only with override label', () => {
  const withoutLabel = evaluateContributors({
    contributors: [{ id: 4, login: 'renovate[bot]', type: 'Bot' }],
    signatures: [],
    labels: [],
    config
  })
  assert.deepEqual(withoutLabel.botsNeedingOverride, [
    { id: 4, login: 'renovate[bot]', type: 'Bot' }
  ])

  const withLabel = evaluateContributors({
    contributors: [{ id: 4, login: 'renovate[bot]', type: 'Bot' }],
    signatures: [],
    labels: ['cla-approved-for-bot'],
    config
  })
  assert.deepEqual(withLabel.botsNeedingOverride, [])
  assert.equal(withLabel.allSatisfied, true)
})

test('mergeSignatures keeps the first signature record per contributor', () => {
  const merged = mergeSignatures(
    [
      {
        id: 1,
        login: 'alice',
        signedAt: '2026-04-09T00:00:00Z',
        commentId: 1,
        pullRequestNumber: 1
      }
    ],
    [
      {
        id: 1,
        login: 'alice-new',
        signedAt: '2026-04-10T00:00:00Z',
        commentId: 2,
        pullRequestNumber: 2
      }
    ]
  )

  assert.deepEqual(merged, [
    {
      id: 1,
      login: 'alice',
      signedAt: '2026-04-09T00:00:00Z',
      commentId: 1,
      pullRequestNumber: 1
    }
  ])
})

test('findManagedComment prefers marker comment over legacy comment', () => {
  const managed = findManagedComment([
    { id: 1, body: 'Legacy CLA Assistant Lite bot comment' },
    { id: 2, body: `${MARKER}\n\nnew comment` }
  ])

  assert.equal(managed.id, 2)
})

test('renderComment shows human and bot instructions together', () => {
  const body = renderComment({
    config,
    unresolved: [],
    humansMissing: [{ id: 1, login: 'alice', type: 'User' }],
    botsNeedingOverride: [{ id: 2, login: 'renovate[bot]', type: 'Bot' }]
  })

  assert.match(body, /Human contributors who still need to sign/)
  assert.match(body, /alice/)
  assert.match(body, /Bot contributors requiring maintainer approval/)
  assert.match(body, /cla-approved-for-bot/)
})
