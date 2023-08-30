CLoud init logs: /var/log/cloud-init-output.log

Available cloud init data: cloud-init query --all

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config



**CNI**

kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
or
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
cilium install --version 1.13.5

**CCM**
See https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/#running-cloud-controller-manager
Do manually for now:
set providerID on node
remove uninitialized taint



**CSI**

kubectl create secret generic ionos-secret -n kube-system --from-literal \
$DATACENTER_ID="{\"username\": \"$IONOS_USERNAME\", \"password\":\"$IONOS_PASSWORD\"}"

clusterid in values updaten
helm install csi-ionos . -n kube-system

test:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myclaim
spec:
  storageClassName: ionos-enterprise-hdd
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
    - name: myfrontend
      image: nginx
      volumeMounts:
        - mountPath: "/var/www/html"
          name: mypd
  volumes:
    - name: mypd
      persistentVolumeClaim:
        claimName: myclaim
```
  

gives error:
error syncing claim "ca8ed7db-9161-4ece-beb9-e281aa6bdaa3": failed to provision volume with StorageClass "ionos-enterprise-hdd": rpc error: code = Internal desc = could not initialize valid client for datacenter f515f395-33c4-476b-96f1-2e06078403c6: could not initialize default cloud(s)