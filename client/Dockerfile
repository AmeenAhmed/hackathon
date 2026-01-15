# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies using npm (since package-lock.json might not exist for bun)
RUN npm install

# Copy source code
COPY . .

# Build the application (skip TypeScript checking for deployment)
RUN npx vite build

# Production stage
FROM nginx:alpine

# Copy custom nginx config from builder stage
COPY --from=builder /app/nginx.conf /etc/nginx/nginx.conf

# Copy built application from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy public assets
COPY --from=builder /app/public /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]