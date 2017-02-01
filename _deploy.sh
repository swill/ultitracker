#!/usr/bin/env bash

# inject the variables required to deploy (but don't track them in git)
source __config.sh

# make sure we have all the variables we need to deploy
if [[ -z ${USER} || -z ${GROUP} || -z ${SERVER} ]]; then
	echo -e "\nConfig values required for 'USER', 'GROUP' and 'SERVER'\n"
	exit 1
fi

# get the 'team' being deployed from the first positional argument
TEAM=$1
if [[ -z ${TEAM} ]]; then
	echo -e "\nThe 'team' positional argument is required to deploy...\n"
	exit 1
fi

# do the deploy
echo -e "\ndeploying '${TEAM}'..."

echo -e "\ncopying binary..."
scp ./bin/ultitracker_linux_amd64 ${USER}@${SERVER}:/home/${USER}/ultitracker_${TEAM}

echo -e "\ncopying app config..."
scp ./config/${TEAM}/config.* ${USER}@${SERVER}:/home/${USER}/ultitracker_${TEAM}

echo -e "\ncopying supervisor config..."
scp ./config/${TEAM}/supervisor.conf ${USER}@${SERVER}:/home/${USER}/ultitracker_${TEAM}

echo -e "\ninstalling update..."
ssh ${USER}@${SERVER} << EOM
mv /home/${USER}/ultitracker_${TEAM}/ultitracker_linux_amd64 /home/${USER}/ultitracker_${TEAM}/ultitracker_${TEAM}
sudo chown ${USER}:${GROUP} /home/${USER}/ultitracker_${TEAM}/ultitracker_${TEAM}
sudo mv /home/${USER}/ultitracker_${TEAM}/supervisor.conf /etc/supervisor/conf.d/ultitracker_${TEAM}.conf
supervisorctl restart ultitracker_${TEAM}
EOM

echo -e "\nservice restarted..."
