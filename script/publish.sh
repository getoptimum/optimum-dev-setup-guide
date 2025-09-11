#!/bin/bash

# publish script

#P2P_IP=34.40.4.192
#for i in `seq 1 20`; do 
#    string=$(openssl rand -base64 2000 | head -c 50);  
#    /mnt/storage/Optimum/optimum-dev-setup-guide/grpc_p2p_client/p2p-client -mode=publish -topic=mytopic --addr=${P2P_IP}:33212 -msg="$string"  
#done


PROXY_IP=34.182.119.107
for i in `seq 1 20`; do 
    string=$(openssl rand -base64 2000 | head -c 2000);  
    /mnt/storage/Optimum/mump2p-cli/mump2p  publish  --message="${string}" --topic="mytopic" --service-url="http://${PROXY_IP}:8080"
done
