ARG ARCH=amd64
FROM --platform=linux/${ARCH} golang:1.19

ADD bin/etcd-badger /usr/local/bin/
#ADD etcdctl /usr/local/bin/
#ADD etcdutl /usr/local/bin/

WORKDIR /var/etcd/
WORKDIR /var/lib/etcd/

EXPOSE 2379 2380

# Define default command.
CMD ["--name etcd --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379 --enable-pprof --log-outputs=stderr --quota-backend-bytes=134217728"]
ENTRYPOINT ["/usr/local/bin/etcd-badger"]
