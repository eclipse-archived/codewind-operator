# codewind-operator

The Codewind operator helps with the deployment of Codewind instances in an OpenShift or Kubernetes cluster.

There must only be one operator per cluster and it must be installed into the Codewind namespace.

To deploy the Codewind operator and set up a first Codewind remote instance, clone this repo to download all the required deploy `.yaml` files. Then log in to your Kubernetes or OpenShift cluster.

After you log in, enter the following command:
```
$ cd {path to cloned codewind-operator}
```

Create the initial namespace in your cluster that must be called `codewind`:
```
$ kubectl create namespace codewind
```

Create a service account for the operator to run under:
```
$ kubectl create -f ./deploy/service_account.yaml
```

Create the access roles in the `codewind` namespace:
```
$ kubectl create -f ./deploy/role.yaml
```

Connect the operator service account to the access roles:
```
$ kubectl create -f ./deploy/role_binding.yaml
```

Create cluster roles. The Codewind operator needs some cluster permissions when querying outside of the installed namespace, for example, when discovering Tekton or other services:
```
$ kubectl create -f ./deploy/cluster_roles.yaml
```

Connect the Operator service account to the cluster roles:
```
$ kubectl create -f ./deploy/cluster_role_binding.yaml
```

Depending which version of Kubernetes or OpenShift you use, create the Custom Resource Definitions (CRD) for your environment.

For OpenShift 3.11.x clusters:
```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd-oc311.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd-oc311.yaml
```

For other versions including:
- OpenShift OCP 4.x
- Code Ready Containers 1.16.2 BuildDate:2020-02-03T23:11:39Z
- Kubernetes 1.16+
```
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_keycloaks_crd.yaml
$ kubectl create -f ./deploy/crds/codewind.eclipse.org_codewinds_crd.yaml
```

Deploy the Codewind operator into the cluster:
```
$ kubectl create -f ./deploy/operator.yaml
```

## Configuring the default config map

See the Codewind operator defaults in the `configmap` file, `./deploy/codewind-configmap.yaml`.

Modify this file and set the `ingressDomain` value to one specific to your cluster.

The Ingress domain is appended to any routes and URLs created by the operator. The ingress must already be registered in your DNS service and resolves correctly from both inside and outside of the cluster.

**Note:** If you are installing into a hosted cloud platform, the ingress domain is usually displayed on your cloud service dashboard.

An example `configmap` file:

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

After making changes you can either import the file using the following command:
```
$ kubectl apply -f ./deploy/codewind-configmap.yaml
```

Or instead edit the `configmap` that the operator already installed:
```
$ kubectl edit configmap codewind-operator -n codewind
```

To check the status of the operator use:
```
kubectl get pods -n codewind
```

The `codewind-operator` pod runs and is ready for work.

## Persistent storage requirements

Keycloak and Codewind pods have storage requirements. Both require available PersistentStorage to be configured and available before you attempt to deploy each service.

Each Keycloak instance requires by default:

- 1Gi capacity
- access mode of RWO (ReadWriteOnly)

Each Codewind instance requires by default:

- 10Gi capacity
- access mode of RWX (ReadWriteMany)

Before continuing, ensure your cluster has the necessary `Persistent Volume` entries available for claiming. If your cluster is not using dynamically assigned storage, you can check the available status by using the command : `kubectl get pv`

In this example there are three Persistent Volumes available one sized 1Gi (mode RWO) and two sized 10Gi (mode RWX) which will allow for one new Keycloak and two new Codewind deployments.

```
$ kubectl get pv
NAME               CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS
pv001              1Gi        RWO            Recycle          Available
pv002              10Gi       RWX            Recycle          Available
pv003              10Gi       RWX            Recycle          Available
```

If you do not have sufficient PV availability and your cluster is not configured for dynamic storage, work with your cluster administrator to configure and register additional storage volumes.

If storage is not available, neither Keycloak nor Codewind will be able to start and will remain in a "pending" state.

## Creating an initial Keycloak service

