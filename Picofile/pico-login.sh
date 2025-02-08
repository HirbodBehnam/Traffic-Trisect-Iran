#!/bin/bash
# TODO: Fill these values!
USERNAME=""
PASSWORD=""
# Request the login page to get CSRF and stuff
rm /tmp/pico-cookies.txt
login_page=$(curl -c /tmp/pico-cookies.txt -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' "https://www.blogsky.com/login?service=picofile.com&returnurl=https://www.picofile.com/account/logon")
if [[ "$login_page" =~ name=\"__RequestVerificationToken\"\ type=\"hidden\"\ value=\"([A-Za-z0-9_\-]+)\" ]]; then
    REQUEST_VERIFICATION_TOKEN="${BASH_REMATCH[1]}"
else
	echo "Cannot get request verifcation token." >&2
	exit 1
fi
echo "Got $REQUEST_VERIFICATION_TOKEN as request verification token"
# Login
logged_in_page=$(curl -b /tmp/pico-cookies.txt -X POST 'https://www.blogsky.com/login?service=picofile.com&returnurl=https://www.picofile.com/account/logon' -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' -H 'Content-Type: application/x-www-form-urlencoded' --data-raw "__RequestVerificationToken=$REQUEST_VERIFICATION_TOKEN&UserName=$USERNAME&Password=$PASSWORD&Action=%D9%88%D8%B1%D9%88%D8%AF")
if [[ "$logged_in_page" =~ window.parent.location.href\ =\ \"(.+)\" ]]; then
    REDIRECT_LOCATION="${BASH_REMATCH[1]}"
else
    echo "Cannot get redirect location." >&2
	exit 1
fi
echo "Redirecting to $REDIRECT_LOCATION"
curl -L -c /tmp/pico-cookies.txt -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0' "$REDIRECT_LOCATION"
echo "Cookies saved in /tmp/pico-cookies.txt"
cat /tmp/pico-cookies.txt