# Usage
```
go mod vendor
bash +x hack/update-codegen.sh
go build -o samplecrd-controller .
./samplecrd-controller -kubeconfig=$HOME/.kube/config -alsologtostderr=true
```
