#!/bin/bash
#EDIT TOKEN:
TOKEN="YOUR_TOKEN"
#ALSO MAKE SURE THAT CURL, RAR and JQ ARE INSTALLED ON YOUR SYSTEM
#Check arguments
if [[ "$#" -lt 1 ]]; then
    echo "Please pass file names as argument."
    exit 1
fi
#Remove old files if left and create new one
rm -rf /tmp/GapUploader
mkdir /tmp/GapUploader
#Generate the rar command
rarCommand="rar a -M0 -v500M /tmp/GapUploader/upload.rar" #You can change this command
for arg in "$@"
do
    rarCommand+=" \"$arg\""
done
#rar the files
eval "$rarCommand"
#upload each file
for filename in /tmp/GapUploader/*.rar; do
    echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
    res=$(curl -X POST -H "content-type: multipart/form-data" -H "token: $TOKEN" -F file=@"$filename" --write-out "\n%{http_code}" https://api.gap.im/upload)
    readarray -t ary <<<"$res"
    if [[ ${ary[1]} != 200 ]]; then #Check if the upload was successful
        echo "$(tput setaf 1)Error on uploading file $filename : ${ary[1]} $(tput sgr 0)"
        continue
    fi
    path=$(jq -r .path <<<"${ary[0]}")
    echo "$path" >> links.txt #Save the link
    rm "$filename" #Remove the file if it is uploaded
done
echo "$(tput setaf 2)Done$(tput sgr 0)"
