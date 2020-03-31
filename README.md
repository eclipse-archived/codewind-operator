# codewind-operator

The Codewind operator helps with the deployment of Codewind instances in an Openshift or Kubernetes cluster.

There must only be one operator per cluster and it must be installed into the Codewind namespace.

To deploy the Codewind operator and setup a first Codewind remote instance, clone this repo, log into your Kubernetes or Openshift cluster and continue with:

```
$ cd {path to repo codewind-operator}
$ kubectl create namespace codewind
(this has to be codewind)
$ kubectl create -f ./deploy/service_account.yaml
(service account for which the operator will run under, a lot of permissions required)
$ kubectl create -f ./deploy/role.yaml
(RBAC permissions to assign - i.e. create Ingress, routes etc)
$ kubectl create -f ./deploy/role_binding.yaml
(connects service account to role)
$ kubectl create -f ./deploy/cluster_roles.yaml
(permissions at namespace level not sufficient, need permissions at cluster level, talk to services outside of namespace)
$ kubectl create -f ./deploy/cluster_role_binding.yaml
(connects service account to role)
```

Create the CRD.  For Openshift 3.11.x clusters:

```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd-oc311.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd-oc311.yaml
```

For other versions of Openshift and Kubernetes:

```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd.yaml
```

Deploy the operator
```
$ kubectl create -f ./deploy/operator.yaml
(downloads images into the cluster)
```

## Configuring the default config map

The Codewind operator defaults can be found in the config-map file  `./deploy/codewind-configmap.yaml`.
Modify this file setting the ingressDomain value to one specific to your cluster.
The Ingress domain will be appended to any routes and URLs created by the operator.
It must already be registered in your DNS service and should resolve correctly from both inside and outside of the cluster.

example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: codewind-operator
  namespace: codewind
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

Import the following YAML to configure a default instance of Keycloak.
For convenience, a copy of this file is provided in this repo under `./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml`.

In this example a new Keycloak service will be created called "devex001" in the namespace "codewind" with a PVC claim of 1GB.

```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Keycloak
metadata:
  name: devex001
  namespace: codewind
spec:
  storageSize: 1Gi
```

e.g:

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

e.g:

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

During deployment of the Keycloak service, the operator will configure the security realm as specified by the defaults config map.

Before Codewind services can be installed, users must be added to Keycloak. This is achieved via the Keycloak Admin web page.
Each Codewind deployment must be tied to an existing user account.

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

- Click:   Administration Console from the link provided
- Log into Keycloak using the Keycloak admin credentials.
  - username :   admin
  - password :   admin

IMPORTANT: Once logged in, change the admin password by clicking the `Admin` link in the top right of the page.
Then choose `Manage Account / Password` and set a new replacement administrator password.

- Switch back to the admin console using the link at the top of the page. Or alternatively logout and log back into Keycloak as the admin user with your new admin password.

## Registering Codewind users

Ensure that the Realm is set to "Codewind" by clicking on the drop down arrow in the top right of the page. Select Codewind if necessary. Then:

- Click: Users
- Click: Add user
- Complete username field:  jane
- Complete email / Firstname / Lastname: as required
- Ensure user enabled: On
- Click: Save

Assign an initial password to the user account by clicking 'Credentials' and then add their initial password.

The field Temporary = On will require Jane to change her password during first connection.  Setting Temporary = Off will make this password valid for continuous use and will not require changing on first connect.

Click:  Set Password to save changes
Log out of the keycloak admin page

## Updating the keycloak password in the operator secret

When the Codewind Operator needs to update Keycloak it uses login credentials saved in a Kubernetes secret.  By default during deployment that secret will have a username and password of "admin". If you changed your admin password in a previous step, you will need to update the keycloak secret to match.

The secret is installed in the same namespace as the operator (codewind) and named `secret-keycloak-user-{keycloakname}`

If you have an adminsration UI for you cluster you may use it to locate the secret and edit the `keycloak-admin-password` field or from the command line using `kubectl edit secret secret-keycloak-user-{keycloakname}` or `oc edit secret secret-keycloak-user-{keycloakname}`

Note: Using the command line tools does require an extra step to base64 your password string before saving it into the secret.  You can base64 encode your new password using:

```
$ echo -n 'myNewPassword' | base64
bXlOZXdQYXNzd29yZA==
```

then save `bXlOZXdQYXNzd29yZA==` as the value for `keycloak-admin-password` rather than `myNewPassword`

## Deploy a Codewind instance

Deploying a new Codewind instance will involve applying one last piece of YAML.

A copy of this yaml is available in this repo under :

`./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml`

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

- the `name` field is the name of the deployment and must be unique within the cluster.
- the `keycloakDeployment` field is the name of the keycloak instance that will provide authentication services. It must have already been provisioned and be running.
- the `username` field is the keycloak registered user who will own this Codewind instance.
- the `loglevel` can be used to increase log levels of the Codewind pods.
- the `storageSize` field sets the PVC size to 10GB.

Apply this yaml and have the operator create and configure both Codewind and Keycloak:

```
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml
codewind.codewind.eclipse.org/codewind-k81235kj created

$ kubectl get codewinds
NAME                USERNAME   NAMESPACE   AGE   KEYCLOAK   AUTHSTATUS   ACCESSURL
jane1               jane       codewind    23m   devex001   Completed    https://codewind-gatekeeper-jane1.10.98.117.7.nip.io
```

The `kubectl get codewinds` command lists all the running Codewind deployments and the username of the developer it has been assigned. The Keycloak service name and auth config status is also displayed along with the Access URL that needs to be added to the IDE when creating a connection.

Note:

If the user was assigned a temporary password, they will need to login to Codewind from a browser and complete the steps necessary to set a new password and activate their account.

1. Open the gatekeeper URL for the Codewind deployment
2. Log in using the provided username and initial password
3. follow the prompts to change the password
4. proceed with setting up the IDE connection using the newly changed password

## Building the Operator

To build the operator container image from source, run the command:

```
$ brew install operator-sdk
$ operator-sdk version
operator-sdk version: "v0.15.2", commit: "ffaf278993c8fcb00c6f527c9f20091eb8dd3352", go version: "go1.13.8 darwin/amd64"
$ export GO111MODULE=on
$ cd {pathToCodewindOperatorCode}
$ go mod tidy
$ operator-sdk build {yourDockerRegistry}/codewind-operator:latest
$ docker push {yourDockerRegistry}/codewind-operator:latest
```

Then before deploying the operator modify the image listed in the file `./deploy/operator.yaml` to point to your build image
