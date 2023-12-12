package git

type HookValues struct {
	SiteId string
	StorageHost string
	StorageName string
	StorageToken string
}

var HookTemplate = `#!/bin/bash
set -euo pipefail

echo "hi"

echo "you just pushed to {{.SiteId}}..."

echo "deleting current content"

curl -X "DELETE" -H "AccessKey: {{.StorageToken}}" "https://{{.StorageHost}}/{{.StorageName}}/"

echo "listing files..."

git checkout main

for FILENAME in $(git ls-files);
do
	echo "Uploading $FILENAME of size $(stat --printf='%s' $FILENAME) bytes..."
	sshpass -p {{.StorageToken}} sftp -q -o StrictHostKeyChecking=no {{.StorageName}}@{{.StorageHost}} <<< $"@put ${FILENAME}"
done

echo "all uploaded goodbye"
`
