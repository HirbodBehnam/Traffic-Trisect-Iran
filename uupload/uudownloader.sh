#!/bin/bash
FILE_ID_REGEX="/view/([0-9a-zA-Z_.]+)"
FILE_URL_REGEX="(https://s6.uupload.ir/filelink/[0-9a-zA-Z_.]+/[0-9a-zA-Z_.]+)"
while IFS= read -r link; do
    echo "$1 $link"
    if [[ $link =~ $FILE_ID_REGEX ]]; then
        file_id=${BASH_REMATCH[1]}
        result=$(curl 'https://uupload.ir/linkbuild.php' -X POST -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0' -H 'Content-Type: application/x-www-form-urlencoded; charset=UTF-8' -H 'X-Requested-With: XMLHttpRequest' -H 'Origin: https://uupload.ir' -H "Referer: $link" --data-raw "filename=$file_id")
        if [[ $result =~ $FILE_URL_REGEX ]]; then
            file_link=${BASH_REMATCH[1]}
            echo "Got link $file_link"
            sleep 6 # do not get banned
            aria2c "$file_link"
        else
            echo "Cannot find the file link from result"
            echo
            echo "$result"
        fi
    else
        echo "Cannot find link ID of $link"
    fi
done < "$1"