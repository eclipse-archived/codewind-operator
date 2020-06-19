#!/bin/bash

# Globals
ACTION=$1
FLG_INGRESS_DOMAIN=""
FLG_OC311=false
FLG_CW_USERNAME=""
FLG_CW_NAME=""

## Defaults
FLG_NAMESPACE="codewind"
FLG_AUTHSERVICE="devex001"

function helpOperatorFlags() {
  echo "  flags for 'operator' command:"
  echo "    -i                   ingress domain eg 172.15.41.1.nip.io"
  echo "    -o                   use when deploying the operator into an Openshift 3.11.x cluster"
}

function helpCodewindFlags() {
  echo "  flags for 'codewind' command:"
  echo "    -u                   registered username"
  echo "    -n                   name of new Codewind deployment"
}

function showHelp() {

  echo "NAME:"
  echo "   install.sh - deploys the codewind-operator or a new Codewind instance"
  echo ""
  echo "USAGE:"
  echo "   install.sh command [flags]"
  echo ""
  echo "COMMANDS:"
  echo "   operator              deploys the codewind-operator into the cluster"
  echo "   codewind              creates a new Codewind deployment in the cluster"
  echo ""
  echo "FLAGS:"

  helpOperatorFlags
  echo ""
  helpCodewindFlags
  echo ""
  echo "EXAMPLES:"
  echo "   install.sh operator -i 172.51.22.43.nip.io"
  echo "   install.sh codewind -n myinstance -u myaccount"
  echo ""
}

function installOperator() {
    echo ""
    echo "--------------------------------"
    echo "New Codewind-Operator deployment"
    echo "--------------------------------"
    echo ""

    if [ -z $FLG_INGRESS_DOMAIN ]
    then
      echo ""
      echo "REQUIRED FLAG: When installing the Codewind-Operator you must supply an ingress domain using the -i option."
      echo ""
      helpOperatorFlags
      echo ""
      exit
    fi

    echo "Ingress Domain: $FLG_INGRESS_DOMAIN"
    echo "Target Openshift 311: $FLG_OC311"

    echo "Creating Codewind namespace:"
    kubectl create namespace $FLG_NAMESPACE

    echo "Deploying Operator Service Account:"
    kubectl apply -f service_account.yaml

    echo "Deploying Operator RBAC Roles:"
    kubectl apply -f role.yaml

    echo "Deploying Operator RBAC Role Bindings:"
    kubectl apply -f role_binding.yaml

    echo "Deploying Operator Cluster Roles:"
    kubectl apply -f cluster_roles.yaml

    echo "Deploying Codewind Cluster Role Bindings:"
    kubectl apply -f cluster_role_binding.yaml
    echo ""

    cd crds
    if [[ $FLG_OC311 == true ]]
    then
    echo "Installing Custom Resource Definitions (CRD) for Openshift 3.11:"
    kubectl apply -f codewind.eclipse.org_keycloaks_crd-oc311.yaml
    kubectl apply -f codewind.eclipse.org_codewinds_crd-oc311.yaml
    else
    echo "Installing Custom Resource Definitions (CRD):"
    kubectl apply -f codewind.eclipse.org_keycloaks_crd.yaml
    kubectl apply -f codewind.eclipse.org_codewinds_crd.yaml
    fi

    cd ..

    echo "Creating Codewind configmap:"

    head -n17 codewind-configmap.yaml > custom-codewind-configmap.yaml
    echo "  ingressDomain: "$FLG_INGRESS_DOMAIN >> custom-codewind-configmap.yaml
    tail -n3 codewind-configmap.yaml >> custom-codewind-configmap.yaml

    kubectl apply -f custom-codewind-configmap.yaml
    rm -f custom-codewind-configmap.yaml

    echo "Deploying Codewind operator:"
    kubectl apply -f operator.yaml

    cd crds

    echo "Requesting a new Keycloak service"
    kubectl apply -f codewind.eclipse.org_v1alpha1_keycloak_cr.yaml

    cd ..

    echo "Reading Keycloak deployments"
    kubectl get keycloaks -n $FLG_NAMESPACE

    containerRunning=false
    lastContainerStatus="unknown"

    echo "Waiting for keycloak (may take a few minutes Pending->ContainerCreating->Running)"
    while [ $containerRunning != true ]
    do
      containerStatus=$(kubectl get pods --no-headers -n $FLG_NAMESPACE -l app=codewind-keycloak,authName=$FLG_AUTHSERVICE | awk '{print $3}')

      if [[ $lastContainerStatus != $containerStatus ]]
      then
        echo 'keycloak ' $containerStatus
        lastContainerStatus=$containerStatus
      fi

      if [[ $containerStatus == "Running" ]]
      then
        containerRunning=true
      else
        sleep 5
      fi
    done
    echo ""
    kubectl get keycloaks -n $FLG_NAMESPACE
}

function installCodewind() {
    echo "-----------------------"
    echo "New Codewind deployment"
    echo "-----------------------"

   if [[ -z $FLG_CW_NAME ]]
   then
    echo ""
    echo "REQUIRED FLAG: When installing a new Codewind deployment you must supply a unique name with the -n option"
    echo ""
    helpCodewindFlags
    echo ""
    exit
   fi

   if [[ -z $FLG_CW_USERNAME ]]
   then
    echo ""
    echo "REQUIRED FLAG: When installing the Codewind deployment you must supply a registered username with the -u option"
    echo ""
    helpCodewindFlags
    echo ""
    exit
   fi

    read -p "Have you remembered to set up '$FLG_CW_USERNAME' in the Keycloak directory (y/n)?" -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]
    then
    echo "Creating Codewind deployment"
    else
    echo "Aborting, no changes made"
    exit
    fi

    cd crds

    head -n15 codewind.eclipse.org_v1alpha1_codewind_cr.yaml > custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "  name: "$FLG_CW_NAME >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "  namespace: "$FLG_NAMESPACE >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "spec:"  >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "  keycloakDeployment: "$FLG_AUTHSERVICE  >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "  username: "$FLG_CW_USERNAME >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    tail -n2 codewind.eclipse.org_v1alpha1_codewind_cr.yaml >> custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml 
    kubectl apply -f custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    rm -f custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    cd ..
    echo ""

    containerRunning=false
    lastContainerStatus="unknown"
    echo "Waiting for codewind (may take a few minutes, expected phases: Pending->ContainerCreating->Running)"
    while [ $containerRunning != true ]
    do
       containerStatus=$(kubectl get pods --no-headers -n $FLG_NAMESPACE -l app=codewind-pfe,codewindName=$FLG_CW_NAME | awk '{print $3}')
       if [[ $lastContainerStatus != $containerStatus ]]
       then
         echo 'codewind: ' $containerStatus
         lastContainerStatus=$containerStatus
       fi

       if [[ $containerStatus == "Running" ]]
       then
         containerRunning=true
       else
         sleep 5
       fi
    done

    echo ""
    kubectl get codewinds $FLG_CW_NAME -n $FLG_NAMESPACE
    exit
}

echo ""

shift $(($OPTIND))
while getopts 'n:u:i:o' cmd
do
  case $cmd in
    i) FLG_INGRESS_DOMAIN=$OPTARG ;;
    o) FLG_OC311=true ;;
    n) FLG_CW_NAME=$OPTARG ;;
    u) FLG_CW_USERNAME=$OPTARG ;;
  esac
done

case "$ACTION" in
    'operator')
        installOperator ;;
    'codewind')
        installCodewind ;;
    *)
        showHelp ;;
esac

