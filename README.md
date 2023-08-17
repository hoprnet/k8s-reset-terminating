# k8s-reset-terminating

Reset resource status from terminating on Ceph resources. 
Forked repository from [k8s-reset-terminating-p](https://github.com/jianz/k8s-reset-terminating-pv)

## Purpose

When delete a Rook Ceph kubernetes resource by accident, it may stuck in the terminating status due to some finalizer prevent it from being deleted. You can use this tool to reset its status back.

## Developing

If you prefer to compile by yourself:

```shell
git clone https://github.com/hoprnet/k8s-reset-terminating.git
cd k8s-reset-terminating
go mod download -x
go get -u -x
go build -o k8s_reset
```

## Usage

```text
Usage:
  k8s_reset [flags] <resource name>

Flags:
      --etcd-ca        string   CA Certificate used by etcd (default "ca.crt")
      --etcd-cert      string   Public key used by etcd (default "etcd.crt")
      --etcd-key       string   Private key used by etcd (default "etcd.key")
      --etcd-host      string   The etcd domain name or IP (default "localhost")
      --etcd-port      int      The etcd port number (default 2379)
      --k8s-key-prefix string   The etcd key prefix for kubernetes resources. (default "registry")
      --k8s-resource-type string   The kubernetes resources type. (default "cephfilesystems")
  -h, --help                    help for k8s_reset
```

For simplicity, you can name the etcd certificate ca.crt, etcd.crt, etcd.key, and put them in the same directory as the tool(k8s_reset).

The tool by default connect to etcd using `localhost:2379`. You should open the ETCD port:

```shell
ssh -L 2379:10.0.0.3:2379 -f vm-stage-bastion -N
```

```shell
./k8s_reset --k8s-resource-type cephfilesystems rook-ceph/ceph-ephimeral
./k8s_reset --k8s-resource-type cephfilesystems rook-ceph/ceph-filesystem
./k8s_reset --k8s-resource-type cephobjectstores rook-ceph/ceph-objectstore
./k8s_reset --k8s-resource-type cephclusters rook-ceph/rook-ceph

```


etcdctl Sample commands

```shell
ssh vm-stage-k8s-master-1
export ETCDCTL_API=3
export ETCDCTL_CACERT=/etc/ssl/etcd/ssl/ca.pem
export ETCDCTL_CERT=/etc/ssl/etcd/ssl/node-vm-stage-k8s-master-1.pem
export ETCDCTL_KEY=/etc/ssl/etcd/ssl/node-vm-stage-k8s-master-1-key.pem
etcdctl get /registry/ceph.rook.io/cephfilesystems/rook-ceph/ceph-ephimeral --keys-only 
etcdctl get --keys-only --from-key '' | grep ceph.rook.io/ceph
```

## License

k8s-reset-terminating is released under the MIT license.
