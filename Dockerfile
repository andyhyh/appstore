FROM alpine:3.6
RUN apk add --update --no-cache wget gcc libc-dev ca-certificates tar

ENV APPSTORE appstore-server
ENV APPSTORE_SERVER_SHA256 6550fa64c90c4745bdc58b8f8015ce87c7f12fb0fb3f027236659d2ff298d5b1

ENV APPSTORE_UI appstore-ui.tar.gz
ENV APPSTORE_UI_SHA256 41f8bfabab0c852b50ac7d7b6c78280f664642f31d987d240ff1d903f45bec6a

RUN wget -q "https://f.128.no/${APPSTORE}"
RUN echo "${APPSTORE_SERVER_SHA256}  ${APPSTORE}" | sha256sum -w -c -
RUN wget -q "https://f.128.no/${APPSTORE_UI}"
RUN echo "${APPSTORE_UI_SHA256}  ${APPSTORE_UI}" | sha256sum -w -c -
RUN tar -zxvf ${APPSTORE_UI} 
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN chmod +x ${APPSTORE}
EXPOSE 8080
ENTRYPOINT ["./appstore-server"]
