#!/bin/bash
PFAU="" # TODO: FILL ME!
# Check the arguments
if [[ "$#" -lt 1 ]]; then
	echo "Please pass the file names you want to upload as arguments." >&2
	exit 1
fi
# Get the stuff needed from picofile
PANEL=$(curl -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' --cookie ".pfau=$PFAU" https://www.picofile.com/panel)
GUID_REGEX='var guid = "([a-z0-9\-]+)"'
if [[ $PANEL =~ $GUID_REGEX ]]; then
	GUID="${BASH_REMATCH[1]}"
else
	echo "Cannot get GUID" >&2
	exit 1
fi
USERNAME_REGEX='var username = "([A-Za-z0-9\-_]+)"'
if [[ $PANEL =~ $USERNAME_REGEX ]]; then
	USERNAME="${BASH_REMATCH[1]}"
else
	echo "Cannot get username" >&2
	exit 1
fi
UPLOADSERVER_REGEX='var uploadServers = "([A-Za-z0-9\-]+)"'
if [[ $PANEL =~ $UPLOADSERVER_REGEX ]]; then
	UPLOADSERVER="${BASH_REMATCH[1]}"
else
	echo "Cannot get uploadserver" >&2
	exit 1
fi
echo "Logged in as $USERNAME. Uploading to server $UPLOADSERVER"
# Upload each file
FILE_NUMBER=1
for filename in "$@"; do
	echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
	rng=$((1 + RANDOM % 10000))
	res=$(curl --write-out "\n%{http_code}" -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' --cookie ".pfau=$PFAU" -F folderid=0 -F filename="$filename" -F upload=@"$filename" "https://$UPLOADSERVER.picofile.com/file/upload$GUID$rng?uploadkey=${GUID}_$FILE_NUMBER&username=$USERNAME")
	readarray -t ary <<<"$res"
	if [[ ${ary[1]} != 200 ]]; then # check if the upload was successful
		echo "$(tput setaf 1)Error on uploading file $filename : ${ary[*]} $(tput sgr 0)"
		continue
	fi
	rm "$filename" # remove the file if it is uploaded
	# Get the link of the file
	rng=$((1 + RANDOM % 10000))
	curl -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' --cookie ".pfau=$PFAU" "https://$UPLOADSERVER.picofile.com/file/fileuploadinfo$GUID$rng?uploadkey=${GUID}_$FILE_NUMBER&username=$USERNAME&0.1234" | jq -r '"https://" + .server + ".picofile.com/file/" + (.fileId | tostring) + "/" + .name + ".html"' >> "links.txt"
	FILE_NUMBER=$((FILE_NUMBER+1))
done