# Data Availability Server Instructions

## Description
The Data Availability Server, `daserver`, allows storage and retrieval of transaction data batches for Arbitrum AnyTrust chains. It can be run in two modes: either committee member or mirror. Commitee members accept time-limited requests to store data batches from an Arbitrum AnyTrust sequencer, and if they store the data then they return a signed certificate promising to store that data. Commitee members and mirrors both respond to requests to retrieve the data batches. Mirrors exist to replicate and serve the data so that committee members to provide resiliency to the network in the case committee members going down, and to make it so committee members don't need to serve requests for the data directly. The data batches are addressed by a keccak256 hash of their contents. This document gives sample configurations for `daserver` in committee member and mirror mode.

### Interfaces
There are two interfaces, a REST interface supporting only GET operations and intended for public use, and an RPC interface intended for use only by the AnyTrust sequencer. Mirrors listen on the REST inferface only and respond to queries on `/get-by-hash/<hex encoded data hash>`. The response is always the same for a given hash so it is cacheable; it contains a `cache-control` header specifying the object is immutable and to cache for up to 28 days. The REST interface has a health check on `/health` which will return 200 if the underling storage is working, otherwise 503.

Committee members listen on the REST interface and additionally listen on the RPC interface for `das_store` RPC messages from the sequencer. The sequencer signs its requests and the committee member checks the signature. The RPC interface also has a health check that checks the underlying storage that responds requests with RPC method `das_healthCheck`.

### Storage
`daserver` can be configured to use one or more of three storage backends; S3, files on local disk, and database on disk (please give us feedback if there are other storage backends you would like supported). If more than one is selected, store requests must succeed to all of them for it to be considered successful, and retrieve requests only require one to succeed.

### Caching
An in-memory cache can be enabled to avoid needing to access underlying storage for retrieve requests .

