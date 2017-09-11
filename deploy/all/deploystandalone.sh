#!/bin/sh

# This script deploys the SafeHarborServer in a standalone test mode that does not
# attempt to access redis, clair, postgres, or a registry.

# MANUAL STEP:
# Log into AWS container registry:
# Get the AWS container registry login command by executing "aws ecr get-login" locally.
# Then paste that command into the development env shell.

# DEPLOY SAFE HARBOR SERVER AND ITS COMPONENTS----------------------------------

# Get images:
sudo docker pull $SafeHarborImageName

# Start SafeHarborServer.
sudo docker run --net=host -d -p $SafeHarborPort:$SafeHarborPort \
	-v $DataVolMountPoint:/safeharbor/data \
	-v /var/run/docker.sock:/var/run/docker.sock \
	$SafeHarborImageName /safeharbor/safeharbor \
	-debug -port=$SafeHarborPort -secretkey=jafakeu9s3ls -toggleemail -stubs -noregistry -inmem -host=$SafeHarborServerHost


# For debugging:
# Start container but don't start SafeHarborServer.
# sudo docker run --net=host -it -p $SafeHarborPort:$SafeHarborPort -w=/safeharbor/ -v $DataVolMountPoint:/safeharbor/data -v /var/run/docker.sock:/var/run/docker.sock $SafeHarborImageName bash