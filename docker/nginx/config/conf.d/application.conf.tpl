include ${HTTP_TO_HTTPS_REDIRECT};

server {
    listen ${NGINX_LISTEN_PORT};

    include ${SSL_CONFIG};

    root /app/public;

    gzip on;
    gzip_comp_level 2;
    gzip_min_length 1000;
    gzip_proxied expired no-cache no-store private auth;
    gzip_types text/plain application/x-javascript application/javascript text/xml text/css application/xml;

    location / {
        try_files $uri /index.php$is_args$args;
    }

    location ~ ^/index\.php(/|$) {
        include fastcgi_params;
        include ${CORS_PROXY_CONFIG};

        fastcgi_pass php-upstream;
        fastcgi_split_path_info ^(.+\.php)(/.*)$;
        fastcgi_read_timeout 600;

        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        fastcgi_param DOCUMENT_ROOT $realpath_root;

        internal;
    }

    location ~* ^.+\.(?:jpg|jpeg|gif|png|ico|css|zip|tgz|gz|rar|bz2|doc|xls|exe|pdf|ppt|tar|mid|midi|wav|bmp|rtf|js|woff|otf|eot|ttf|svg|pdf|pptx|html)$ {
        expires max;
        add_header Pragma public;
        add_header Cache-Control "public, must-revalidate, proxy-revalidate";
        try_files $uri $uri/ =404;
    }

    location ~ \.php$ {
        return 404;
    }

    error_log /var/log/nginx/error.log;
    access_log /var/log/nginx/access.log;
}
