#!/bin/sh

/usr/local/bin/egress-ip-phase $EGRESS_IP_NAMESPACE $EGRESS_IP_NAME update "Configuring"
if [ $? -ne 0 ]; then
   echo "Failed to update status for EgressIP "$EGRESS_PUBLIC_IP
   exit 1
fi

local_ip=$(ip -f inet addr show eth0 | sed -En -e 's/.*inet ([0-9.]+).*/\1/p')
if [ $? -ne 0 ]; then
   exit 1
fi

echo "Local IP is "$local_ip". Waiting for configuring EgressIP "$EGRESS_PUBLIC_IP
/usr/local/bin/egress-ip-phase $EGRESS_IP_NAMESPACE $EGRESS_IP_NAME wait "Configured"
if [ $? -ne 0 ]; then
   echo "Failed to wait for configuring EgressIP "$EGRESS_PUBLIC_IP
   exit 1
fi

echo "Successfully confgiured EgressIP "$EGRESS_PUBLIC_IP" on "$local_ip
