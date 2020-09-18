#!/bin/bash
# Copyright (c) 2020 Red Hat, Inc.

#LOG="/reults/output.txt"

#kubectl is not available in this env

#Need to install MCO Operator
#Need to create Object Store
#Need to create secret for Object Store
#Need to deploy MCO CR
#Need to wait till all deployment is done

check_mco_crd() {
      echo "******** Checking if MCO CRD is created" 
      MCOCRD=`oc get mco --no-headers=true|awk '{ print $1 }'`
      echo "$MCOCRD"
      if [[ "$MCOCRD" == "observability" ]]; then
            echo "Observality CRD is created"
      else
            echo "failure - Observability CRD is missing"  
            exit 1        
      fi        

}
check_mco_operator() {
      echo "******** Checking if MCO Operator is running" 
      MCOOPR=`oc get mco --no-headers=true|awk '{ print $1 }'`
      echo "$MCOCRD"
      if [[ "$MCOOPR" == "observability" ]]; then
            echo "MCO Operator is running"
      else
            echo "failure - MCO Operator is not running"  
            exit 1        
      fi        

}
create_mco_cr(){
      echo "******** Creating MCO CR and waiting for Obs. Install to complete" 
}

run_test(){
      echo "******** Running Tests..." 
}

check_mco_crd
if [ "$?" -ne 0 ]; then
      exit 1
fi      
check_mco_operator
if [ "$?" -ne 0 ]; then
      exit 1
fi
create_mco_cr
if [ "$?" -ne 0 ]; then
      exit 1
fi
run_test

#./example-client-go 
