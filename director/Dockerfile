FROM alpine:latest

RUN apk add --no-cache xl2tpd ppp \
    && mkdir -p /var/run/xl2tpd \
    && touch /var/run/xl2tpd/l2tp-control

COPY xl2tpd.conf /etc/xl2tpd/xl2tpd.conf
COPY options.xl2tpd.client /etc/ppp/options.xl2tpd.client
COPY run.sh /

RUN chmod 755 /run.sh

CMD ["/run.sh"]