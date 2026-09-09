# Build stage
FROM node:20-alpine3.21 AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build

# Runtime stage — unprivileged nginx on port 8080 (non-root).
FROM nginxinc/nginx-unprivileged:1.27.4-alpine3.21

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
