FROM alpine:3.6
RUN apk add --update --no-cache ca-certificates

WORKDIR /appstore
COPY ./bin/appstore-server .
COPY ./ui ./ui

# If the binary was built without using musl, create a symlink to the
# library that is expected in the binary. 
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
EXPOSE 8080
ENTRYPOINT ["./appstore-server"]
