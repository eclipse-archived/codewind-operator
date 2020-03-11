# codewind-operator

Work in progress


TODO:  Doc steps to modify the config map




## Example of creating a Keycloak service

```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Keycloak
metadata:
  name: codewind-keycloak-k3a237fj
spec:
  workspaceID: k3a237fj
  deploymentRef: devex-0001
```

Results in a new Keycloak service being created and accessible via kubectl:

```bash
$ kubectl get keycloaks
NAME                         DEPLOYMENT   NAMESPACE   AGE   ACCESS
codewind-keycloak-k3a237fj   devex-0001   codewind    28m   https://codewind-keycloak-k3a237fj.10.100.111.145.nip.io
```

This Keycloak instance will be created along with:

1. service account
2. deployment
3. pod
4. service
5. replicaset
6. ingress
7. secrets
8. PVC

```

NAME                         SECRETS   AGE
codewind-keycloak-k3a237fj   1         9m29s

NAME                                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/codewind-keycloak-k3a237fj   1/1     1            1           9m29s

NAME                                              READY   STATUS    RESTARTS   AGE
pod/codewind-keycloak-k3a237fj-6c958c6785-44nzx   1/1     Running   0          9m29s

NAME                                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/codewind-keycloak-k3a237fj   ClusterIP   10.104.73.250   <none>        8080/TCP   9m29s

NAME                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/codewind-keycloak-k3a237fj-6c958c6785   1         1         1       9m29s

NAME                         HOSTS                                             ADDRESS   PORTS     AGE
codewind-keycloak-k3a237fj   codewind-keycloak-k3a237fj.10.98.191.164.nip.io             80, 443   9m29s

NAME                                     TYPE                                  DATA   AGE
codewind-keycloak-k3a237fj-token-qgdff   kubernetes.io/service-account-token   3      9m29s
secret-keycloak-user-k3a237fj            Opaque                                2      9m29s

NAME                             STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
codewind-keycloak-pvc-k3a237fj   Bound    pvc-fde726e7-3b92-4f92-82e5-05807726b08e   1Gi        RWO            hostpath       9m29s
```

## Example of creating a Codewind service

```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Codewind
metadata:
  name: codewind-k81235kj
spec:
  size: 1
  workspaceID: k81235kj
  keycloakDeployment: devex-0001
  username: cody-sprint
  storageSize: 10Gi
```

Results in a new Codewind deployment and accessible via kubectl:

```bash
$ kubectl get codewinds
NAME                USERNAME      NAMESPACE   AGE   AUTH         ACCESSURL
codewind-k81235kj   cody-sprint   default     57s   devex-0001   .....
```
