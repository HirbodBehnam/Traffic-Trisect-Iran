#!/bin/bash
#EDIT TOKEN:
TOKEN="YOUR_TOKEN"
#ALSO MAKE SURE THAT CURL and JQ ARE INSTALLED ON YOUR SYSTEM
#Check arguments
if [[ "$#" -lt 1 ]]; then
    echo "Please pass file names as argument."
    exit 1
fi
#upload each file
for filename in "$@"; do
    echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
    res=$(curl -X POST -H "content-type: multipart/form-data" -H "token: $TOKEN" -F file=@"$filename" --write-out "\n%{http_code}" https://api.gap.im/upload)
    readarray -t ary <<<"$res"
    if [[ ${ary[1]} != 200 ]]; then #Check if the upload was successful
        echo "$(tput setaf 1)Error on uploading file $filename : ${ary[1]} $(tput sgr 0)"
        continue
    fi
    path=$(jq -r .path <<<"${ary[0]}")
    echo "$path" >> links.txt #Save the link
done
echo "$(tput setaf 2)Done$(tput sgr 0)"