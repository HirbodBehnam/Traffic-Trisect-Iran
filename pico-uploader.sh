#!/bin/bash
PFAU="" # TODO: FILL ME!
# Check the arguments
if [[ "$#" -lt 2 ]]; then
	echo "Please pass the upload rar name as first argument and file names you want to upload as next arguments." >&2
	exit 1
fi
# Remove old files if left and create new one
rm -rf /tmp/PicoUploader
mkdir /tmp/PicoUploader
# Generate the rar command
resultName="$1"
rarCommand="rar a -M0 -v500M /tmp/PicoUploader/$resultName.rar --" # you can change this command
shift
for arg in "$@"; do
	rarCommand+=" \"$arg\""
done
# rar the files
eval "$rarCommand"
# Get the stuff needed from picofile
PANEL=$(curl --cookie ".pfau=$PFAU" https://www.picofile.com/panel)
GUID_REGEX='var guid = "([a-z0-9\-]+)"'
if [[ $PANEL =~ $GUID_REGEX ]]; then
    GUID="${BASH_REMATCH[1]}"
else
    echo "Cannot get GUID" >&2
fi
USERNAME_REGEX='var username = "([A-Za-z0-9\-_]+)"'
if [[ $PANEL =~ $USERNAME_REGEX ]]; then
    USERNAME="${BASH_REMATCH[1]}"
else
    echo "Cannot get username" >&2
fi
UPLOADSERVER_REGEX='var uploadServers = "([A-Za-z0-9\-]+)"'
if [[ $PANEL =~ $UPLOADSERVER_REGEX ]]; then
    UPLOADSERVER="${BASH_REMATCH[1]}"
else
    echo "Cannot get uploadserver" >&2
fi
# Upload each file
FILE_NUMBER=1
for filename in /tmp/PicoUploader/*.rar; do
	echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
    rng=$((1 + RANDOM % 10000))
	res=$(curl --write-out "\n%{http_code}" --cookie ".pfau=$PFAU" -F folderid=0 -F filename="$filename" -F upload=@"$filename" "https://$UPLOADSERVER.picofile.com/file/upload$GUID$rng?uploadkey=${GUID}_$FILE_NUMBER&username=$USERNAME")
	readarray -t ary <<<"$res"
	if [[ ${ary[1]} != 200 ]]; then # check if the upload was successful
		echo "$(tput setaf 1)Error on uploading file $filename : ${ary[*]} $(tput sgr 0)"
		continue
	fi
	rm "$filename" # remove the file if it is uploaded
    # Get the link of the file
    rng=$((1 + RANDOM % 10000))
    curl --cookie ".pfau=$PFAU" "https://$UPLOADSERVER.picofile.com/file/fileuploadinfo$GUID$rng?uploadkey=${GUID}_$FILE_NUMBER&username=$USERNAME&0.1234" | jq '"https://" + .server + ".picofile.com/file/" + (.fileId | tostring) + "/" + .name + ".html"' >> "$resultName.txt"
    FILE_NUMBER=$((FILE_NUMBER+1))
done