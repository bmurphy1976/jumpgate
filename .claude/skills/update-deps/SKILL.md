---
name: "update-deps"
description: "Check this project's dependency versions, compare them to available updates, review release notes, and produce a risk-assessed update plan. Use when the user asks to audit dependency updates or runs /update-deps."
---

# update-deps

Check all project dependencies for available updates and produce a risk-assessed update plan.

## Dependencies

This project has two categories of dependencies:

### Go Modules
- **File:** `go.mod`
- Direct dependencies only (ignore indirect/transitive)

### Front-end Dependencies
- **File:** `common/common.go`
- Versions and CDN URLs are declared as Go `const` values
- Read the file to discover the current constants and their values

## Procedure

### 1. Read current versions
- Read `go.mod` and extract the direct dependency block (the first `require` block)
- Read `common/common.go` and extract the version constants and their current values

### 2. Check for available updates

**Go modules:** Run this command and capture the output:
```
go list -m -u all
```
Lines containing `[vX.Y.Z]` in brackets indicate an available update.

**Front-end dependencies:** Fetch these URLs and extract the `version` field from the JSON response:
- `https://registry.npmjs.org/htmx.org/latest`
- `https://registry.npmjs.org/sortablejs/latest`
- `https://registry.npmjs.org/@mdi/font/latest`

### 3. Assess risk for each update

For every dependency where an update is available:

1. **Classify the version bump:**
   - Patch (e.g., 1.2.3 → 1.2.4) — safe, bug fixes only
   - Minor (e.g., 1.2.3 → 1.3.0) — low risk, new features, backward-compatible
   - Major (e.g., 1.2.3 → 2.0.0) — breaking changes likely

2. **Check for breaking changes:** Fetch release notes for each updated dependency using the URLs below. Fetch all of them **in parallel** to minimize round trips. Summarize anything relevant — especially breaking API changes, removed features, or required migration steps.

   **Release note URLs** (use these exact URLs — do NOT guess repository paths):

   | Dependency | Release notes URL |
   |------------|------------------|
   | github.com/a-h/templ | `https://github.com/a-h/templ/releases` |
   | github.com/labstack/echo/v5 | `https://github.com/labstack/echo/releases` |
   | gopkg.in/yaml.v3 | `https://github.com/go-yaml/yaml/releases` |
   | modernc.org/sqlite | `https://raw.githubusercontent.com/modernc-org/sqlite/master/CHANGELOG.md` |
   | htmx | `https://github.com/bigskysoftware/htmx/releases` |
   | sortablejs | `https://github.com/SortableJS/Sortable/releases` |
   | @mdi/font | `https://github.com/Templarian/MaterialDesign-Webfont/releases` |

   **Important:** Do NOT try to fetch GitLab pages (they use dynamic rendering and return empty content). For modernc.org/sqlite, always use the raw GitHub mirror URL above. If a release page fails to load, use the available web or search capability to look up `"<package> <version> release notes"` as a fallback — do not retry the same URL.

3. **Flag risks clearly:**
   - Major version bumps get an explicit warning
   - Any update with known breaking changes gets details on what would need to change in this codebase
   - Indirect dependency updates (shown by `go list -m -u all` but not in the direct `require` block) should be noted but de-prioritized — they're managed by direct deps

### 4. Present the update plan

Output a markdown table followed by details, release note links, and a recommendation. Example:

| Dependency | Current | Available | Bump | Risk | Notes |
|------------|---------|-----------|------|------|-------|
| github.com/example/libA | v1.2.3 | v1.2.5 | patch | safe | routine update |
| github.com/example/libB | v3.1.0 | v4.0.0 | major | breaking | handler API redesign, see migration guide |
| ExampleLib | 2.0.4 | 2.1.0 | minor | low | new shorthand syntax added |
| ExampleOther | 1.5.0 | 1.5.0 | — | — | up to date |

#### Breaking change details

**github.com/example/libB → v4.0.0**
libB v4 changes the handler signature from `Func(ctx Context) error` to `Func(ctx Context)`. This affects every handler in `handlers/`. Migration would require updating all handler return types and removing explicit error returns.

**Recommendation:** Skip libB v4 for now — the migration is significant. Apply all patch and minor updates.

#### Release notes
For each dependency with an available update, include a direct link to its release notes or changelog so the user can review manually:
- [libA releases](https://github.com/example/libA/releases)
- [libB releases](https://github.com/example/libB/releases)
- [ExampleLib releases](https://github.com/example/ExampleLib/releases)

#### Recommended updates
- Apply: libA v1.2.5, ExampleLib 2.1.0
- Skip: libB v4.0.0 (breaking, requires migration)
- No action: ExampleOther (current)

### 5. Apply updates (only after user approves)

**Go modules:**
```
go get <module>@<version>
go mod tidy
```

**Front-end dependencies:** Update the version constant in `common/common.go`. URLs are built from the version constant via string concatenation, so only the version value needs changing.

**Verify:** Run `make test` after all updates are applied. If tests fail, investigate and report before continuing.
