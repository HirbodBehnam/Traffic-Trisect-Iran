#!/bin/bash
#EDIT TOKEN:
TOKEN="YOUR_TOKEN"
if [[ "$#" -lt 1 ]]; then #Check number of arguemnt
    echo "Please pass file names as argument."
    exit 1
fi
FILE_NAME="$1"
#ALSO MAKE SURE THAT CURL, RAR and JQ ARE INSTALLED ON YOUR SYSTEM
upload() {
	#$1 is filename
	res=$(curl -X POST "https://bot.sapp.ir/$TOKEN/uploadFile" -H 'content-type: multipart/form-data' -F file=@"$1")
	local ok
	ok=$(jq -r .resultMessage <<<"$res")
	if [[ "$ok" != "OK" ]]; then
		echo "Error on file $1:"
		jq .description <<<"$res"
		touch /tmp/SoroushUploader/.nodelete
		return
	fi
	local id
	id=$(jq -r .fileUrl <<<"$res")
	echo "https://bot.sapp.ir/$TOKEN/downloadFile/$id" >>"uploaded_files.txt"
	local fname
	fname=$(basename "$1")
	echo "bot.sapp.ir/$TOKEN/downloadFile/$id:$fname" >>"file_names.txt"
	echo "IDMan.exe /d https://bot.sapp.ir/$TOKEN/downloadFile/$id /f $fname /a" >> idm.bat
}
#Lets start
rm -rf /tmp/SoroushUploader
#At first rar the file and split it into a temp directory
mkdir /tmp/SoroushUploader
rar a -M0 -v100M /tmp/SoroushUploader/u.rar "$FILE_NAME" # You can also change the chunk size. Max upload size is 100MB
#Then get the file names and upload eachone to server
for filename in /tmp/SoroushUploader/*.rar; do
	while [ "$(jobs | wc -l)" -ge 10 ]; do # Change 10 if needed
		sleep 5
	done
	upload "$filename" &
done
while [[ "$(jobs)" =~ "Running" ]]; do
	sleep 5
done
echo "Done uploading files"
#Delete the files after
[ ! -f /tmp/SoroushUploader/.nodelete ] && rm -rf /tmp/SoroushUploader
