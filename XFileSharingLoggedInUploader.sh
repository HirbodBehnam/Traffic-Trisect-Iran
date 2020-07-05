#!/bin/bash
#Use this script to upload to the filehosters that use XFILESHARING PRO FILE SHARING SCRIPT (https://sibsoft.net/xfilesharing.html)
#This script is for uploading with registered account
#Login into site with your account. After logon, look for xfss cookie. copy its content here. You must do this every time you want to upload
#Also it must be generated with the same id as server
#If you want to get it on server try running this command and check the cookies:
#curl -v -F op=login -F token=8e2f556adec156926f42f6cc40fbf238 -F rand="" -F login="your_password" -F password="your_password" http://uplod.ir/
#I'm not quite sure what is token, or if it changes for a specific site or not
XFSS="YOUR_TOKEN"
#ALSO MAKE SURE THAT CURL, RAR and JQ ARE INSTALLED ON YOUR SYSTEM
#Check arguments
if [[ "$#" -lt 1 ]]; then
    echo "Please pass file names as argument."
    exit 1
fi
#Remove old files if left and create new one
rm -rf /tmp/XFSUploader
mkdir /tmp/XFSUploader
#Generate the rar command
rarCommand="rar a -M0 -v1G /tmp/XFSUploader/upload.rar" # You can and might change the file size
for arg in "$@"
do
    rarCommand+=" \"$arg\""
done
#rar the files
eval "$rarCommand"
#Upload each file
for filename in /tmp/XFSUploader/*.rar; do
    echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
    res=$(curl -F file=@"$filename" -F sess_id="$XFSS" -F utype=reg -b xfss="$XFSS" "http://s6.uplod.ir/cgi-bin/upload.cgi?upload_type=file&utype=reg") # You can change the url. Use inspect element on main form of upload to find the url
    result=$(jq -r .[0]file_status <<<"$res")
	if [[ $result != "OK" ]] ; then
		echo "$(tput setaf 1)Error on uploading file $filename : $result $(tput sgr 0)"
		continue
	fi
	token=$(jq -r .[0].file_code <<<"$res")
	base=$(basename "$filename")
	echo "/$base/$token" >> links.txt
    rm "$filename" #Remove the file if it is uploaded
done
echo "$(tput setaf 2)Done$(tput sgr 0)"