# Usage
```
# 保证项目目录samplecrd-code在github.com/xing393939目录下
go mod vendor
bash +x hack/update-codegen.sh
go build -o samplecrd-controller .
./samplecrd-controller -kubeconfig=$HOME/.kube/config -alsologtostderr=true

# 测试创建crd和network对象
kubectl apply -f crd/network.yaml
kubectl apply -f example/example-network.yaml

# 测试删除crd和network对象
kubectl delete -f example/example-network.yaml
kubectl delete -f crd/network.yaml
```
