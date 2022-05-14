# Data Availability Server Instructions

#### Image:
`offchainlabs/nitro-node:v2.0.0-alpha.4`

#### Usage of daserver

Commonly used options:

      --addr string                                                   HTTP-RPC server listening interface (default "localhost")
      --port uint                                                     Port to listen on (default 9876)
      --log-level int                                                 log level (default 3)
      
      --data-availability.local-disk.data-dir string                  The directory to use as the DAS file-based database
      --data-availability.local-disk.key-dir string                   The directory to read the bls keypair ('das_bls.pub' and 'das_bls') from
      --data-availability.local-disk.l1-node-url string               URL of L1 Ethereum node
      --data-availability.local-disk.sequencer-inbox-address string   L1 address of SequencerInbox contract
      --data-availability.mode string                                 mode ('onchain', 'local-disk', or 'aggregator') (default "onchain")
      
Other options can be found by running `daserver --help`      


## Sample Deployment

### Local Disk Mode

The Data Availability Service (DAS) in local disk mode requires:
- A BLS private key to sign the Data Availability Certificates it returns to clients (the batch poster) requesting to Store data.
- The Ethereum L1 address of the sequencer inbox contract, in order to find the batch poster signing address.
- An Ethereum L1 RPC endpoint to query the sequencer inbox contract.
- A persistent volume to write the stored data to.

Once the DAS is set up, the local public key in `das_bls.pub` should be communicated out-of-band to the operator of the Data Availability Service Aggregator, along with a protocol (http/https), host, and port that can be publicly reach, so that it can be added to the committee keyset.

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
The BLS keypair can be generated using the `datool keygen` utility. It can be passed to the `dasever` executable by file or on the command line.

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
        image: offchainlabs/nitro-node:v2.0.0-alpha.4
        imagePullPolicy: Always
        resources:
          limits:
            cpu: "8"
            memory: 58Gi
          requests:
            cpu: "8"
            memory: 58Gi
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

This deployment sets up a DAS server using the Anytrust Goerli Devnet. It uses the devnet L1 inbox contract at 0xd5cbd94954d2a694c7ab797d87bf0fb1d49192bf. For the Anytrust Goerli Devnet you must specify a Goerli L1 RPC endpoint.

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
          /usr/local/bin/daserver --data-availability.local-disk.l1-node-url <YOUR ETHEREUM L1 RPC ENDPOINT> --addr '0.0.0.0' --data-availability.mode local-disk --data-availability.local-disk.key-dir /home/user/data/keys --data-availability.local-disk.data-dir /home/user/data/db --data-availability.local-disk.sequencer-inbox-address '0xd5cbd94954d2a694c7ab797d87bf0fb1d49192bf'
        image: offchainlabs/nitro-node:v2.0.0-alpha.4
        imagePullPolicy: Always
        resources:
          limits:
            cpu: "8"
            memory: 58Gi
          requests:
            cpu: "8"
            memory: 58Gi
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


#### Optional: Validating deployment
In the docker image there is the `datool` utility that can be used to Store and Retrieve messages from a DAS. 
In order for the DAS to accept the Store requests, you must set the flags `--data-availability.local-disk.sequencer-inbox-address none --data-availability.local-disk.l1-node-url none` when running `daserver` since you will not be able to sign the test messages with the batch poster's private.

After testing the `--data-availability.local-disk.sequencer-inbox-address` and `--data-availability.local-disk.l1-node-url` flags should be restored.

```
$ /usr/local/bin/datool client store --url http://localhost:9876 --message "Hello world"
Base64 Encoded Cert: gIQusW9kVVDt3bIIi6jo+RlFSqVd/40z/aSADWJvOV9H7WwRsLW4CJYN8m9b/EcdBMGZWw/9IFWSWtG+KNa6rf0AAAAAYn1lrwAAAAAAAAABCgVCGJWsseHBNRgaOVBeNj4eH3kZhZGIfxjCr8Uf22FtS3+8f839VxX5OASahFqODMP/JgiHQARAQPVsbllvWjJz8ZJ13a0Y094O2VKjyRog7qNM3VwyPkkvfhycmfNN
$ ./target/bin/datool client retrieve --url http://localhost:9876 --cert "gIQusW9kVVDt3bIIi6jo+RlFSqVd/40z/aSADWJvOV9H7WwRsLW4CJYN8m9b/EcdBMGZWw/9IFWSWtG+KNa6rf0AAAAAYn1lrwAAAAAAAAABCgVCGJWsseHBNRgaOVBeNj4eH3kZhZGIfxjCr8Uf22FtS3+8f839VxX5OASahFqODMP/JgiHQARAQPVsbllvWjJz8ZJ13a0Y094O2VKjyRog7qNM3VwyPkkvfhycmfNN"
Message: Hello world
```