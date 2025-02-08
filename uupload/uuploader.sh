#!/bin/bash
HASH="" # TODO: FILL ME!
# Check the arguments
if [[ "$#" -lt 1 ]]; then
  echo "Please pass the file names you want to upload as arguments." >&2
  exit 1
fi
#Upload each file
RESULT_REGEX="<img src=\"css/images/file.png\" style=\"margin-bottom:4px;\" alt=\"([0-9a-zA-Z_.]+)\" />"
for filename in "$@"; do
  echo "$(tput setaf 2)Uploading file: $filename $(tput sgr 0)"
  res=$(curl -x socks5://127.0.0.1:10808 -X POST -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0' -H 'Origin: https://uupload.ir' -H 'Connection: keep-alive' -H 'Referer: https://uupload.ir/' -F hash=$HASH -F __userfile[]=@"$filename" -F ittl=86400 https://s6.uupload.ir/sv_process.php)
  # Get the link of the file
  if [[ $res =~ $RESULT_REGEX ]]; then
    echo "Done uploading $filename"
    echo "https://uupload.ir/view/${BASH_REMATCH[1]}/" >> links.txt
    #rm "$filename" # remove the file if it is uploaded
  else
    echo "Could not find the download link in result"
    echo
    echo "$res"
  fi
done