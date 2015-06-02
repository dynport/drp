# drp

Docker Reverse Proxy

## Usage
	# check the current configuration (8001 is the admin port)
	$ curl 172.17.42.1:8001 -s
	{}

	# set a new configuration for the root path "/" (all requests): forward all requests to elasticsearch
	$ curl -X POST -d '{"address": "http://172.17.42.1:9200", "path": "/"}' -s 172.17.42.1:8001
	{"address":"http://172.17.42.1:9200","path":"/"}

	# check the current configuration
	$ curl 172.17.42.1:8001 -s
	{"/":{"address":"http://172.17.42.1:9200","path":"/"}}

	# test the configuration (8000 is the proxy port)
	$ curl 127.0.0.1:8000
	{
	  "status" : 200,
	  "name" : "Fin",
	  "cluster_name" : "elasticsearch",
	  "version" : {
		"number" : "1.5.2",
		"build_hash" : "62ff9868b4c8a0c45860bebb259e21980778ab1c",
		"build_timestamp" : "2015-04-27T09:21:06Z",
		"build_snapshot" : false,
		"lucene_version" : "4.10.4"
	  },
	  "tagline" : "You Know, for Search"
	}
