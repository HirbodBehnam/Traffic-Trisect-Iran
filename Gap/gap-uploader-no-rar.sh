#!/bin/bash
# EDIT TOKEN:
TOKEN="YOUR_TOKEN"
# ALSO MAKE SURE THAT CURL and JQ ARE INSTALLED ON YOUR SYSTEM
# Upload each file
for filename in "$@"; do
	echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
	res=$(curl -X POST -H "content-type: multipart/form-data" -H "token: $TOKEN" -F file=@"$filename" --write-out "\n%{http_code}" https://api.gap.im/upload)
	readarray -t ary <<<"$res"
	if [[ ${ary[1]} != 200 ]]; then # check if the upload was successful
		echo "$(tput setaf 1)Error on uploading file $filename : ${ary[1]} $(tput sgr 0)"
		continue
	fi
	path=$(jq -r .path <<<"${ary[0]}")
	echo "$path" >>links.txt # save the link
done
echo "$(tput setaf 2)Done$(tput sgr 0)"
