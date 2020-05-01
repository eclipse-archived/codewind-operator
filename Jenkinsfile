#!groovy

pipeline {

    agent {
        kubernetes {
              label 'go-pod-1-13-buster'
            yaml """
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: go
    image: golang:1.13-buster
    tty: true
    command:
    - cat
    resources:
      limits:
        memory: "4Gi"
        cpu: "1"
      requests:
        memory: "4Gi"
        cpu: "1"
"""
        }
    }

    options {
        timestamps()
        skipStagesAfterUnstable()
    }

    environment {
        CODE_DIRECTORY_FOR_GO = 'src/github.com/eclipse/codewind-operator'
        DEFAULT_WORKSPACE_DIR_FILE = 'temp_default_dir'
        CODECOV_TOKEN = credentials('codecov-token')
    }

    stages {

        stage ('Build') {

            // This when clause disables Tagged build
            when {
                beforeAgent true
                not {
                    buildingTag()
                }
            }

            steps {
                container('go') {
                    sh '''#!/bin/bash

                        echo "Starting setup for build.....: GOPATH=$GOPATH"
                        set -x
                        whoami

                        # add the base directory to the gopath
                        DEFAULT_CODE_DIRECTORY=$PWD
                        cd ../..
                        export GOPATH=$GOPATH:$(pwd)

                        # create a new directory to store the code for go compile
                        if [ -d $CODE_DIRECTORY_FOR_GO ]; then
                            rm -rf $CODE_DIRECTORY_FOR_GO
                        fi
                        mkdir -p $CODE_DIRECTORY_FOR_GO
                        cd $CODE_DIRECTORY_FOR_GO

                        # copy the code into the new directory for go compile
                        cp -r $DEFAULT_CODE_DIRECTORY/* .
                        echo $DEFAULT_CODE_DIRECTORY >> $DEFAULT_WORKSPACE_DIR_FILE

                        # go cache setup
                        mkdir .cache
                        cd .cache
                        mkdir go-build
                        cd ../

                        # now compile the code
                        export HOME=$JENKINS_HOME
                        export GOCACHE=/home/jenkins/agent/$CODE_DIRECTORY_FOR_GO/.cache/go-build
                        export GOARCH=amd64
                        export GO111MODULE=on
                        go mod tidy

                        # This is copied from the operator SDK build command.
                        go build -o /home/jenkins/agent/$CODE_DIRECTORY_FOR_GO/build/_output/bin/codewind-operator -gcflags all=-trimpath=/home/jenkins/agent/src/github.com/eclipse -asmflags all=-trimpath=/home/jenkins/agent/src/github.com/eclipse github.com/eclipse/codewind-operator/cmd/manager

                        # clean up the cache directory
                        cd ../../
                        rm -rf .cache
                    '''
                    dir('/home/jenkins/agent/src/github.com/eclipse/codewind-operator/build/_output/bin') {
                        sh 'echo "Stashing: $(find .)"'
                        stash includes: 'codewind-operator', name: 'operator-binary'
                    }
                }
            }
        }

        stage('Build Image') {
            agent {
                label "docker-build"
            }
            // This when clause disables Tagged build
            when {
                beforeAgent true
                not {
                    buildingTag()
                }
            }

            options {
                timeout(time: 10, unit: 'MINUTES')
            }

            steps {
                echo 'Building Operator Docker Image'
                withDockerRegistry([url: 'https://index.docker.io/v1/', credentialsId: 'docker.com-bot']) {
                    unstash "operator-binary"
                    script {
                        sh '''#!/usr/bin/env bash
                            set -x
                            # Just check the operator binary is still there.
                            ls -lrt codewind-operator
                            mkdir -p build/_output/bin
                            # Put the unstashed binary back in the right location.
                            mv codewind-operator build/_output/bin
                            docker build -f build/Dockerfile -t eclipse/codewind-operator-amd64:latest .

                            # If CHANGE_TARGET is set then this is a PR so don't push to master.
                            if [[ -z "$CHANGE_TARGET" ]]; then
                                export TAG=$GIT_BRANCH
                                if [[ "$GIT_BRANCH" = "master" ]]; then
                                    TAG="latest"
                                fi

                                echo "Publish Eclipse Codewind operator $TAG"

                                docker tag eclipse/codewind-operator-amd64:latest eclipse/codewind-operator-amd64:$TAG
                                docker push eclipse/codewind-operator-amd64:$TAG

                                if [[ $GIT_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then
                                    IFS='.' # set '.' as delimiter
                                    read -ra TOKENS <<< "$GIT_BRANCH"
                                    IFS=' ' # reset delimiter
                                    export TAG_CUMULATIVE=${TOKENS[0]}.${TOKENS[1]}

                                    echo "Publish Eclipse Codewind operator $TAG_CUMULATIVE"

                                    docker tag eclipse/codewind-operator-amd64:latest eclipse/codewind-operator-amd64:$TAG_CUMULATIVE
                                    docker push eclipse/codewind-operator-amd64:$TAG_CUMULATIVE
                                fi
                            fi
                        '''
                    }
                }
                echo 'End of Building Operator Docker Image stage'
            }
        }
    }

    post {
        success {
            echo 'Build SUCCESS'
        }
    }
}
