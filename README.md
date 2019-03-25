# Wirey

Tool to manage local [wireguard](https://www.wireguard.com/) interfaces in a distributed system.

By using a remote distributed backend, wirey can synchronize wireguard peers among a cluster of machines
in order to let them share the same tunnel without having to manually configure them by hand.

Each machine should be able to see the same distributed backend in order to join the pool.

## Implemented backends

- etcd
- consul
- http(s) - with optional basic auth

### ETCD

The etcd backend is useful when you want to use etcd to synchronize wireguard peers.

Example usage:

- endpoint: the listen ip address on the current machine
- ipaddr: the ip address you want to assign to the interface
- etcd comma seprated list of etcd servers

```bash
./bin/wirey --endpoint 192.168.33.11 --ipaddr 172.30.0.4 --etcd 192.168.33.10:2379
```

### CONSUL

The consul backend is useful when you want to use consul to synchronize wireguard peers.

Example usage:

- endpoint: the listen ip address on the current machine
- ipaddr: the ip address you want to assign to the interface
- consul ip from the consul server

```bash
./bin/wirey --endpoint 192.168.33.11 --ipaddr 172.30.0.4 --consul 192.168.33.10
```

### HTTP(s) with optional basic auth

The http backend is useful when you want to write your own implementation.

The only suppported auth mechanism for now is Basic Authentication.

Example usage:

- endpoint: the listen ip address on the current machine
- ipaddr: the ip address you want to assign to the interface
- http: the http endpoint where to reach the server without trailing slash (/)
- httpbasicauth: username and password to use if the server implements basic auth, in the form `username:password`

```bash
./bin/wirey --endpoint 192.168.33.12 --ipaddr 10.30.0.80 --http http://192.168.33.10:8080 --httpbasicauth "time:series"
```

Example usage using env variables:

```bash
export WIREY_ENDPOINT="192.168.33.12"
export WIREY_IPADDR="10.30.0.80"
export WIREY_HTTP="http://192.168.33.10:8080"
export WIREY_HTTPBASICAUTH="time:series"
./bin/wirey
```

#### HTTP Server endpoints
You can find an example of http server in [examples/httpbackend](examples/httpbackend)

Starting from the endpoint you provide you provide to wirey, the expected routes are:

#### POST `/{ifname}/{publickeysha}`

**URL parameters:**

- ifname: interface name, wirey defaults to `wg0`
- publickeysha: the sha256 of the public key, this is just used as a key and as of now it's not matched with anything in `wirey` since the real public key is embedded in the body.

**URL Example:**

```
https://myservice.com/wireguard-discovery/wg0/234sfkske03kdssk32
```

**Request Body example:**

```json
{
    "Endpoint": "192.168.33.11:2345",
    "IP": "10.30.0.10",
    "PublicKey": "T053azhMRW1sV2tQbjVISUgycnZtQWt5bDdKN3hJL3IwMjhDWG1zNVRpbz0K"
}
```

**Expected status codes:**

- 201 Created
- 401 Unauthorized (for basic auth)

#### GET `/{ifname}`

**URL Example:**

```
https://myservice.com/wireguard-discovery/wg0
```

**URL parameters:**

- ifname: interface name, wirey defaults to `wg0`

**Description:**

Returns all the peers for the provided interface.


**Expected status codes:**

- 200 OK
- 401 Unauthorized (for basic auth)

**Response body example:**

```json
[
    {
        "Endpoint": "192.168.33.11:2345",
        "IP": "10.30.0.10",
        "PublicKey": "T053azhMRW1sV2tQbjVISUgycnZtQWt5bDdKN3hJL3IwMjhDWG1zNVRpbz0K"
    },
    {
        "Endpoint": "192.168.33.12:2345",
        "IP": "10.30.0.80",
        "PublicKey": "ZlE5a005ZDV1enpGei8xc25STXpnb3U4MVJkYVFmTXczL0NRR2svdEFpRT0K"
    },
    {
        "Endpoint": "192.168.33.13:2345",
        "IP": "10.30.0.60",
        "PublicKey": "WUp2cDFPb0FhTkU5UC9vdlQrb0tIK29XRGtxVDhQenlzZnR1R1p4eEF5OD0K"
    }
]
```


## Local Development

Due to the nature of this project (networking on the root namespace) the easiest way to test if wirey works is by using Vagrant.

A brave person could transpile that to a set of rootless runc containers, or even a set of docker containers with the network namespace transposed from root to the container itself.

BTW, to use vagrant:

The machines available are:

- discovery-server
- net-1
- net-2
- net-3

1. Start the vagrant machines and the sync

```bash
vagrant up
vagrant rsync-auto
```

2. Compile wirey and execute it on both the machines

```bash
make
```

### on net-1

```bash
vagrant ssh net-1
sudo su -
cd /vagrant
./bin/wirey --endpoint 192.168.33.11 --ipaddr 172.30.0.4 --etcd 192.168.33.10:2379
```

### on net-2

```bash
vagrant ssh net-2
sudo su -
cd /vagrant
./bin/wirey --endpoint 192.168.33.12 --ipaddr 172.30.0.5 --etcd 192.168.33.10:2379
```

### on net-3

```bash
vagrant ssh net-2
sudo su -
cd /vagrant
./bin/wirey --endpoint 192.168.33.13 --ipaddr 172.30.0.6 --etcd 192.168.33.10:2379
```

### Verify that the interfaces are up

```bash
vagrant ssh net-1
ping 172.30.0.11
```

Result:
```
PING 172.30.0.11 (172.30.0.11) 56(84) bytes of data.
64 bytes from 172.30.0.11: icmp_seq=1 ttl=64 time=0.414 ms
64 bytes from 172.30.0.11: icmp_seq=2 ttl=64 time=2.54 ms
```

### Check the wg status in a machine

```bash
vagrant ssh net-1
wg show
```

Result:
```
interface: wg0
  public key: 12XP/T4UEfLx6REuFxZWNPrrmrox5xgSRMNExCeNEws=
  private key: (hidden)
  listening port: 2345

peer: 59Je0kMsYkWkQ52Rt7o9Ss60QP3fTcoTQgJgsWDW/QQ=
  endpoint: 192.168.33.12:2345
  allowed ips: 0.0.0.0/0
  latest handshake: 1 minute, 55 seconds ago
  transfer: 820 B received, 764 B sent
```


### Check the etcd store

```bash
vagrant ssh discovery-server
docker exec -e ETCDCTL_API=3 -e ETCDCTL_ENDPOINTS=http://192.168.33.10:2379  -ti etcd etcdctl get --prefix=true /wirey
```

Result:
```
/wirey/wg0/12XP/T4UEfLx6REuFxZWNPrrmrox5xgSRMNExCeNEws=

{"PublicKey":"MTJYUC9UNFVFZkx4NlJFdUZ4WldOUHJybXJveDV4Z1NSTU5FeENlTkV3cz0K","Endpoint":"192.168.33.11:2345","IP":"172.30.0.4"}
/wirey/wg0/59Je0kMsYkWkQ52Rt7o9Ss60QP3fTcoTQgJgsWDW/QQ=

{"PublicKey":"NTlKZTBrTXNZa1drUTUyUnQ3bzlTczYwUVAzZlRjb1RRZ0pnc1dEVy9RUT0K","Endpoint":"192.168.33.12:2345","IP":"172.30.0.11"}
```
