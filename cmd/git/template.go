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

var HookTemplate = `#!/bin/bash
set -euo pipefail

echo "hi..."

echo "you just pushed to {{.SiteId}}, setting up git-ftp..."

git config git-ftp.url "{{.StorageUrl}}"
git config git-ftp.user "{{.StorageName}}"
git config git-ftp.password "{{.StorageToken}}"
git config git-ftp.syncroot "{{.SiteSubDir}}"

echo "checking out main branch..."

git checkout main

if [ -f "toast.yml" ]; then
   echo "file toast.yml found, attempting to build site..."
   toast build
else
   echo "file toast.yml not found, assuming site is raw HTML..."
fi

echo "uploading files to CDN..."

git ftp push --auto-init

echo "all uploaded, now purging CDN cache..."

# todo, quiet curl

curl "{{.PurgeCacheUrl}}" -H "content-type: application/json" -H "Authorization: {{.PurgeCacheJwt}}" -d '{"siteid": "{{.SiteId}}"}'

# todo, also need a curl here to supabase to update last_updated_at and deployed_sha

echo "cache purged, now cleaning up local files..."

rm -rf /tmp/{{.SiteId}}

echo "all cleaned up, goodbye"
`
