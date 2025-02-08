import re
import requests
import sys

HASH = "" # TODO: FILL ME!
IMAGE_REGEX = r"<img src=\"css/images/file.png\" style=\"margin-bottom:4px;\" alt=\"([.\w]+)\" />"

payload = (('hash', (None, HASH)),)
for f in sys.argv[1:]:
    payload = payload + (('__userfile[]', open(f, 'rb')),)
payload = payload + (('ittl', (None, 86400)),)

resp = requests.post('https://s6.uupload.ir/sv_process.php', files=payload)
regex_match = re.findall(IMAGE_REGEX, str(resp.content.decode()))
if len(regex_match) == 0:
    print("Cannot find the links. Here is the page content:")
    print(resp.content.decode())
    exit(1)

for link in regex_match:
    print(f"https://uupload.ir/view/{link}/")