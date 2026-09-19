FROM ladder-base:client-go-obf
WORKDIR /app
COPY excision.patch /tmp/excision.patch
RUN patch -p1 --forward --batch -i /tmp/excision.patch && rm -f /tmp/excision.patch
