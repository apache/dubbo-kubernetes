FROM php:8.4-cli-bookworm AS runtime
WORKDIR /app
ARG SERVICE
COPY src/${SERVICE}/index.php /app/index.php
RUN php -l /app/index.php
ENV PORT=8080 PHP_CLI_SERVER_WORKERS=4
USER 65532:65532
EXPOSE 8080
CMD ["sh", "-c", "exec php -S \"0.0.0.0:${PORT}\" -t /app /app/index.php"]
