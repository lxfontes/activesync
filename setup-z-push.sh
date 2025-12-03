#!/usr/bin/env bash
set -xeuo pipefail

ZPUSH_VERSION=2.7.6
ZPUSH_URL=https://github.com/Z-Hub/Z-Push/archive/refs/tags/${ZPUSH_VERSION}.tar.gz

mkdir -p \
    /usr/share/z-push \
    /var/lib/z-push \
    /var/log/z-push

wget -q -O /tmp/zpush.tar.gz ${ZPUSH_URL}
mkdir /tmp/z-push
tar -zxf /tmp/zpush.tar.gz -C /tmp/z-push --strip-components=1

cp -r /tmp/z-push/src/* /usr/share/z-push
rm -rf /tmp/zpush.tar.gz /tmp/z-push

ln -s /usr/share/z-push/z-push-admin.php /usr/local/bin/z-push-admin
ln -s /usr/share/z-push/z-push-top.php /usr/local/bin/z-push-top

chmod 755 /var/lib/z-push /var/log/z-push
chown -R nginx:nginx \
    /usr/share/z-push \
    /var/lib/z-push \
    /var/log/z-push
