#!/usr/bin/env bash

set -ef

ssl_cert='/certs/domain.crt'

if [[ -f ${ssl_cert} ]]; then
   NGINX_LISTEN_PORT='443 ssl'
   SSL_CONFIG='https_config.conf'
   HTTP_TO_HTTPS_REDIRECT='http_to_https_redirect.conf'
fi

if [[ -n '${USE_CORS_PROXY}' ]]; then
    export CORS_PROXY_CONFIG='cors_proxy.conf'
fi

export NGINX_LISTEN_PORT=${NGINX_LISTEN_PORT:-80} \
       HTTP_TO_HTTPS_REDIRECT=${HTTP_TO_HTTPS_REDIRECT:-empty.conf} \
       SSL_CONFIG=${SSL_CONFIG:-empty.conf} \
       CORS_PROXY_CONFIG=${CORS_PROXY_CONFIG:-empty.conf}

envsubst '$NGINX_LISTEN_PORT $HTTP_TO_HTTPS_REDIRECT $SSL_CONFIG $CORS_PROXY_CONFIG' < '/etc/nginx/conf.d/application.conf.tpl' > '/etc/nginx/conf.d/default.conf'

nginx -g 'daemon off;'
