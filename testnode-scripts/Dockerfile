FROM node:17
WORKDIR /workspace
COPY ./package.json ./yarn.lock ./
RUN yarn
COPY ./*.ts ./tsconfig.json ./
RUN yarn build
