#!/bin/sh

set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

fail() {
	echo "$1" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		fail "missing required command: $1"
	fi
}

require_command git
require_command gh

cd "$ROOT"

require_clean_worktree() {
	if [ -n "$(git status --short)" ]; then
		fail "release requires a clean working tree"
	fi
}

fetch_main_tags() {
	git fetch origin main --tags
}

version_on_ref() {
	ref=$1
	if git cat-file -e "$ref:VERSION" 2>/dev/null; then
		git show "$ref:VERSION" | tr -d ' \t\r\n'
	fi
}

release_branch_name() {
	version=$1
	printf 'release/%s\n' "$version"
}

tag_name() {
	version=$1
	printf 'v%s\n' "$version"
}

pr_title() {
	version=$1
	printf 'chore: release %s\n' "$version"
}

pr_body() {
	version=$1
	cat <<EOF
Prepare Jumpgate $version for release.

This PR updates VERSION for the next release. The tag and GitHub Release will be created after this PR is approved and merged into main.
EOF
}

find_merged_release_pr() {
	release_branch=$1
	gh pr list --state merged --base main --head "$release_branch" --json number --limit 1 --jq 'if length == 0 then "" else .[0].number end'
}

find_open_release_pr() {
	release_branch=$1
	gh pr list --state open --base main --head "$release_branch" --json number --limit 1 --jq 'if length == 0 then "" else .[0].number end'
}

merged_pr_commit() {
	pr_number=$1
	gh pr view "$pr_number" --json mergeCommit --jq '.mergeCommit.oid // ""'
}

prepare_release() {
	require_clean_worktree
	fetch_main_tags

	version=$("$ROOT/scripts/version.sh" next)
	release_branch=$(release_branch_name "$version")
	merged_pr=$(find_merged_release_pr "$release_branch")
	open_pr=$(find_open_release_pr "$release_branch")

	if [ -n "$merged_pr" ]; then
		fail "release PR #$merged_pr for $release_branch is already merged; use release-publish instead"
	fi

	if [ -n "$open_pr" ]; then
		fail "release PR #$open_pr for $release_branch is already open"
	fi

	if git rev-parse --verify --quiet "refs/heads/$release_branch" >/dev/null; then
		fail "local branch $release_branch already exists"
	fi

	if git ls-remote --exit-code --heads origin "$release_branch" >/dev/null 2>&1; then
		fail "remote branch $release_branch already exists"
	fi

	git checkout -b "$release_branch" origin/main
	printf '%s\n' "$version" > "$ROOT/VERSION"
	git add VERSION
	git commit --allow-empty -m "chore: release $version"
	git push -u origin "$release_branch"
	gh pr create --base main --head "$release_branch" --title "$(pr_title "$version")" --body "$(pr_body "$version")"

	printf 'Prepared release PR for %s on %s\n' "$version" "$release_branch"
}

publish_release() {
	require_clean_worktree
	fetch_main_tags

	version=$(version_on_ref origin/main || true)
	if [ -z "$version" ]; then
		fail "origin/main does not contain VERSION"
	fi

	release_branch=$(release_branch_name "$version")
	tag=$(tag_name "$version")

	if git rev-parse --verify --quiet "refs/tags/$tag" >/dev/null; then
		fail "tag $tag already exists"
	fi

	pr_number=$(find_merged_release_pr "$release_branch")
	if [ -z "$pr_number" ]; then
		fail "no merged release PR found for $release_branch"
	fi

	merge_commit=$(merged_pr_commit "$pr_number")
	if [ -z "$merge_commit" ]; then
		fail "release PR #$pr_number does not have a merge commit"
	fi

	if ! git merge-base --is-ancestor "$merge_commit" origin/main; then
		fail "merge commit $merge_commit is not reachable from origin/main"
	fi

	notes_file=$(mktemp)
	trap 'rm -f "$notes_file"' EXIT
	git show origin/main:RELEASE_NOTES.md > "$notes_file"

	git tag -a "$tag" "$merge_commit" -m "Jumpgate $version"
	git push origin "$tag"
	gh release create "$tag" --title "Jumpgate $version" --notes-file "$notes_file"

	printf 'Published %s from PR #%s\n' "$tag" "$pr_number"
}

case "${1:-}" in
	prepare)
		prepare_release
		;;
	publish)
		publish_release
		;;
	*)
		echo "usage: $0 [prepare|publish]" >&2
		exit 1
		;;
esac
