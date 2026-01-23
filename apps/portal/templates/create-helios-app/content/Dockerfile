FROM node:18-alpine

# Create app directory
WORKDIR /usr/src/app

# Install app dependencies
COPY package*.json ./
RUN npm install

# Bundle app source
COPY . .

# Expose the port defined in the template
EXPOSE ${{ values.port }}

CMD [ "node", "index.js" ]