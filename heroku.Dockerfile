FROM node:16

RUN npm install heroku -g

WORKDIR /site

ENTRYPOINT ["heroku"]
