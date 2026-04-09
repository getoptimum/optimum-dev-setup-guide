Usage of ./p2p_client_multi_streams_publish:
  -count int
    	number of messages to publish (default 1)
  -datasize int
    	size of random of messages to publish (default 100)
  -end-index int
    	index-1 (default 10000)
  -ipfile string
    	file with a list of IP addresses
  -output string
    	file to write the outgoing data hashes
  -poisson
    	Enable Poisson arrival
  -sleep duration
    	optional delay between publishes (e.g., 1s, 500ms) (default 50ms)
  -start-index int
    	beginning index is 0: default 0
  -topic string
    	topic name

# publish 
 ./p2p_client_multi_streams_publish -count 3 -datasize 12222 -end-index 20 -ipfile ip.p2pnode.bnb.tsv  -output outgoing-data-hash.txt -sleep 1s -topic=mytopic


#subscribe
./p2p_client_multi_streams_subscribe -end-index 20 -ipfile ip.p2pnode.bnb.tsv -output-data incoming-data-hash.txt -topic mytopic


# play with the code
