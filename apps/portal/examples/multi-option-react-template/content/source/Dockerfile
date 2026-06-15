# Build stage
FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
ARG PORT=${{ values.port }}
RUN sed -i "s/listen\(  *\)80;/listen ${PORT};/" /etc/nginx/conf.d/default.conf
EXPOSE ${PORT}
CMD ["nginx", "-g", "daemon off;"]
