package git

type HookValues struct {
	SiteId        string
	SiteSubDir    string
	StorageUrl    string
	StorageName   string
	StorageToken  string
	PurgeCacheUrl string
	PurgeCacheJwt string
}

var HookTemplate = `#!/usr/bin/env bash
set -euo pipefail

echo "hi..."

echo "you just pushed to {{.SiteId}}!"

while read oldrev newrev refname
do
  branch=$(git rev-parse --symbolic --abbrev-ref $refname)
done

echo "you just pushed to branch $branch!"

git config core.bare false

cd ..

# https://stackoverflow.com/questions/10507942
GIT_DIR=".git" git checkout "$branch"

if [ -f "toast.yml" ]; then
  echo "file toast.yml found, attempting to build site..."
  toast build
else
  echo "file toast.yml not found, assuming site is raw HTML..."
fi

echo "uploading files to CDN..."

cd {{.SiteSubDir}}

lftp -u {{.StorageName}},{{.StorageToken}} -e "mirror --reverse --parallel=4 --verbose --delete .; bye" {{.StorageUrl}}

echo "all uploaded, now purging CDN cache..."

curl --silent "{{.PurgeCacheUrl}}" -H "content-type: application/json" -H "Authorization: {{.PurgeCacheJwt}}" -d '{"siteid": "{{.SiteId}}"}'

# todo, also need a curl here to supabase to update last_updated_at and deployed_sha

echo "cache purged, now cleaning up local files..."

rm -rf /tmp/{{.SiteId}}

echo "all cleaned up, goodbye"
`
