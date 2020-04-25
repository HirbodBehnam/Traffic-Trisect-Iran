# api.gap.im has two IP address (so far):
# 77.238.120.242: That the server is in Iran
# 195.201.142.60: That the server is in Germany; You can get this IP with https://www.ipaddressguide.com/ping
# If you run this file, it forces your server to upload to the Germany server.
# You can revert this back by editing /etc/hosts file and removing this line
# Run this script with bash to set the IP address
echo "195.201.142.60 api.gap.im" >> /etc/hosts
