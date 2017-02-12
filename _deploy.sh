#!/usr/bin/env bash

# inject the variables required to deploy (but don't track them in git)
source __config.sh

# make sure we have all the variables we need to deploy
if [[ -z ${USER} || -z ${GROUP} || -z ${SERVER} ]]; then
	echo -e "\nConfig values required for 'USER', 'GROUP' and 'SERVER'\n"
	exit 1
fi

# do the deploy
echo -e "\ndeploying ultitracker..."

echo -e "\ncopying binary..."
scp ./bin/ultitracker_linux_amd64 ${USER}@${SERVER}:/home/${USER}/ultitracker

echo -e "\ncopying app config..."
scp ./config.* ${USER}@${SERVER}:/home/${USER}/ultitracker

echo -e "\ncopying supervisor config..."
scp ./supervisor.conf ${USER}@${SERVER}:/home/${USER}/ultitracker

echo -e "\ninstalling update..."
ssh ${USER}@${SERVER} << EOM
mv /home/${USER}/ultitracker/ultitracker_linux_amd64 /home/${USER}/ultitracker/ultitracker
sudo chown ${USER}:${GROUP} /home/${USER}/ultitracker/ultitracker
sudo mv /home/${USER}/ultitracker/supervisor.conf /etc/supervisor/conf.d/ultitracker.conf
supervisorctl restart ultitracker
EOM

echo -e "\nservice restarted..."
