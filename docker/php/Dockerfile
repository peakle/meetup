FROM registry.wake-app.net/wakeapp/php:7.4.11-fpm

RUN apt-get update && apt-get install -y \
		libfreetype6-dev \
		libjpeg62-turbo-dev \
		libpng-dev \
		libc-client-dev \
		libkrb5-dev \
		nghttp2 \
	&& docker-php-ext-configure gd --with-freetype=/usr/include/ --with-jpeg=/usr/include/ \
	&& docker-php-ext-install -j$(nproc) gd

RUN cp /usr/local/etc/php/php.ini-development /usr/local/etc/php/php.ini

RUN curl "https://github.com/tideways/php-xhprof-extension/archive/v5.0.2.tar.gz" -fsL -o ./a.tar.gz  && \
     tar xf ./a.tar.gz && \
     cd ./php-xhprof-extension-5.0.2 && \
     phpize && \
     ./configure && \
     make && \
     make install && \
     cd .. && rm -rf ./a.tar.gz ./php-xhprof-extension-5.0.2

COPY ./php.ini /usr/local/etc/php/
