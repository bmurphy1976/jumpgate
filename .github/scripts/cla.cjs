const fs = require('node:fs')
const path = require('node:path')

const MARKER = '<!-- jumpgate-cla -->'
const LEGACY_COMMENT_MARKER = 'CLA Assistant Lite bot'
const RECHECK_COMMAND = 'recheck'

function normalizeText(value) {
  return String(value || '')
    .trim()
    .replace(/\s+/g, ' ')
    .toLowerCase()
}

function isMatchingPhrase(body, phrase) {
  return normalizeText(body) === normalizeText(phrase)
}

function isRecheckCommand(body) {
  return normalizeText(body) === RECHECK_COMMAND
}

function loadConfig() {
  const workspace = process.env.GITHUB_WORKSPACE || process.cwd()
  const configPath = path.join(workspace, '.github', 'cla.json')
  return JSON.parse(fs.readFileSync(configPath, 'utf8'))
}

function normalizeSignatureRecord(record) {
  const id = Number(record?.id)
  if (!Number.isInteger(id) || id <= 0) {
    return null
  }

  const login = record.login || record.name
  if (!login) {
    return null
  }

  return {
    id,
    login,
    signedAt: record.signedAt || record.created_at || null,
    commentId: record.commentId || record.comment_id || null,
    pullRequestNumber:
      record.pullRequestNumber || record.pullRequestNo || null
  }
}

function normalizeSignaturesPayload(payload) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    throw new Error('CLA signatures file must contain a JSON object')
  }

  if (Array.isArray(payload.signedContributors)) {
    return {
      payload: {
        signedContributors: payload.signedContributors
          .map(normalizeSignatureRecord)
          .filter(Boolean)
      },
      changed: false
    }
  }

  if (Array.isArray(payload.signatures)) {
    return {
      payload: {
        signedContributors: payload.signatures
          .map(normalizeSignatureRecord)
          .filter(Boolean)
      },
      changed: true
    }
  }

  throw new Error(
    'CLA signatures file must contain a signedContributors array'
  )
}

function makeEmptySignaturesPayload() {
  return {
    signedContributors: []
  }
}

function dedupeById(items) {
  const byId = new Map()
  for (const item of items) {
    if (!byId.has(item.id)) {
      byId.set(item.id, item)
    }
  }
  return [...byId.values()]
}

function dedupeContributors(contributors) {
  return dedupeById(contributors)
}

function mergeSignatures(existingRecords, newRecords) {
  return dedupeById([...existingRecords, ...newRecords]).sort(
    (left, right) => left.id - right.id
  )
}

function findManagedComment(comments) {
  return (
    comments.find(comment => String(comment.body || '').includes(MARKER)) ||
    comments.find(comment =>
      String(comment.body || '').includes(LEGACY_COMMENT_MARKER)
    ) ||
    null
  )
}

function formatContributorList(contributors) {
  return contributors
    .map(contributor => `- @${contributor.login}`)
    .join('\n')
}

function formatUnresolvedList(unresolved) {
  return unresolved
    .map(identity => {
      const label = identity.email ? `${identity.name} <${identity.email}>` : identity.name
      return `- ${identity.role}: ${label}`
    })
    .join('\n')
}

function renderComment({
  config,
  unresolved,
  humansMissing,
  botsNeedingOverride
}) {
  const lines = [MARKER]
  const allSatisfied =
    unresolved.length === 0 &&
    humansMissing.length === 0 &&
    botsNeedingOverride.length === 0

  if (allSatisfied) {
    lines.push('CLA requirements satisfied for this pull request.')
    return lines.join('\n\n')
  }

  lines.push('CLA requirements are not yet satisfied for this pull request.')

  if (unresolved.length > 0) {
    lines.push(
      [
        '**Unresolved commit identities**',
        '',
        formatUnresolvedList(unresolved),
        '',
        'Update the commit author or committer identity so GitHub can associate it with a user account, then comment `recheck`.'
      ].join('\n')
    )
  }

  if (humansMissing.length > 0) {
    lines.push(
      [
        '**Human contributors who still need to sign**',
        '',
        formatContributorList(humansMissing),
        '',
        'Comment exactly:',
        '',
        '```text',
        config.signingPhrase,
        '```'
      ].join('\n')
    )
  }

  if (botsNeedingOverride.length > 0) {
    lines.push(
      [
        '**Bot contributors requiring maintainer approval**',
        '',
        formatContributorList(botsNeedingOverride),
        '',
        `A maintainer can add the \`${config.botOverrideLabel}\` label to approve these bot contributions for this pull request.`
      ].join('\n')
    )
  }

  return lines.join('\n\n')
}