Keycloak is deployed and set up using the operator.

To request a Keycloak service, import `yaml`, which the watching Codewind operator reacts to.

For convenience, a sample `.yaml` file is provided in this repo under `./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml`.

In this example, a new Keycloak service is created and called `devex001` in the `codewind` namespace with a PVC claim of 1GB.
```yaml
apiVersion: codewind.eclipse.org/v1alpha1
kind: Keycloak
metadata:
  name: devex001
  namespace: codewind
spec:
  storageSize: 1Gi
```

For example:
```bash
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml
keycloak.codewind.eclipse.org/devex001 created

$ kubectl get keycloaks -n codewind
NAME       NAMESPACE   AGE   ACCESS
devex001   codewind    4s    https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

During deployment, the operator creates the following items:

1. A service account
2. A deployment
3. A pod
4. A service
5. An ingress or route
6. A self signed TLS certificate
7. A storage claim
8. Any secrets

You can check these using standard Kubernetes or `oc` commands, such as:
```
$ kubectl get serviceaccount -n codewind
$ kubectl get deployments -n codewind
$ kubectl get pods -n codewind
$ kubectl get services -n codewind
$ kubectl get pvc -n codewind
```

These commands show each kind, as shown in the following examples:
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

During deployment of the Keycloak service, the operator configures the security realm as specified by the defaults config map.

Before you can install Codewind services, you need to be added to Keycloak. Use the Keycloak Admin web page to add new users.

Each Codewind deployment must be tied to an existing user account.

To see the Keycloak deployment running in the `codewind` namespace and capture its Access URL use the following command:

```
$ kubectl get keycloaks -n codewind
NAME       NAMESPACE   AGE     ACCESS
devex001   codewind    5m22s   https://codewind-keycloak-devex001.10.98.117.7.nip.io
```

By default, Keycloak is installed with an admin account where:
- Keycloak administrator username = admin
- Keycloak password = admin

Open the Keycloak Access URL in a browser and accept the self signed certificate warnings.

If you are unable to connect to Keycloak, check that the pod has started running and that storage is provisioned.

You can inspect the storage claim status with:
```
$ kubectl get pvc -n codewind
(check that the status shows **Bound** for the entry codewind-keycloak-pvc-{keycloakName})
```

Inspect the Keycloak pod status with:
```
$ kubect get pods -n codewind
(check that the returned codewind-keycloak-{keycloakName} entry shows **Running** with 1 container of 1 ready)
```

- Click **Administration Console** from the link provided.
- Log in to Keycloak using the Keycloak admin credentials.
  - username: admin
  - password: admin

**IMPORTANT:** After you log in, change the admin password by clicking the **Admin** link on the page. Then choose **Manage Account / Password** and set a new replacement administrator password.

- Switch back to the admin console using the link or log out and log back in to Keycloak as the admin user with your new admin password.

## Registering Codewind users

Ensure that the Realm is set to `Codewind` by clicking on the dropdown arrow on the page. Select **Codewind** if necessary, then:
- Click **Users**.
- Click **Add user**.
- Complete the **username** field.
- Complete the **email**, **Firstname**, and **Lastname** fields as required.
- Ensure **user enabled** is **On**.
- Click **Save**.

Assign an initial password to the user account by clicking **Credentials** and then add the initial password.

The field **Temporary** = **On will** requires users to change their passwords during first connection. Set **Temporary** = **Off will** to make this password valid for continuous use and not require changing on first connect.

Click **Set Password to save changes**.
Log out of the Keycloak admin page.

## Updating the Keycloak password in the operator secret

When the Codewind Operator needs to update Keycloak, it uses login credentials saved in a Kubernetes secret. By default during initial deployment, that secret has a user name and password of **admin.** If you changed your admin password in a previous step, you need to update the Keycloak secret to match.

The secret is installed in the same namespace as the `codewind` operator and is named `secret-keycloak-user-{keycloakname}`.

If you have an administration UI for your cluster, you can use it to locate the secret and edit the `keycloak-admin-password` field, or you can use the command line tools:

`$ kubectl edit secret secret-keycloak-user-{keycloakname} -n codewind`

or

`$ oc edit secret secret-keycloak-user-{keycloakname} -n codewind`

**Note:** Using the command line tools requires an extra step to base64 encode your password string before saving it into the secret. You can base64 encode your new password using this command:

```
$ echo -n 'myNewPassword' | base64
bXlOZXdQYXNzd29yZA==
```

Then, save `bXlOZXdQYXNzd29yZA==` as the value for `keycloak-admin-password` rather than the clear text `myNewPassword`.

## Deploy a Codewind instance

Deploying a new Codewind instance involves applying one last piece of `yaml`.

A copy of this `yaml` is available in `./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml`.

To deploy Codewind, change the following fields:
- **name**: A unique name for this deployment
- **keycloakDeployment**: The Keycloak service used for authentication
- **username**: A user name already registered in the specified Keycloak service

An example of valid `yaml` is:

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

<<<<<<< HEAD
**Note:**
- The **name** field is the name of the deployment and must be unique within the cluster. It should contain numbers and letters only, no spaces or punctuation.
- The **keycloakDeployment** field is the name of the Keycloak instance that provides authentication services. Keycloak must have already been provisioned and be running.
- The **username** field is the Keycloak registered user who will own this Codewind instance. Use alphanumeric characters only.
- The **loglevel** can be used to increase log levels of the Codewind pods. Allowed values one of either **error**, **warn**, **info**, **debug** or **trace**.
- The **storageSize** field sets the PVC size to 10GB.
=======
Note:

- the `name` field is the name of the deployment and must be unique within the cluster. It should contain numbers and letters only (no spaces or punctuation)
- the `keycloakDeployment` field is the name of the keycloak instance that will provide authentication services. Keycloak must have already been provisioned and be running.
- the `username` field is the keycloak registered user who will own this Codewind instance. (alpha numeric characters only)
- the `loglevel` can be used to increase log levels of the Codewind pods. allowed values one of either: error, warn, info, debug or trace
- the `storageSize` field sets the PVC size to 10GB (ensure there is a Persistent Volume available to service this storage request)

Apply this yaml and have the operator create and configure both Codewind and Keycloak with one command:
>>>>>>> 4ae61b2... Storage requirements

Apply this `yaml` and have the operator create and configure both Codewind and Keycloak with one command:
```
$ kubectl apply -f ./deploy/crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml
codewind.codewind.eclipse.org/codewind-k81235kj created
```

To view all the Codewind deployments in the `codewind` namespace:
```
$ kubectl get codewinds -n codewind
NAME                USERNAME   NAMESPACE   AGE   KEYCLOAK   AUTHSTATUS   ACCESSURL
jane1               jane       codewind    23m   devex001   Completed    https://codewind-gatekeeper-jane1.10.98.117.7.nip.io
```

The `kubectl get codewinds` command lists all the running Codewind deployments in the specified namespace. Each line represents a deployment and includes the user name of the developer it is assigned to, the Keycloak service name, and the auth config status. Most importantly, users need their Access URL, which they add to the IDE when creating a connection. Use the `-n` flag to target a specific namespace, for example, `-n codewind`.

**Note:** If the user was assigned a temporary password, they need to log in to Codewind from a browser and complete these next steps to set a new password and activate their account.

1. Open the gatekeeper Access URL obtained in the previous step for the Codewind deployment.
2. Log in using the provided user name and initial password.
3. Follow the prompts to change the password.
4. Proceed with setting up the IDE connection using the newly changed password.

## Removing a Codewind instance

To remove a Codewind instance, enter the following command where `<name>` is the name of the instance: 
`$ kubectl delete codewinds <name> -n codewind`

## Building the operator

To build the operator container image from source, move the cloned repo into your go directory, for example:
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

Before deploying the operator with any changes, modify the image field listed in the `./deploy/operator.yaml` file, setting it to the location of your built and pushed operator image.
