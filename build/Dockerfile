# /*******************************************************************************
#  * Copyright (c) 2020 IBM Corporation and others.
#  * All rights reserved. This program and the accompanying materials
#  * are made available under the terms of the Eclipse Public License v2.0
#  * which accompanies this distribution, and is available at
#  * http://www.eclipse.org/legal/epl-v20.html
#  *
#  * Contributors:
#  *     IBM Corporation - initial API and implementation
#  *******************************************************************************/

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL vendor="Codewind" \
      name="Codewind Operator" \
      version="0.0.1" \
      summary="Codewind Operator" \
      description="An operator that helps manage the deployment and configuration of Codewind Remote instances on Kubernetes"

ENV OPERATOR=/usr/local/bin/codewind-operator \
    USER_UID=1001 \
    USER_NAME=codewind-operator

# install operator binary
COPY build/_output/bin/codewind-operator ${OPERATOR}

# install our custom config map
COPY deploy/codewind-configmap.yaml deploy/
COPY build/bin /usr/local/bin
COPY LICENSE /
COPY NOTICE.md /

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
