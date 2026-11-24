# hubx-mfe image: the Vite build served by nginx, which forwards /api to the gateway (same origin, no CORS).
# Build context is hubx-mfe/ only: services never share code. GATEWAY_URL is resolved at container start.
FROM node:24.11-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.29-alpine
# The official image renders /etc/nginx/templates/*.template with envsubst on start.
COPY <<'CONF' /etc/nginx/templates/default.conf.template
server {
    listen 8080;
    root /usr/share/nginx/html;

    location /api/ {
        proxy_pass ${GATEWAY_URL};
        proxy_ssl_server_name on;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        try_files $uri /index.html;
    }
}
CONF
COPY --from=build /src/dist /usr/share/nginx/html
ENV NGINX_ENVSUBST_FILTER=GATEWAY_URL
EXPOSE 8080