function buildFailureMessage({ unresolved, humansMissing, botsNeedingOverride }) {
  if (unresolved.length > 0) {
    return 'CLA check failed because one or more commit identities could not be matched to GitHub users.'
  }
  if (humansMissing.length > 0) {
    return 'CLA check failed because one or more human contributors still need to sign the CLA.'
  }
  return 'CLA check failed because a non-allowlisted bot contributor requires maintainer approval.'
}

function collectContributorIdentities(commits) {
  const contributors = []
  const unresolved = new Map()

  for (const commit of commits) {
    const identities = [
      {
        role: 'author',
        user: commit.author,
        raw: commit.commit?.author
      },
      {
        role: 'committer',
        user: commit.committer,
        raw: commit.commit?.committer
      }
    ]

    for (const identity of identities) {
      if (identity.user?.id) {
        contributors.push({
          id: identity.user.id,
          login: identity.user.login,
          type: identity.user.type || 'User'
        })
        continue
      }

      const name = identity.raw?.name || 'unknown'
      const email = identity.raw?.email || ''
      const key = `${identity.role}:${name}:${email}`
      if (!unresolved.has(key)) {
        unresolved.set(key, {
          role: identity.role,
          name,
          email
        })
      }
    }
  }

  return {
    contributors: dedupeContributors(contributors),
    unresolved: [...unresolved.values()]
  }
}

function findNewSignatureRecords({
  comments,
  contributors,
  signatures,
  signingPhrase,
  pullRequestNumber
}) {
  const contributorsById = new Map(
    contributors
      .filter(contributor => contributor.type !== 'Bot')
      .map(contributor => [contributor.id, contributor])
  )
  const signedIds = new Set(signatures.map(signature => signature.id))
  const newSignatures = []
  const seenIds = new Set()

  for (const comment of comments) {
    if (!isMatchingPhrase(comment.body, signingPhrase)) {
      continue
    }

    const contributor = contributorsById.get(comment.user?.id)
    if (!contributor) {
      continue
    }
    if (signedIds.has(contributor.id) || seenIds.has(contributor.id)) {
      continue
    }

    newSignatures.push({
      id: contributor.id,
      login: contributor.login,
      signedAt: comment.created_at,
      commentId: comment.id,
      pullRequestNumber
    })
    seenIds.add(contributor.id)
  }

  return newSignatures
}

function evaluateContributors({ contributors, signatures, labels, config }) {
  const signatureIds = new Set(signatures.map(signature => signature.id))
  const labelSet = new Set(labels)
  const humansMissing = []
  const botsNeedingOverride = []

  for (const contributor of contributors) {
    const isBot = contributor.type === 'Bot'
    if (isBot) {
      const isAllowlisted = config.botAllowlist.includes(contributor.login)
      const isOverrideApplied = labelSet.has(config.botOverrideLabel)
      if (!isAllowlisted && !isOverrideApplied) {
        botsNeedingOverride.push(contributor)
      }
      continue
    }

    if (!signatureIds.has(contributor.id)) {
      humansMissing.push(contributor)
    }
  }

  return {
    humansMissing,
    botsNeedingOverride,
    allSatisfied:
      humansMissing.length === 0 && botsNeedingOverride.length === 0
  }
}

async function ensureSignaturesBranch(github, context, config) {
  try {
    await github.rest.git.getRef({
      owner: context.repo.owner,
      repo: context.repo.repo,
      ref: `heads/${config.signaturesBranch}`
    })
  } catch (error) {
    if (error.status === 404) {
      throw new Error(
        `CLA signatures branch ${config.signaturesBranch} does not exist.`
      )
    }
    throw error
  }
}

async function loadSignatures(github, context, config) {
  try {
    const response = await github.rest.repos.getContent({
      owner: context.repo.owner,
      repo: context.repo.repo,
      path: config.signaturesPath,
      ref: config.signaturesBranch
    })

    const content = Buffer.from(response.data.content, 'base64').toString('utf8')
    const parsed = JSON.parse(content)
    const normalized = normalizeSignaturesPayload(parsed)

    return {
      sha: response.data.sha,
      payload: normalized.payload,
      changed: normalized.changed
    }
  } catch (error) {
    if (error.status !== 404) {
      throw error
    }

    const payload = makeEmptySignaturesPayload()
    const result = await github.rest.repos.createOrUpdateFileContents({
      owner: context.repo.owner,
      repo: context.repo.repo,
      path: config.signaturesPath,
      branch: config.signaturesBranch,
      message: 'Initialize CLA signatures file',
      content: Buffer.from(JSON.stringify(payload, null, 2) + '\n').toString(
        'base64'
      )
    })

    return {
      sha: result.data.content.sha,
      payload,
      changed: false
    }
  }
}

