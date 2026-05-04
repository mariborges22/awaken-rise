# Stage 1: Build Angular
FROM node:20-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Production Nginx (Non-Root)
FROM nginxinc/nginx-unprivileged:alpine

# Copiar os arquivos compilados do Angular
COPY --from=build /app/dist /usr/share/nginx/html

# Copiar a configuração do Nginx (Proxy Reverso)
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080