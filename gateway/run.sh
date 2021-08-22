#!/bin/sh

/usr/local/bin/egress-ip-phase $EGRESS_IP_NAMESPACE $EGRESS_IP_NAME update "Running"
if [ $? -ne 0 ]; then
   echo "Failed to update status for EgressIP "$EGRESS_IP
   exit 1
fi

pod_ip=$(hostname -i)
iptables -t nat -I POSTROUTING -o eth0 -s 192.168.0.0/16 -j SNAT --to $pod_ip

echo "Running gateway for EgressIP "$EGRESS_IP
/usr/sbin/xl2tpd -c /etc/xl2tpd/xl2tpd.conf -D

#EGRESS_PRIVATE_IP=$(/usr/local/bin/azpip $EGRESS_IP privateip | tail -1)
#if [ $? -ne 0 ]; then
#    exit 1
#fi
# echo $EGRESS_PRIVATE_IP

#LISTEN_ADDR=$(ip -f inet addr show cbr0 | sed -En -e 's/.*inet ([0-9.]+).*/\1/p')
#if [ $? -ne 0 ]; then
#   exit 1
#fi
# echo $LISTEN_ADDR

#EGRESS_PRIVATE_IP=$(ip -f inet addr show eth0 | sed -En -e 's/.*inet ([0-9.]+).*/\1/p')
#if [ $? -ne 0 ]; then
#   exit 1
#fi
# echo $EGRESS_PRIVATE_IP

#iptables -t nat -I POSTROUTING -o eth0 -s 192.168.0.0/16 -j SNAT --to $EGRESS_PRIVATE_IP
#sed -i 's/listen-addr = .*/listen-addr = '$LISTEN_ADDR'/' /etc/xl2tpd/xl2tpd.conf