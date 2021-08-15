# Egress IP Operator

The Egress IP Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) which could direct egress traffic of pods through specified public IP.

## Getting started

Firstly, ensure you have an VM-based AKS cluster (VMSS is not supported) with CNI networking. The Standard public IP address resources that will be used for egress traffic needs to be created in the node resource group of AKS. And your need to create an Azure service principle which will be used by the egress-ip-operator and add it as Contrinutor of the AKS node resource group.

Deploy egress-ip-operator into your AKS cluster:
```
git clone https://github.com/yingeli/egress-ip-operator.git
cd egress-ip-operator
make deploy IMG=yingeli/egress-ip-operator:0.1.44
```

A namespace "egress-ip" will be created during the deployment. Now we add the credential of the Azure service principle as secrets:
```
kubectl create secret generic azure-credential --namespace=egress-ip --from-literal='clientid=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' --from-literal='clientsecret=xxxxxxxxxxxxxxxxxxxxxxxxxxxxx' --from-literal='tenantid=xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxxx'
```

To associate a new egress ip to the cluster, add below EgressIP CRD resource with your public IP:
```
apiVersion: egressip.yingeli.github.com/v1alpha1
kind: EgressIP
metadata:
  name: egress-ip-001
  namespace: egress-ip
spec:
  # Add your public IP here
  ip: XXX.XXX.XXX.XXX
  podSelector:
    matchLabels:
      app: curl-001
```

Newly created pod with labal "app: curl-001" will use the public IP specified for source IP of the egress traffic automatically. To test it, you can apply below deployment:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: curl-001
  labels:
    app: curl-001
spec:
  replicas: 1
  selector:
    matchLabels:
      app: curl-001
  template:
    metadata:
      labels:
        app: curl-001
    spec:
      containers:
      - name: curl
        image: curlimages/curl
        command: ['sh', '-c', "sleep 3600"]
```

Run "curl -4 icanhazip.com" in the pod will return the public IP that you specified in EgressIP CRD.