#!/bin/sh

sed -i 's/lns = .*/lns = '$EGRESS_GATEWAY'/' /etc/xl2tpd/xl2tpd.conf
/usr/sbin/xl2tpd -c /etc/xl2tpd/xl2tpd.conf -D

# startup xl2tpd ppp daemon then send it a connect command
# (sleep 3 && echo "c egressgw" > /var/run/xl2tpd/l2tp-control) &
#/usr/sbin/xl2tpd -p /var/run/xl2tpd.pid -c /etc/xl2tpd/xl2tpd.conf -C /var/run/xl2tpd/l2tp-control -D