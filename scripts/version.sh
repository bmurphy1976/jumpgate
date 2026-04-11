#!/bin/sh

set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

release_version() {
	version=$(tr -d ' \t\r\n' < "$ROOT/VERSION")
	if [ -z "$version" ]; then
		echo "VERSION is empty" >&2
		exit 1
	fi
	printf '%s\n' "$version"
}

git_dir() {
	if [ -d "$ROOT/.git" ]; then
		printf '%s\n' "$ROOT/.git"
		return 0
	fi

	if [ -f "$ROOT/.git" ]; then
		path=$(sed -n 's/^gitdir: //p' "$ROOT/.git")
		case "$path" in
			"")
				return 1
				;;
			/*)
				printf '%s\n' "$path"
				;;
			*)
				printf '%s\n' "$ROOT/$path"
				;;
		esac
		return 0
	fi

	return 1
}

commit_hash() {
	if ! dir=$(git_dir); then
		return 0
	fi

	if [ ! -f "$dir/HEAD" ]; then
		return 0
	fi

	head=$(tr -d '\r\n' < "$dir/HEAD")
	sha=""

	case "$head" in
		"ref: "*)
			ref=${head#ref: }
			if [ -f "$dir/$ref" ]; then
				sha=$(tr -d '\r\n' < "$dir/$ref")
			elif [ -f "$dir/packed-refs" ]; then
				sha=$(awk -v ref="$ref" '$2 == ref { print $1; exit }' "$dir/packed-refs")
			fi
			;;
		*)
			sha=$head
			;;
	esac

	if [ -n "$sha" ]; then
		printf '%s\n' "$sha" | cut -c1-7
	fi
}

service_version() {
	release=$(release_version)
	commit=$(commit_hash)
	if [ -n "$commit" ]; then
		printf '%s+%s\n' "$release" "$commit"
		return 0
	fi
	printf '%s\n' "$release"
}

next_version() {
	cd "$ROOT"

	prefix=$(date +%Y.%m)
	patch=$(
		git tag -l "v$prefix.*" |
			sed "s/^v$prefix\\.//" |
			awk '
				BEGIN { max = -1 }
				/^[0-9]+$/ && $1 > max { max = $1 }
				END {
					if (max < 0) {
						print 0
					} else {
						print max + 1
					}
				}
			'
	)

	printf '%s.%s\n' "$prefix" "$patch"
}

ldflags() {
	release=$(release_version)
	commit=$(commit_hash)
	printf '%s\n' "-X dashboard/internal/buildinfo.ReleaseVersion=$release -X dashboard/internal/buildinfo.Commit=$commit"
}

case "${1:-service}" in
	release)
		release_version
		;;
	commit)
		commit_hash
		;;
	service)
		service_version
		;;
	next)
		next_version
		;;
	ldflags)
		ldflags
		;;
	*)
		echo "usage: $0 [release|commit|service|next|ldflags]" >&2
		exit 1
		;;
esac
