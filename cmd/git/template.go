package git

type HookValues struct {
	SiteId       string
	SiteSubDir   string
	StorageUrl   string
	StorageName  string
	StorageToken string
	PostPushUrl  string
	PostPushJwt  string
}

var HookTemplate = `#!/usr/bin/env bash
set -euo pipefail

echo "Welcome to Ecstatic's git interface..."

echo "You just pushed to site ID {{.SiteId}}..."

while read oldrev newrev refname
do
  branch=$(git rev-parse --symbolic --abbrev-ref $refname)
  newsha="$newrev"
done

echo "You just pushed to branch ${branch}...."

git config core.bare false

cd ..

# https://stackoverflow.com/questions/10507942
GIT_DIR=".git" git checkout "${branch}"

if [ -f "devbox.json" ]; then
  echo "File devbox.json found, attempting to build site..."
  devbox install
  devbox run install
  devbox run build
else
  echo "File devbox.json not found, assuming site is raw HTML..."
fi

cd {{.SiteSubDir}}

echo "Deleting all existing files from CDN..."

# ignore errors here; this often returns 400 "directory might not be empty" but it worked
curl -XDELETE --silent --output /dev/null -H 'AccessKey: {{.StorageToken}}' '{{.StorageUrl}}/{{.StorageName}}/' || true

uploadfile() {
	echo "Uploading to CDN file ${1}..."
  curl -XPUT --silent --output /dev/null -H 'AccessKey: {{.StorageToken}}' --data-binary "@${1}" "{{.StorageUrl}}/{{.StorageName}}/${1}"
}

# make the above function available to spawned shells
export -f uploadfile

# https://stackoverflow.com/questions/11003418/calling-shell-functions-with-xargs
find . -not -path '*/.*' -type f | xargs -P 10 -I {} bash -c 'uploadfile "$@"' _ {}

echo "Updating site row metadata and purging CDN cache..."

curl -XPOST --silent --output /dev/null "{{.PostPushUrl}}" -H "Authorization: {{.PostPushJwt}}" --json "{\"siteid\": \"{{.SiteId}}\", \"sha\": \"$newsha\"}"

echo "Cleaning up local files after publish..."

rm -rf /tmp/{{.SiteId}}

echo "All cleaned up, goodbye!"
`
