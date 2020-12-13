#!/bin/bash
#Use this script to upload to the filehosters that use XFILESHARING PRO FILE SHARING SCRIPT (https://sibsoft.net/xfilesharing.html)
#This script is for uploading without registered account (as guest)
#ALSO MAKE SURE THAT CURL, RAR and JQ ARE INSTALLED ON YOUR SYSTEM
URL_BASE="http://upload.ir"
#Check arguments
if [[ "$#" -lt 1 ]]; then
	echo "Please pass file names as argument."
	exit 1
fi
#Remove old files if left and create new one
rm -rf /tmp/XFSUploader
mkdir /tmp/XFSUploader
#Generate the rar command
rarCommand="rar a -M0 -v1G /tmp/XFSUploader/$1.rar" # You can and might change the file size
shift
for arg in "$@"; do
	rarCommand+=" \"$arg\""
done
#rar the files
eval "$rarCommand"
#Upload each file
for filename in /tmp/XFSUploader/*.rar; do
	echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
	res=$(curl -F file=@"$filename" -F utype=anon "http://s6.uplod.ir/cgi-bin/upload.cgi?upload_type=file&utype=reg") # You can change the url. Use inspect element on main form of upload to find the url
	result=$(jq -r .[0].file_status <<<"$res")
	if [[ $result != "OK" ]]; then
		echo "$(tput setaf 1)Error on uploading file $filename : $result $(tput sgr 0)"
		continue
	fi
	token=$(jq -r .[0].file_code <<<"$res")
	base=$(basename "$filename")
	echo "$URL_BASE/$base/$token" >>links.txt
	rm "$filename" #Remove the file if it is uploaded
done
echo "$(tput setaf 2)Done$(tput sgr 0)"
