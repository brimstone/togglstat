FROM brimstone/golang:latest-onbuild as builder

FROM brimstone/debian:sid as cert
RUN package openssl
RUN openssl s_client -connect api.track.toggl.com:443 -showcerts \
            2>&1 </dev/null \
  | tac \
  | awk '/END/ {f=1} f==1 {print} NR>1 && /BEGIN/ {exit}' \
  | tac > /ca.pem

FROM brimstone/debian:sid as tzdata
RUN package tzdata


FROM scratch
COPY --from=builder /app /togglstat
COPY --from=cert /ca.pem /etc/ssl/certs/ca.crt
COPY --from=tzdata /usr/share/zoneinfo /usr/share/zoneinfo
ENV LOG_LEVEL=INFO
ENTRYPOINT ["/togglstat"]
