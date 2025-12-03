#!/usr/bin/env bash
set -xeuo pipefail

dnf install -y epel-release
dnf config-manager --enable crb -y
dnf install -y https://rpms.remirepo.net/enterprise/remi-release-9.rpm
dnf update -y
dnf upgrade -y

dnf module enable -y php:remi-8.1
dnf install -y \
	procps-ng \
	wget \
	nginx \
	git \
	php \
	php-cli \
	php-imap \
	php-intl \
	php-soap \
	php-process \
	php-mbstring \
	php-opcache \
	libmemcached-devel \
	php-pecl-memcached

systemctl enable nginx