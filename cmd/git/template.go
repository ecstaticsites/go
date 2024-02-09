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

echo "uploading files to CDN..."

git ftp push --auto-init

echo "all uploaded, now purging CDN cache..."

curl "{{.PurgeCacheUrl}}" -H "Authorization: {{.PurgeCacheJwt}}" -d '{"siteid": "{{.SiteId}}"}'

echo "cache purged, now cleaning up local files..."

rm -rf /tmp/{{.SiteId}}

echo "all cleaned up, goodbye"
`
