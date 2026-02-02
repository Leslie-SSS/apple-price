#!/bin/bash
cd /home/leslie/keepbuild/projects/apple-price
/usr/bin/docker compose -f docker-compose.yml build "$@"
