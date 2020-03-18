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

The Codewind operator defaults can be found in the file  `./deploy/codewind-configmap.yaml`  You will need to modify this file setting the ingressDomain value to one specific to your cluster. The Ingress domain will be appended to any routes and URLs created by the operator. It must already be registered in your DNS service and should resolve correctly from both inside and outside of the cluster.

eg yaml:

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

Import the following YAML to configure a default instance of Keycloak. For convenience, a copy of this file is provided in `./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml`

In this example a new Keycloak service will be created called "devex001" in the namespace "codewind" with a PVC claim of 1GB


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

During deployment,  the operator will create: 

1. A service account
2. A deployment
3. A pod
4. A service
5. An ingress or route
6. A self signed TLS certificate
7. A storage claim
8. Any secrets

eg:

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

## Preparing Keycloak for Codewind

During deployment of the Keycloak service, the operator will configur the security realm as specified by the defaults config map.

Before Codewind services can be installed, users must be added to Keycloak achieved via the Keycloak Admin web page.  Each Codewind deployment must be tied to a existing user account.

To see the Keycloak access URL use the command :

```
$ kubectl get keycloaks
NAME       NAMESPACE   AGE     ACCESS
devex001   codewind    5m22s   https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

By default, Keycloak is installed with an admin account where :

- Keycloak administrator username = admin
- Keycloak password = admin

Open the keycloak Access URL in a browser and accept the self signed certificate warnings. 

*Click:   Administration Console from the link provided
* Log into Keycloak using the Keycloak admin credentials.
  - username :   admin
  - password :   admin
  
IMPORTANT: Once logged in,  change the admin password by clicking the `Admin` link in the top right of the page. Then choose `Manage Account / Password` and set a new replacement administrator password.

* Switch back to the admin console using the link at the top of the page. Or alternatively logout and log back into Keycloak as the admin user with your new admin password.

# Registering Codewind users :

Ensure that the Realm is set to "Codewind" by clicking on the drop down arrow in the top right of the page. Select Codewind if necessary. Then:

* Click: Users
* Click: Add user
* Complete username field:  jane
* Complete email / Firstname / Lastname: as required
* Ensure user enabled: On
* Click: Save

Assign an initial password to the user account by clicking 'Credentials' and then add their initial password.

The field Temporary = On will require Jane to change her password during first connection.  Setting Temporary = Off will make this password valid for continuous use and will not require changing on first connect.

Click:  Set Password to save changes
Log out of the keycloak admin page


## Deploy a Codewind instance

Deploying a new Codewind instance will involve applying a piece of YAML.

A copy of this yaml is available : 

`./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml `

To deploy Codewind change the following fields :

- name: a unique name for this deployment
- keycloakDeployment: the keycloak service used for authentication
- username: a username already registered in the specified keycloak service

An example of valid yaml is :

```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Codewind
metadata:
  name: jane1
  namespace: codewind
spec:
  keycloakDeployment: devex001
  username: jane
  logLevel: info
  storageSize: 10Gi
```

Things to note :

* the `name` field is the name of the deployment and must be unique within the cluster.
* the `keycloakDeployment` field is the name of the keycloak instance that will provide authentication services.  It must have already been provisioned and running.
* the `username` field is the keycloak registered user who will own this Codewind instance.
* the `workspaceID` is a short name label used to identify this deployment.
* the `loglevel` can be used to increase log levels of the codewind pods.
* the `storageSize` field sets the PVC size to 10GB.

Apply this yaml and have the operator create and configure both Codewind and Keycloak:

```
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml
codewind.codewind.eclipse.org/codewind-k81235kj created

$ kubectl get codewinds
NAME                USERNAME   NAMESPACE   AGE   KEYCLOAK   AUTHSTATUS   ACCESSURL
jane1               jane       codewind    23m   devex001   Completed    https://codewind-gatekeeper-jane1.10.98.117.7.nip.io
```

The `kubectl get codewinds` command lists all the running Codewind deployments and the username of the developer it has been assigned. The Keycloak service name and auth config status is also displyed along with the Access URL that needs to be added to the IDE when creating a connection.

Note:

If the user was assigned a temporary password, they will need to login to Codewind from a browser and complete the steps necessary to set a new password and activte their account.

1. Open the gatekeeper URL for the Codewind deployment
2. Log in using the provided username and initial password
3. follow the prompts to change the password
4. proceed with setting up the IDE connection using the newly changed password
