FROM golang:1.16 AS build

WORKDIR /app

COPY . .

RUN go build

FROM photon

WORKDIR /app

VOLUME /data

COPY --from=build /app/MovieNight /app
COPY --from=build /app/settings_example.json /data/config/settings.json

EXPOSE 8089
EXPOSE 1935

ENTRYPOINT ["/app/MovieNight", "--config", "/data/config/settings.json", "--static", "/data/static"]