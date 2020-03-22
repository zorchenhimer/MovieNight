#
# ------ building mmovienight from source ------
#

FROM golang:1.13-alpine AS build

WORKDIR /app

RUN apk add alpine-sdk

COPY . .

RUN make



#
# ------ creating image to run movienight ------
#

FROM alpine:latest

WORKDIR /app

VOLUME /config

COPY --from=build /app /app
COPY --from=build /app/settings_example.json /config/settings.json

RUN chmod +x /app/docker/start.sh

EXPOSE 8089
EXPOSE 1935

CMD ["/app/docker/start.sh"]