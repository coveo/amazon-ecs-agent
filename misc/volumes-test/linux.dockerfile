# Copyright 2014-2015 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.
FROM busybox:1.29.3
MAINTAINER Amazon Web Services, Inc.

RUN mkdir /data
RUN echo "test" > /data/test-file

VOLUME ["/data"]

ENTRYPOINT ["sh", "-c"]
CMD ["trap \"exit 0\" SIGTERM SIGINT; while true; do sleep 1; done"]
