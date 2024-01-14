# simple Load Balancer with Round Robin Algorithm.

RoundRobin algorithm used to send request to the backend server and support retries with up to 3 retries per server.

It also doing healty checks for each servers every 2 minute with simple TCP request and assume its availability.

While its simple, its also performs passive reocvery and cleaning for unhealthy backends.

# How to use

Example:

To add followings as load balanced backends

- http://localhost:3031
- http://localhost:3032
- http://localhost:3033
- http://localhost:3034

```bash
go run main.go --backend=http://localhost:3031,http://localhost:3032,http://localhost:3033,http://localhost:3034
```
