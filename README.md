# network
```
# 保证项目目录samplecrd-code在github.com/xing393939目录下
go mod vendor
bash +x hack/codegen-network.sh
go build -o network-controller ./cmd/network
./network-controller -kubeconfig=$HOME/.kube/config -alsologtostderr=true

# 测试创建crd和network对象
kubectl apply -f crd/network.yaml
kubectl apply -f example/example-network.yaml

# 测试删除crd和network对象
kubectl delete -f example/example-network.yaml
kubectl delete -f crd/network.yaml
```

# etcdcluster
```
# 保证项目目录samplecrd-code在github.com/xing393939目录下
go mod vendor
bash +x hack/codegen-etcdcluster.sh
go build -o etcdcluster-controller ./cmd/etcdcluster
./etcdcluster-controller -kubeconfig=$HOME/.kube/config -alsologtostderr=true

# 测试创建crd和network对象
kubectl apply -f crd/etcdcluster.yaml
kubectl apply -f example/example-etcdcluster.yaml

# 测试删除crd和network对象
kubectl delete -f example/example-etcdcluster.yaml
kubectl delete -f crd/etcdcluster.yaml
```