### Synchronizing state
`daserver` also has an optional REST aggregator which, in the case that a data batch is not found in cache or storage, queries for that batch from a list other of REST servers, and then stores that batch locally. This is how committee members that miss storing a batch (not all committee members are required by the AnyTrust protocol to report success in order to post the batch's certificate to L1) can automatically repair gaps in data they store, and how mirrors can sync (a sync mode that eagerly syncs all batches is planned for a future release). A public list of REST endpoints is published online, which  `daserver` can be configured to download and use, and addititional endpoints can be specified in configuration.

## Image:
`offchainlabs/nitro-node:v2.0.0-beta.8-5ed2c72`

## Usage of daserver

Options for both committee members and mirrors:
```
 # Server options
      --enable-rest                                                                                enable the REST server listening on rest-addr and rest-port
      --log-level int                                                                              log level; 1: ERROR, 2: WARN, 3: INFO, 4: DEBUG, 5: TRACE (default 3)
      --rest-addr string                                                                           REST server listening interface (default "localhost")
      --rest-port uint                                                                             REST server listening port (default 9877)

 # L1 options
      --data-availability.l1-node-url string                                                       URL for L1 node, only used in standalone daserver; when running as part of a node that node's L1 configuration is used
      --data-availability.sequencer-inbox-address string                                           L1 address of SequencerInbox contract

 # Storage options
      --data-availability.local-db-storage.data-dir string                                         directory in which to store the database
      --data-availability.local-db-storage.discard-after-timeout                                   discard data after its expiry timeout
      --data-availability.local-db-storage.enable                                                  enable storage/retrieval of sequencer batch data from a database on the local filesystem
	  
      --data-availability.local-file-storage.data-dir string                                       local data directory
      --data-availability.local-file-storage.enable                                                enable storage/retrieval of sequencer batch data from a directory of files, one per batch

      --data-availability.s3-storage.access-key string                                             S3 access key
      --data-availability.s3-storage.bucket string                                                 S3 bucket
      --data-availability.s3-storage.discard-after-timeout                                         discard data after its expiry timeout
      --data-availability.s3-storage.enable                                                        enable storage/retrieval of sequencer batch data from an AWS S3 bucket
      --data-availability.s3-storage.object-prefix string                                          prefix to add to S3 objects
      --data-availability.s3-storage.region string                                                 S3 region
      --data-availability.s3-storage.secret-key string                                             S3 secret key

 # Cache options
      --data-availability.local-cache.enable                                                       Enable local in-memory caching of sequencer batch data
      --data-availability.local-cache.expiration duration                                          Expiration time for in-memory cached sequencer batches (default 1h0m0s)
	  
 # REST fallback options
      --data-availability.rest-aggregator.enable                                                   enable retrieval of sequencer batch data from a list of remote REST endpoints; if other DAS storage types are enabled, this mode is used as a fallback
      --data-availability.rest-aggregator.online-url-list string                                   a URL to a list of URLs of REST das endpoints that is checked at startup; additive with the url option
      --data-availability.rest-aggregator.urls strings                                             list of URLs including 'http://' or 'https://' prefixes and port numbers to REST DAS endpoints; additive with the online-url-list option
      --data-availability.rest-aggregator.sync-to-storage.eager                                    eagerly sync batch data to this DAS's storage from the rest endpoints, using L1 as the index of batch data hashes; otherwise only sync lazily
      --data-availability.rest-aggregator.sync-to-storage.eager-lower-bound-block uint             when eagerly syncing, start indexing forward from this L1 block
```
```
Options only for committee members:
      --enable-rpc                                                                                 enable the HTTP-RPC server listening on rpc-addr and rpc-port
      --rpc-addr string                                                                            HTTP-RPC server listening interface (default "localhost")
      --rpc-port uint                                                                              HTTP-RPC server listening port (default 9876)

      --data-availability.key.key-dir string                                                       the directory to read the bls keypair ('das_bls.pub' and 'das_bls') from; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified
      --data-availability.key.priv-key string                                                      the base64 BLS private key to use for signing DAS certificates; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified
````

Options generating/using JSON config:
```
      --conf.dump                                                                                  print out currently active configuration file
      --conf.file strings                                                                          name of configuration file
```

Options for producing Prometheus metrics:
```
      --metrics                                                                                    enable metrics
      --metrics-server.addr string                                                                 metrics server address (default "127.0.0.1")
      --metrics-server.port int                                                                    metrics server port (default 6070)
      --metrics-server.update-interval duration                                                    metrics server update interval (default 3s)
```

Some options are not shown because they are only used by nodes, or they are experimental/advanced. A complete list of options can be found by running `daserver --help`

## Sample Deployments

### Sample Committee Member

Using `daserver` as a committe member requires:
- A BLS private key to sign the Data Availability Certificates it returns to clients (the sequencer aka batch poster) requesting to Store data.
- The Ethereum L1 address of the sequencer inbox contract, in order to find the batch poster signing address.
- An Ethereum L1 RPC endpoint to query the sequencer inbox contract.
- A persistent volume to write the stored data to if using one of the local disk modes.
- A S3 bucket, and credentials (secret key, access key) of an IAM user that is able to read and write from it if you are uisng the S3 mode.

Once the DAS is set up, the local public key in `das_bls.pub` should be communicated out-of-band to the operator of the chain, along with a protocol (http/https), host, and port of the RPC server that can be reached by the sequencer, so that it can be added to the committee keyset.

#### Set up persistent volume

This is the persistent volume for storing the DAS database and BLS keypair.

```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: das-server
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 200Gi
  storageClassName: gp2
```

#### Generate key
The BLS keypair must be generated using the `datool keygen` utility. It can be passed to the `dasever` executable by file or on the command line.

In this sample deployment we use a k8s deployment to run `datool keygen` to create it as a file on the volume that the DAS will use. After this deployment has run once, the deployment can be torn down and deleted.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: das-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: das-server
  template:
    metadata:
      labels:
        app: das-server
    spec:
      containers:
      - command:
        - bash
        - -c
        - |
          mkdir -p /home/user/data/keys
          /usr/local/bin/datool keygen --dir /home/user/data/keys
          sleep infinity
        image: offchainlabs/nitro-node:v2.0.0-beta.8-5ed2c72
        imagePullPolicy: Always
        resources:
          limits:
            cpu: "4"
            memory: 10Gi
          requests:
            cpu: "4"
            memory: 10Gi
        ports:
        - containerPort: 9876
          protocol: TCP
        volumeMounts:
        - mountPath: /home/user/data/
          name: data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: das-server
```

#### Create DAS deployment

This deployment sets up a DAS server using the Arbitrum Nova Mainnet. It uses the L1 inbox contract at 0x211e1c4c7f1bf5351ac850ed10fd68cffcf6c21b. For the Arbitrum Nova Mainnet you must specify a Mainnet Ethereum L1 RPC endpoint.

This configuration sets up all three storage types. To disable any of them, remove the --data-availability.(local-file-storage|local-disk-storage|s3-storage).enable option, and the other options for that storage type can also be removed. If updating an existing deployment from `offchainlabs/nitro-node:v2.0.0-alpha.4` that is using the local files on disk storage type, you should use at least `local-file-storage`.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: das-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: das-server
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 50%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: das-server
    spec:
      containers:
      - command:
        - bash
        - -c
        - |
          mkdir -p /home/user/data/db
		  mkdir -p /home/user/data/badgerdb
          /usr/local/bin/daserver --data-availability.l1-node-url <YOUR ETHEREUM L1 RPC ENDPOINT>
--enable-rpc --rpc-addr '0.0.0.0' --enable-rest --rest-addr '0.0.0.0' --log-level 3 --data-availability.local-file-storage.enable --data-availability.local-file-storage.data-dir /home/user/data/db  --data-availability.local-db-storage.enable --data-availability.local-db-storage.data-dir /home/user/data/badgerdb --data-availability.s3-storage.enable --data-availability.s3-storage.access-key "<YOUR ACCESS KEY>" --data-availability.s3-storage.bucket <YOUR BUCKET> --data-availability.s3-storage.region <YOUR REGION> --data-availability.s3-storage.secret-key "<YOUR SECRET KEY>" --data-availability.s3-storage.object-prefix "YOUR OBJECT KEY PREFIX/" --data-availability.key.key-dir /home/user/data/keys --data-availability.local-cache.enable --data-availability.rest-aggregator.enable --data-availability.rest-aggregator.online-url-list "https://nova.arbitrum.io/das-servers" --data-availability.sequencer-inbox-address '0x211e1c4c7f1bf5351ac850ed10fd68cffcf6c21b'
        image: offchainlabs/nitro-node:v2.0.0-beta.8-5ed2c72
        imagePullPolicy: Always
        resources:
          limits:
            cpu: "4"
            memory: 10Gi
          requests:
            cpu: "4"
            memory: 10Gi
        ports:
        - containerPort: 9876
          hostPort: 9876
          protocol: TCP
        - containerPort: 9877
          hostPort: 9877
          protocol: TCP
        volumeMounts:
        - mountPath: /home/user/data/
          name: data
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /health/
            port: 9877
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 1
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: das-server
```


### Sample Mirror

Using `daserver` as a mirror requires:
- The Ethereum L1 address of the sequencer inbox contract, for syncing all batch data (future capability)
- An Ethereum L1 RPC endpoint to query the sequencer inbox contract.
- A persistent volume to write the stored data to if using one of the local disk modes.
- A S3 bucket, and credentials (secret key, access key) of an IAM user that is able to read and write from it if you are uisng the S3 mode. 

The mirror does not require a BLS key since it will not be accepting store requests from the sequencer.

Once the mirror is set up, please communicate a URL to reach it to the chain operator so they can add it to the public mirror list.

#### Set up persistent volume

This is the persistent volume for storing the DAS database.

```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: das-mirror
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 200Gi
  storageClassName: gp2
```

#### Create DAS deployment

This deployment sets up a DAS server using the Arbitrum Nova Mainnet. It uses the L1 inbox contract at 0x211e1c4c7f1bf5351ac850ed10fd68cffcf6c21b. For the Arbitrum Nova Mainnet you must specify a Mainnet Ethereum L1 RPC endpoint.

This configuration sets up all three storage types. To disable any of them, remove the --data-availability.(local-file-storage|local-disk-storage|s3-storage).enable option, and the other options for that storage type can also be removed.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: das-mirror
spec:
  replicas: 1
  selector:
    matchLabels:
      app: das-mirror
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 50%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: das-mirror
    spec:
      containers:
      - command:
        - bash
        - -c
        - |
          mkdir -p /home/user/data/db
		  mkdir -p /home/user/data/badgerdb
          /usr/local/bin/daserver --data-availability.l1-node-url <YOUR ETHEREUM L1 RPC ENDPOINT>
--enable-rest --rest-addr '0.0.0.0' --log-level 3 --data-availability.local-file-storage.enable --data-availability.local-file-storage.data-dir /home/user/data/db  --data-availability.local-db-storage.enable --data-availability.local-db-storage.data-dir /home/user/data/badgerdb --data-availability.s3-storage.enable --data-availability.s3-storage.access-key "<YOUR ACCESS KEY>" --data-availability.s3-storage.bucket <YOUR BUCKET> --data-availability.s3-storage.region <YOUR REGION> --data-availability.s3-storage.secret-key "<YOUR SECRET KEY>" --data-availability.s3-storage.object-prefix "YOUR OBJECT KEY PREFIX/" --data-availability.local-cache.enable --data-availability.rest-aggregator.enable --data-availability.rest-aggregator.urls "http://your-committee-member.svc.cluster.local:9877" --data-availability.rest-aggregator.online-url-list "https://nova.arbitrum.io/das-servers" --data-availability.rest-aggregator.sync-to-storage.eager --data-availability.rest-aggregator.sync-to-storage.eager-lower-bound-block 15025611 --data-availability.sequencer-inbox-address '0x211e1c4c7f1bf5351ac850ed10fd68cffcf6c21b'
        image: offchainlabs/nitro-node:v2.0.0-beta.8-5ed2c72
        imagePullPolicy: Always
        resources:
          limits:
            cpu: "4"
            memory: 10Gi
          requests:
            cpu: "4"
            memory: 10Gi
        ports:
        - containerPort: 9877
          hostPort: 9877
          protocol: TCP
        volumeMounts:
        - mountPath: /home/user/data/
          name: data
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /health/
            port: 9877
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 1
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: das-mirror
```



### Testing
#### Basic validation: health check data is present
In the docker image there is the `datool` utility that can be used to Store and Retrieve messages from a DAS. We will take advantage of a data hash that will always be present if the the health check is enabled.

From the pod:
```
$ /usr/local/bin/datool client rest getbyhash --url http://localhost:9877   --data-hash 0x8b248e2bd8f75bf1334fe7f0da75cc7c1a34e00e00a22a96b7a43d580d250f3d
Message: Test-Data

$ /usr/local/bin/datool client rpc getbyhash --url http://localhost:9876   --data-hash 0x8b248e2bd8f75bf1334fe7f0da75cc7c1a34e00e00a22a96b7a43d580d250f3d
Message: Test-Data
```

If you do not have the health check configured yet, you can trigger one manually as follows:
```
$ curl http://localhost:9877/health
```

Using curl to check the REST endpoint
```
$ curl  https://<HOST>/<PATH>/get-by-hash/8b248e2bd8f75bf1334fe7f0da75cc7c1a34e00e00a22a96b7a43d580d250f3d
{"data":"VGVzdC1EYXRh"}
```

#### Further validation: using Store interface directly

The Store interface of `daserver` validates that requests to store data are signed by the the Batch Poster's ECDSA key, identified via a call to the Sequencer Inbox contract on L1. It can also be configured to accept Store requests signed with another ECDSA key of your chosing. This could be useful for running load tests, canaries, or troubleshooting your own infrastructure. Using this facility, a load test could be constructed by writing a script to store arbitrary amounts of data at an arbitrary rate; a canary could be constructed to store and retrieve data on some interval.

Generate an ECDSA keypair:
```
$ /usr/local/bin/datool keygen --dir /dir-of-your-choice/ --ecdsa
```

Then add the following configuration option to `daserver`:
```
--data-availability.extra-signature-checking-public-key /dir-of-your-choice/ecdsa.pub

OR

--data-availability.extra-signature-checking-public-key 0x<contents of ecdsa.pub>
```

Now you can use the `datool` utility to send Store requests signed with the ecdsa private key:
```
$ /usr/local/bin/datool rpc store  --url http://localhost:9876 --message "Hello world" --signing-key /tmp/ecdsatest/ecdsa

OR

$ /usr/local/bin/datool client rpc store  --url http://localhost:9876 --message "Hello world" --signing-key "0x<contents of ecdsa>"
```

The above command outputs the `Hex Encoded Data Hash: ` which can be used to retrieve the data:
```
$ /usr/local/bin/datool client rpc getbyhash --url  http://localhost:9876 --data-hash 0x052cca0e379137c975c966bcc69ac8237ac38dc1fcf21ac9a6524c87a2aab423
Message: Hello world
$ /usr/local/bin/datool client rest getbyhash --url  http://localhost:9877 --data-hash 0x052cca0e379137c975c966bcc69ac8237ac38dc1fcf21ac9a6524c87a2aab423
Message: Hello world
```

The retention period defaults to 24h but can be configured for `datool client rpc store` with the option:
```
--das-retention-period
```

### Deployment recommendations
The REST interface is cacheable, consider using a CDN or caching proxy in front of your REST endpoint.

If you are running a mirror, the REST interface on your committee member does not have to be exposed publicly. Your mirrors can sync on your private network from the REST interface of your committee member and other public mirrors.

### Metrics
If metrics are enabled in configuration, then several useful metrics are available at the configured port (default 6070), at path `debug/metrics` or `debug/metrics/prometheus`.

| Metric | Description
| - | - |
| arb_das_rest_getbyhash_requests | Count of REST GetByHash calls |
| arb_das_rest_getbyhash_success | Successful REST GetByHash calls |
| arb_das_rest_getbyhash_failure | Failed REST GetByHash calls |
| arb_das_rest_getbyhash_bytes | Bytes retrieved with REST GetByHash calls |
| arb_das_rest_getbyhash_duration (p50, p75, p95, p99, p999, p9999) | Duration of REST GetByHash calls (ns) |
| arb_das_rpc_store_requests | Count of RPC Store calls |
| arb_das_rpc_store_success | Successful RPC Store calls |
| arb_das_rpc_store_failure | Failed RPC Store calls |
| arb_das_rpc_store_bytes | Bytes retrieved with RPC Store calls |
| arb_das_rpc_store_duration (p50, p75, p95, p99, p999, p9999) | Duration of RPC Store calls (ns) |