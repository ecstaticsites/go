package git

type HookValues struct {
	SiteId       string
	SiteSubDir   string
	StorageHost  string
	StoragePort  string
	StorageName  string
	StorageToken string
}

var HookTemplate = `#!/bin/bash
set -euo pipefail

echo "hi..."

echo "you just pushed to {{.SiteId}}, setting up git-ftp..."

git config git-ftp.url "ftp://{{.StorageHost}}:{{.StoragePort}}/"
git config git-ftp.user "{{.StorageName}}"
git config git-ftp.password "{{.StorageToken}}"
git config git-ftp.syncroot "{{.SiteSubDir}}"

echo "checking out main branch..."

git checkout main

echo "uploading files to CDN..."

git ftp push --auto-init

echo "pushed, now deleting local files..."

rm -rf /tmp/{{.SiteId}}

echo "all uploaded goodbye"
`
