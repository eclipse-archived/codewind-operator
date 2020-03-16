# codewind-operator

Run the following commands to install the operator into your cluster:

```
$ kubectl create namespace codewind
$ kubectl create -f ./deploy/service_account.yaml;
$ kubectl create -f ./deploy/role.yaml;
$ kubectl create -f ./deploy/role_binding.yaml;
$ kubectl create -f ./deploy/operator.yaml;
```

## Configuring default config map

Operator defaults can be found in the file  `./deploy/codewind-configmap.yaml`  you will need to modify this file with your cluster ingress domain using the example as a guide. The Ingress domain will be appended to any routes and URLs created by the operator. It must already be registered in your DNS service and resolve from both inside and outside of the clutser.

eg:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: codewind-operator
data:
  ingressDomain: 10.98.117.7.nip.io
  defaultRealm: codewind
  storageKeycloakSize: 1Gi
  storageCodewindSize: 10Gi
```

Import the file using :

```
$ kubectl apply -f ./deploy/codewind-configmap.yaml
```


## Creating an initial Keycloak service

Keycloak is deployed and setup using the operator.

Import the following YAML to configure a default instance of keycloak. A copy of this file is provided in `./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml`


```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Keycloak
metadata:
  name: devex001
  namespace: codewind
spec:
  storageSize: 1Gi
```

eg:

```bash
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml
keycloak.codewind.eclipse.org/devex001 created

$ kubectl get keycloaks
NAME       NAMESPACE   AGE   ACCESS
devex001   codewind    4s    https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

During deployment,  the operator will create any required 

1. service accounts
2. a deployment
3. a pod
4. a service
5. an ingress or route
6. a self signed TLS certificate
7. a storage claim
8. any secrets
```

NAME                         SECRETS   AGE
codewind-keycloak-devex001   1         2m53s


NAME                                              READY   STATUS    RESTARTS   AGE
pod/codewind-keycloak-devex001-7454d4ff6c-fnrsr   1/1     Running   0          2m10s


NAME                                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/codewind-keycloak-devex001   ClusterIP   10.111.228.52   <none>        8080/TCP   2m10s


NAME                                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/codewind-keycloak-devex001   1/1     1            1           2m10s

NAME                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/codewind-keycloak-devex001-7454d4ff6c   1         1         1       2m10s
```

## Preparing Keycloak for codewind

During deployment of the Keycloak service, the operator has configured the security realm as specified by the defaults config map.

Before Codewind services can be installed, users must be added via the Keycloak UI.  Each Codewind deployment should be tied to a user account.

To see the Keycloak service, use the command :

```
$ kubectl get keycloaks
NAME       NAMESPACE   AGE     ACCESS
devex001   codewind    5m22s   https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

By default, Keycloak is installed with :

- Keycloak administrator: admin
- Keycloak password: admin


* Open the keycloak URL in a browser and accept the self signed certificate warnings. `kubectl get keycloaks`

* Select:   Administration Console from the link provided

* Log into Keycloak using the Keycloak admin credentials.
  - username :   admin
  - password :   admin
  
Once logged in,  change the admin password by clicking the `Admin` link in the top right of the page. Then choose `Manage Account / Password` and set a replacment password.

* Switch back to the admin console using the link at the top of the page. Or alternativly logout and log back into the Keycloak as admin with your new admin  password.

# Registering Codewind users :


After logging into the Keycloak Administration website, ensure that the Realm is set to "Codewind" by clicking on the drop down arrow in the top right of the page. Select Codewind if necessary. 

Click:  Users
Click:  Add user
Add a new username :   jane
email / Firstname / Lastname as required
Ensure user enabled :  On
Click:  Save

Assign an initial password to user jane by clicking 'Credentials' and add an initial password.

Leaving the field Temporary = On will require Jane to change her password during first connection.  Setting Temporary = Off will make this new password valid for continuous use.

Click:  Set Password to save changes
Log out of the keycloak admin page


## Deploy a Codewind instance

A Codewind deployment yaml has the following description :

(A copy of this yaml is available : `./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml `)

To deploy Codewind change the following fields :

* name = a unique name for this deployment
* workspaceID = a short name for this deployment
* keycloakDeployment = the keycloak service used for authentication
* username = a username already registered in the specified keycloak service

An example of valid yaml is :

```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Codewind
metadata:
  name: codewind-k81235kj
  namespace: codewind
spec:
  workspaceID: k81235kj
  keycloakDeployment: devex001
  logLevel: info
  username: jane
  storageSize: 10Gi
```

Apply this yaml and have the operator create and configure both Codewind and Keycloak:

```
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml
codewind.codewind.eclipse.org/codewind-k81235kj created
$ kubectl get codewinds
NAME                USERNAME   NAMESPACE   AGE   KEYCLOAK   AUTHSTATUS   ACCESSURL
codewind-k81235kj   jane       codewind    10s   devex001   Completed    https://codewind-gatekeeper-k81235kj.10.98.117.7.nip.io
```

This command lists all the running Codewind deployments and the username of the developer it has been assigned. The keycloak service name and auth config status is shown along with the Acces URL that needs to be added to the IDE when creating a connection.

