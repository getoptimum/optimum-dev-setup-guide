#!/bin/bash
# subscribe script

#P2P_IP=34.40.44.199
#/mnt/storage/Optimum/optimum-dev-setup-guide/grpc_p2p_client/p2p-client -mode=subscribe -topic=mytopic --addr=${P2P_IP}:33212

PROXY_IP=34.105.5.25
/mnt/storage/Optimum/mump2p-cli/mump2p subscribe --topic="mytopic" --service-url="http://${PROXY_IP}:8080"
