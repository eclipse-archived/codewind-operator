#!/bin/bash

# Globals
ACTION=$1
FLG_INGRESS_DOMAIN=""
FLG_OC311=false
FLG_CW_USERNAME=""
FLG_CW_NAME=""

function installOperator() {
    echo ""
    echo "########################"
    echo "   Codewind Operator"
    echo "########################"
    echo ""

    if [ -z $FLG_INGRESS_DOMAIN ]
    then
      echo "When installing the Codewind-Operator you must supply an ingress domain using the -i option"
      exit
    fi

    echo "Ingress Domain: $FLG_INGRESS_DOMAIN"
    echo "Target Openshift 311: $FLG_OC311"

    echo "Creating Codewind namespace:"
    kubectl create namespace codewind

    echo "Deploying Operator Service Account:"
    kubectl create -f ./service_account.yaml

    echo "Deploying Operator RBAC Roles:"
    kubectl create -f ./role.yaml

    echo "Deploying Operator RBAC Role Bindings:"
    kubectl create -f ./role_binding.yaml

    echo "Deploying Operator Cluster Roles:"
    kubectl create -f ./cluster_roles.yaml

    echo "Deploying Codewind Cluster Role Bindings:"
    kubectl create -f ./cluster_role_binding.yaml
    echo ""

    if [[ $FLG_OC311 == true ]]
    then
    echo "Installing Custom Resource Definitions (CRD) for Openshift 3.11:"
    kubectl create -f ./crds/codewind.eclipse.org_keycloaks_crd-oc311.yaml
    kubectl create -f ./crds/codewind.eclipse.org_codewinds_crd-oc311.yaml
    else
    echo "Installing Custom Resource Definitions (CRD):"
    kubectl create -f ./crds/codewind.eclipse.org_keycloaks_crd.yaml
    kubectl create -f ./crds/codewind.eclipse.org_codewinds_crd.yaml
    fi

    echo "Creating Codewind configmap:"
    cp ./codewind-configmap.yaml ./custom-codewind-configmap.yaml
    sed -i "" "s|codewind.apps-crc.testing|$FLG_INGRESS_DOMAIN|g" custom-codewind-configmap.yaml
    kubectl create -f ./custom-codewind-configmap.yaml
    rm -f ./custom-codewind-configmap.yaml

    echo "Deploying Codewind operator:"
    kubectl create -f ./operator.yaml

    echo "Requesting a new Keycloak service"
    kubectl create -f ./crds/codewind.eclipse.org_v1alpha1_keycloak_cr.yaml

    echo "Reading Keycloak deployments"
    kubectl get keycloaks -n codewind

}

function installCodewind() {
    echo "----------------------------------"
    echo "Install a new Codewind deployment"
    echo "----------------------------------"

   if [[ -z $FLG_CW_NAME ]]
   then
    echo "When installing a new Codewind deployment you must supply a unique name with the -n option"
    exit
   fi

   if [[ -z $FLG_CW_USERNAME ]]
   then
    echo "When installing the Codewind deployment you must supply a registered username with the -u option"
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

    echo "Creating Codewind deployment"
    cp ./crds/codewind.eclipse.org_v1alpha1_codewind_cr.yaml ./custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    sed -i "" "s|name: jane1|name: $FLG_CW_NAME|g" ./custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    sed -i "" "s|username: jane|username: $FLG_CW_USERNAME|g" ./custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    kubectl create -f ./custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    rm -f ./custom-codewind.eclipse.org_v1alpha1_codewind_cr.yaml
    echo "Check status using the command 'kubectl get codewinds'"
    echo ""
    echo ""
    exit
}

echo "############################"
echo "Codewind Operator install.sh"
echo "############################"

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
esac
