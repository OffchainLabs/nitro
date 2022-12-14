FROM node:16
WORKDIR /workspace
COPY ./package.json ./yarn.lock ./
RUN yarn
COPY ./*.ts ./tsconfig.json ./
RUN yarn build
ENTRYPOINT ["node", "index.js"]
