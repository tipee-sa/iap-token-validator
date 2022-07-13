FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY iap-token-validator /iap-token-validator
USER nonroot:nonroot

ENTRYPOINT ["/iap-token-validator"]
