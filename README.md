gothroughthis
=============

An EventSource server in Go

This is a server written in Go to implement Server-Sent events. 
The events are recieved via a redis-client connection
and published to all clients who connect to <server-address>/subscribe/<channel-name>

The redis client connection uses this project : https://github.com/garyburd/redigo
EventSource connections are handled using code modified from this project : https://github.com/antage/eventsource