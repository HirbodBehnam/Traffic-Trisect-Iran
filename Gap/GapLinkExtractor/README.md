
# Gap Link Extractor Bot
This is a simple bot that saves all of the links of the files that is sent to it into a text file.
## Running
At first install the [Gap Bot API](https://github.com/MrMahdi313/GapBot):
```
pip3 install GapBot
```
Then download the [file](https://raw.githubusercontent.com/HirbodBehnam/Traffic-Trisect-Iran/master/GapLinkExtractor/LinkExtractor.py). At the [Gap Dashboard](https://my.gap.im/), create a new bot and set the callback to `http://server_ip:5000/webhook`.

Then run the python file.

Also remember to open the port 5000 on firewall.

Bot saves all links to `links.txt` file.
