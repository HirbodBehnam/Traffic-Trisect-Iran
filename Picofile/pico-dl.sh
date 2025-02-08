#!/bin/bash
while read -r LINK; do
	NUMS=(${LINK//[!0-9]/ })
	ID="${NUMS[1]}"
	echo "Downloading $LINK with file ID $ID"
	DL_LINK=$(curl "https://s32.picofile.com/file/generateDownloadLink?fileId=$ID" \
	-X 'POST' \
	-H 'accept: */*' \
	-H 'accept-language: en-US,en;q=0.9,fa;q=0.8' \
	-H 'content-length: 0' \
	-H 'origin: https://s32.picofile.com' \
	-H 'priority: u=1, i' \
	-H "referer: $LINK" \
	-H 'sec-ch-ua: "Chromium";v="128", "Not;A=Brand";v="24", "Microsoft Edge";v="128"' \
	-H 'sec-ch-ua-mobile: ?0' \
	-H 'sec-ch-ua-platform: "Windows"' \
	-H 'sec-fetch-dest: empty' \
	-H 'sec-fetch-mode: cors' \
	-H 'sec-fetch-site: same-origin' \
	-H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0' \
	-H 'x-requested-with: XMLHttpRequest')
	echo "Got $DL_LINK"
	wget "$DL_LINK"
done < links.txt
