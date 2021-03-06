#!/bin/bash

echo "Preparing certificates"
ls /srv/
cp /srv/postgresql.conf /var/lib/postgresql/data/postgresql.conf
cp /srv/server.crt /var/lib/postgresql/data/server.crt
cp /srv/server.key /var/lib/postgresql/data/server.key
chmod -R 600 /var/lib/postgresql/data/server.crt
chmod -R 600 /var/lib/postgresql/data/server.key
chown -R postgres: /var/lib/postgresql/data/
echo "Everything copied; SSL ready"
