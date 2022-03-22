FROM node:17
WORKDIR /workspace
RUN npm install -g typescript
COPY ./package.json ./yarn.lock ./
RUN yarn
COPY ./*.ts ./
RUN tsc *.ts
