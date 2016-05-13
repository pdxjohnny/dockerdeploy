FROM scratch

WORKDIR /app

COPY ./dockerdeploy_linux-amd64 /app/run

ENTRYPOINT ["/app/run"]