async function saveSignatures(github, context, config, sha, payload, message) {
  const response = await github.rest.repos.createOrUpdateFileContents({
    owner: context.repo.owner,
    repo: context.repo.repo,
    path: config.signaturesPath,
    branch: config.signaturesBranch,
    sha,
    message,
    content: Buffer.from(JSON.stringify(payload, null, 2) + '\n').toString(
      'base64'
    )
  })

  return response.data.content.sha
}

async function listIssueComments(github, context, issueNumber) {
  return github.paginate(github.rest.issues.listComments, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issueNumber,
    per_page: 100
  })
}

async function listPullRequestCommits(github, context, pullNumber) {
  return github.paginate(github.rest.pulls.listCommits, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: pullNumber,
    per_page: 100
  })
}

async function getPullRequest(github, context) {
  if (context.payload.pull_request) {
    return context.payload.pull_request
  }

  const response = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number
  })
  return response.data
}

async function upsertComment(github, context, issueNumber, body, comments) {
  const existing = findManagedComment(comments)
  if (existing) {
    await github.rest.issues.updateComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      comment_id: existing.id,
      body
    })
    return
  }

  await github.rest.issues.createComment({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issueNumber,
    body
  })
}

async function run({ github, context, core }) {
  const config = loadConfig()

  if (
    context.eventName === 'issue_comment' &&
    !context.payload.issue?.pull_request
  ) {
    core.info('Ignoring non-pull-request issue comment.')
    return
  }

  if (context.eventName === 'issue_comment') {
    const body = context.payload.comment?.body || ''
    if (!isMatchingPhrase(body, config.signingPhrase) && !isRecheckCommand(body)) {
      core.info('Ignoring unrelated pull request comment.')
      return
    }
  }

  await ensureSignaturesBranch(github, context, config)

  const pullRequest = await getPullRequest(github, context)
  const issueNumber = pullRequest.number
  const comments = await listIssueComments(github, context, issueNumber)
  const commits = await listPullRequestCommits(github, context, issueNumber)
  const { contributors, unresolved } = collectContributorIdentities(commits)

  let signaturesState = await loadSignatures(github, context, config)
  let signatures = signaturesState.payload.signedContributors
  let signaturesSha = signaturesState.sha

  if (signaturesState.changed) {
    signaturesSha = await saveSignatures(
      github,
      context,
      config,
      signaturesSha,
      signaturesState.payload,
      'Normalize CLA signatures file schema'
    )
  }

  const newSignatures = findNewSignatureRecords({
    comments,
    contributors,
    signatures,
    signingPhrase: config.signingPhrase,
    pullRequestNumber: issueNumber
  })

  if (newSignatures.length > 0) {
    signatures = mergeSignatures(signatures, newSignatures)
    signaturesSha = await saveSignatures(
      github,
      context,
      config,
      signaturesSha,
      { signedContributors: signatures },
      `Record CLA signatures for pull request #${issueNumber}`
    )
  }

  const evaluation = evaluateContributors({
    contributors,
    signatures,
    labels: (pullRequest.labels || []).map(label => label.name),
    config
  })

  const commentBody = renderComment({
    config,
    unresolved,
    humansMissing: evaluation.humansMissing,
    botsNeedingOverride: evaluation.botsNeedingOverride
  })

  await upsertComment(github, context, issueNumber, commentBody, comments)

  if (unresolved.length > 0 || !evaluation.allSatisfied) {
    core.setFailed(
      buildFailureMessage({
        unresolved,
        humansMissing: evaluation.humansMissing,
        botsNeedingOverride: evaluation.botsNeedingOverride
      })
    )
  }
}

module.exports = {
  MARKER,
  RECHECK_COMMAND,
  buildFailureMessage,
  collectContributorIdentities,
  dedupeContributors,
  evaluateContributors,
  findManagedComment,
  findNewSignatureRecords,
  isMatchingPhrase,
  isRecheckCommand,
  loadConfig,
  mergeSignatures,
  normalizeSignaturesPayload,
  normalizeText,
  renderComment,
  run
}
