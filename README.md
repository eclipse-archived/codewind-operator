# codewind-operator

The Codewind operator helps with the deployment of Codewind instances in an Openshift or Kubernetes cluster.

There must only be one operator per cluster and it must be installed into the Codewind namespace.

To deploy the Codewind operator and setup a first Codewind remote instance, clone this repo to download all the required deploy yaml files, then log into your Kubernetes or Openshift cluster.

Once logged in continue with:

```
$ cd {path to cloned codewind-operator}
```

Create the initial namespace in your cluster (must be called codewind)
```
$ kubectl create namespace codewind
```

Create a service account which the operator will run under
```
$ kubectl create -f ./deploy/service_account.yaml
```

Create the access roles in the codewind namespace
```
$ kubectl create -f ./deploy/role.yaml
```

Connect the operator service account to the access roles
```
$ kubectl create -f ./deploy/role_binding.yaml
```

Create cluster roles. The Codewind operator needs some cluster permissions when querying outside of the installed namespace, for example when discovering Tekton or other services:

```
$ kubectl create -f ./deploy/cluster_roles.yaml
```

Connect the Operator service account to the cluster roles
```
$ kubectl create -f ./deploy/cluster_role_binding.yaml
```

Depending which version of Kubernetes or OpenShift you using, create the Custom Resource Definitions (CRD) for your environment.

For OpenShift 3.11.x clusters:
```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd-oc311.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd-oc311.yaml
```

For other versions including :
- OpenShift OCP 4.x
- Code Ready Containers 1.16.2 BuildDate:2020-02-03T23:11:39Z
- Kubernetes 1.16+

```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd.yaml
```

Deploy the Codewind operator into the cluster
```
$ kubectl create -f ./deploy/operator.yaml
```

## Configuring the default config map

The Codewind operator defaults can be found in the configmap file  `./deploy/codewind-configmap.yaml`.

Modify this file and set the ingressDomain value to one specific to your cluster.

The Ingress domain will be appended to any routes and URLs created by the operator. The ingress must already be registered in your DNS service and should resolve correctly from both inside and outside of the cluster.

Tip: If you are installing into a hosted cloud platform the ingress domain will usually be displayed on your cloud service dashboard.

An example configmap file:

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

After making changes you can either import the file using:

```
$ kubectl apply -f ./deploy/codewind-configmap.yaml
```

or instead edit the configmap which the operator has already installed:
```
$ kubectl edit configmap codewind-operator -n codewind
```

To check the status of the operator use :

```
kubectl get pods -n codewind
```

If successful, you should see the codewind-operator pod running and ready for work.


## Creating an initial Keycloak service

Keycloak is deployed and setup using the operator.

Requesting a Keycloak service is achived by importing YAML which the watching Codewind operator will react to.

For convenience, a sample yaml file is provided in this repo under `./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml`.

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

$ kubectl get keycloaks -n codewind
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

You can check these using standard Kubernetes or oc commands such as:

```
$ kubectl get serviceaccount -n codewind
$ kubectl get deployments -n codewind
$ kubectl get pods -n codewind
$ kubectl get services -n codewind
$ kubectl get pvc -n codewind
```

which will show each kind e.g:

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

Before Codewind services can be installed, users must be added to Keycloak. Adding new users is performed using the Keycloak Admin web page.

Each Codewind deployment must be tied to an existing user account.

To see the Keycloak deployment running in the codewind namespace and capture its access URL use the command :

```
$ kubectl get keycloaks -n codewind
NAME       NAMESPACE   AGE     ACCESS
devex001   codewind    5m22s   https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

By default, Keycloak is installed with an admin account where:

- Keycloak administrator username = admin
- Keycloak password = admin

Open the keycloak Access URL in a browser and accept the self signed certificate warnings.

If you are unable to connect to Keycloak, check that the pod has started running and that storage has been provisioned.

You can inspect the storage claim status with :

```
$ kubectl get pvc -n codewind
(check that the status shows **Bound** for the entry codewind-keycloak-pvc-{keycloakName})
```

and inspect the Keycloak pod status with :
```
$ kubect get pods -n codewind
(check that the returned codewind-keycloak-{keycloakName} entry shows **Running** with 1 container of 1 ready)
```

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

When the Codewind Operator needs to update Keycloak it uses login credentials saved in a Kubernetes secret. By default during initial deployment, that secret will have a username and password of "admin". If you changed your admin password in a previous step, you will need to update the keycloak secret to match.

The secret is installed in the same namespace as the operator (codewind) and named `secret-keycloak-user-{keycloakname}`

If you have an administration UI for your cluster you may use it to locate the secret and edit the `keycloak-admin-password` field, or you can use the command line tools:

`$ kubectl edit secret secret-keycloak-user-{keycloakname} -n codewind`

or

`$ oc edit secret secret-keycloak-user-{keycloakname} -n codewind`

Note: Using the command line tools does require an extra step to base64 encode your password string before saving it into the secret.  You can base64 encode your new password using:

```
$ echo -n 'myNewPassword' | base64
bXlOZXdQYXNzd29yZA==
```

then save `bXlOZXdQYXNzd29yZA==` as the value for `keycloak-admin-password` rather than the clear text `myNewPassword`

## Deploy a Codewind instance

Deploying a new Codewind instance will involve applying one last piece of yaml.

A copy of this yaml is available in:

`./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml`

To deploy Codewind, change the following fields :

- **name**: a unique name for this deployment
- **keycloakDeployment**: the keycloak service used for authentication
- **username**: a username already registered in the specified keycloak service

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

Note:

- the `name` field is the name of the deployment and must be unique within the cluster. It should contain numbers and letters only (no spaces or punctuation)
- the `keycloakDeployment` field is the name of the keycloak instance that will provide authentication services. Keycloak must have already been provisioned and be running.
- the `username` field is the keycloak registered user who will own this Codewind instance. (alpha numeric characters only)
- the `loglevel` can be used to increase log levels of the Codewind pods. allowed values one of either: error, warn, info, debug or trace
- the `storageSize` field sets the PVC size to 10GB.

Apply this yaml and have the operator create and configure both Codewind and Keycloak with one command:

```
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml
codewind.codewind.eclipse.org/codewind-k81235kj created
```

To view all the Codewind deployments in the codewind namespace:
```
$ kubectl get codewinds -n codewind
NAME                USERNAME   NAMESPACE   AGE   KEYCLOAK   AUTHSTATUS   ACCESSURL
jane1               jane       codewind    23m   devex001   Completed    https://codewind-gatekeeper-jane1.10.98.117.7.nip.io
```

The `kubectl get codewinds` command lists all the running Codewind deployments in the specified namespace.  Each line represents a deployment and includes the username of the developer it has been assigned to. The Keycloak service name and auth config status. Most importantly users will need their Access URL which they will add to the IDE when creating a connection.  Use the -n flag to target a specific namespace e.g. `-n codewind`

Note:

If the user was assigned a temporary password, they will need to login to Codewind from a browser and complete these next steps to set a new password and activate their account.

1. Open the gatekeeper ACCESS URL obtained in the previous step for the Codewind deployment
2. Log in using the provided username and initial password
3. follow the prompts to change the password
4. proceed with setting up the IDE connection using the newly changed password


## Building the Operator

To build the operator container image from source moved the cloned repo into your go directory eg:

```
~/go/src/github.com/eclipse/codewind-operator
```

Then run the commands:

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

Before deploying the operator with any changes, modify the image field listed in the file `./deploy/operator.yaml` setting it to the location of your built and pushed operator image.

