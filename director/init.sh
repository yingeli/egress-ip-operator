#!/bin/sh

local_gateway=$(ip route list 0/0 | awk '{ print $3}')
ip route add $LOCAL_NETWORK via $local_gateway
if [ $? -ne 0 ]; then
   echo "Failed to add route for "$LOCAL_NETWORK" via "$local_gateway
   exit 1
fi
ip route delete 0/0