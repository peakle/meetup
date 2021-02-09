FROM nginx:1.11-alpine

RUN set -x \
    && addgroup -g 82 -S www-data \
    && adduser -u 82 -D -S -G www-data www-data

COPY config /etc/nginx/
COPY boot.sh /usr/local/bin/boot_nginx

CMD ["sh", "/usr/local/bin/boot_nginx"]
