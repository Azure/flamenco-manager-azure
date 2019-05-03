#!/bin/bash

set -e

FMANAGER_VERSION="2.4.3-106-g4a64138"
FWORKER_VERSION="2.3.1"
FFMPEG_VERSION="4.1.3"
AZURE_PREEMPT_MONITOR_VERSION="1.1"

BLENDER_TAR_BZ2=$(curl -s https://builder.blender.org/download/ | grep -oE "(blender-2\.80-[a-z0-9]+-linux-glibc[0-9]+-x86_64\.tar\.bz2)")
BLENDER_TAR="${BLENDER_TAR_BZ2%.*}"
BLENDER_DIR="${BLENDER_TAR%.*}"

WORKER_COMPONENTS_DIR="/mnt/flamenco-resources/apps"
MY_DIR="$(dirname "$(readlink -f "$0")")"

## Set up the firewall via UWF
sudo -s <<EOT
set -e
cd /etc/ufw

# Make a backup before we modify the file.
cp before.rules before.rules~$(date --iso-8601=sec)

# Filter out everything between '*nat' and 'COMMIT':
sed '/^*nat/,/COMMIT/d' -i before.rules
cat >>before.rules <<EOF
*nat
# Forward 80 -> 8080 and 443 -> 8443, so that Flamenco Manager does not need root.
:PREROUTING ACCEPT [0:0]
-A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination :8080
-A PREROUTING -i eth0 -p tcp --dport 443 -j DNAT --to-destination :8443
COMMIT
EOF

ufw allow OpenSSH
ufw allow proto tcp from any to any port 80
ufw allow proto tcp from any to any port 443
ufw allow proto tcp from any to any port 8080
ufw allow proto tcp from any to any port 8443
echo y | ufw enable
ufw reload  # just in case it already was enabled
EOT


## Install system packages
sudo -s <<EOT
apt-get install -qy software-properties-common
apt-key adv --recv 9DA31620334BD75D9DCB49F368818C72E52529D4
cat > /etc/apt/sources.list.d/mongodb-org-4.0.list <<END
deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu bionic/mongodb-org/4.0 multiverse
END
apt-get update -q
DEBIAN_FRONTEND=noninteractive apt-get install -qy \
    -o APT::Install-Recommends=false -o APT::Install-Suggests=false \
    imagemagick mongodb-org-server
systemctl enable mongod.service
EOT

echo
echo "Setting up /etc/fstab"
# Remove any old reference to the SMB shares
grep -v 'file.core.windows.net' < /etc/fstab > stripped-fstab
# fstab-smb is uploaded by the Go code before uploading this script.
cat stripped-fstab fstab-smb > new-fstab
sudo cp new-fstab /etc/fstab

# Make all directories that are used as SMB mount points.
sudo mkdir -p $(awk '{ print $2 }' < fstab-smb)
sudo mount -a

echo "Setting up user for Flamenco Manager"
FM_USER=flamanager
if ! getent passwd $FM_USER >/dev/null 2>&1; then
    sudo useradd --groups flamenco --create-home --no-user-group $FM_USER
fi
MANAGER_HOME=$(getent passwd $FM_USER | cut -d: -f6)


echo "Downloading Components"
mkdir -p $HOME/flamenco-components
cd $HOME/flamenco-components
COMPONENTS_DIR=$(pwd)

wget -qN \
    https://www.flamenco.io/download/flamenco-manager-${FMANAGER_VERSION}-linux.tar.gz \
    https://www.flamenco.io/download/flamenco-worker-${FWORKER_VERSION}-linux.tar.gz \
    https://flamenco.io/download/azure-preempt-monitor/azure-preempt-monitor-v${AZURE_PREEMPT_MONITOR_VERSION}-linux.tar.gz \
    https://builder.blender.org/download/${BLENDER_TAR_BZ2} \
    https://johnvansickle.com/ffmpeg/releases/ffmpeg-${FFMPEG_VERSION}-amd64-static.tar.xz


echo "Installing Components"

# Flamenco Manager
cd $MANAGER_HOME
if [ ! -e flamenco-manager-${FMANAGER_VERSION} ]; then
    echo "  - Flamenco Manager ${FMANAGER_VERSION} -> $MANAGER_HOME"
    sudo -u $FM_USER tar zxf $COMPONENTS_DIR/flamenco-manager-${FMANAGER_VERSION}-linux.tar.gz
    sudo -u $FM_USER rm -f flamenco-manager
    sudo -u $FM_USER ln -s flamenco-manager-${FMANAGER_VERSION} flamenco-manager
else
    echo "  - Flamenco Manager ${FMANAGER_VERSION} [already installed]"
fi
sudo cp $MY_DIR/flamenco-manager.service /etc/systemd/system/
sudo systemctl daemon-reload


# Flamenco Worker components (Worker itself + apps)
# --atime-preserve=system --touch is necessary to extract on an SMB share without errors/warnings.
mkdir -p $WORKER_COMPONENTS_DIR
cd $WORKER_COMPONENTS_DIR

if [ ! -e flamenco-worker-${FWORKER_VERSION} ]; then
    echo "  - Flamenco Worker ${FWORKER_VERSION} -> $WORKER_COMPONENTS_DIR"
    tar zxf $COMPONENTS_DIR/flamenco-worker-${FWORKER_VERSION}-linux.tar.gz \
        --atime-preserve=system --touch
    rm -f flamenco-worker
    ln -s flamenco-worker-${FWORKER_VERSION} flamenco-worker
else
    echo "  - Flamenco Worker ${FWORKER_VERSION} [already installed]"
fi

if [ ! -e azure-preempt-monitor-v${AZURE_PREEMPT_MONITOR_VERSION} ]; then
    echo "  - Azure Preempt Monitor ${AZURE_PREEMPT_MONITOR_VERSION} -> $WORKER_COMPONENTS_DIR"
    tar zxf $COMPONENTS_DIR/azure-preempt-monitor-v${AZURE_PREEMPT_MONITOR_VERSION}-linux.tar.gz \
        --atime-preserve=system --touch
    rm -f azure-preempt-monitor
    ln -s azure-preempt-monitor-v${AZURE_PREEMPT_MONITOR_VERSION} azure-preempt-monitor
else
    echo "  - Azure Preempt Monitor ${AZURE_PREEMPT_MONITOR_VERSION} [already installed]"
fi

BLENDER_VERSION=${BLENDER_DIR/blender-}
if [ ! -e $BLENDER_DIR ]; then
    echo "  - Blender ${BLENDER_VERSION} -> $WORKER_COMPONENTS_DIR"
    tar jxf $COMPONENTS_DIR/${BLENDER_TAR_BZ2} \
        --atime-preserve=system --touch
    rm -f blender
    ln -s $BLENDER_DIR blender
else
    echo "  - Blender ${BLENDER_VERSION} [already installed]"
fi

FFMPEG_DIR=ffmpeg-${FFMPEG_VERSION}-amd64-static
if [ ! -e $FFMPEG_DIR ]; then
    echo "  - FFmpeg ${FFMPEG_VERSION} -> $WORKER_COMPONENTS_DIR"
    tar Jxf $COMPONENTS_DIR/ffmpeg-${FFMPEG_VERSION}-amd64-static.tar.xz \
        --atime-preserve=system --touch
    rm -f ffmpeg
    ln -s $FFMPEG_DIR ffmpeg
else
    echo "  - FFmpeg ${FFMPEG_VERSION} [already installed]"
fi

# Configure Flamenco Manager
cd $MANAGER_HOME
if [ ! -e flamenco-manager.yaml ]; then
    echo "Installing default flamenco-manager.yaml"
    sudo -u $FM_USER cp $MY_DIR/default-flamenco-manager.yaml flamenco-manager.yaml
else
    echo "flamenco-manager.yaml already exists, not touching"
fi
if [ -e $MY_DIR/client_credentials.json ]; then
    sudo -u $FM_USER cp $MY_DIR/client_credentials.json azure_credentials.json
    sudo -u $FM_USER chmod 600 azure_credentials.json
    rm $MY_DIR/client_credentials.json
fi

# Configure Flamenco Worker
cd /mnt/flamenco-resources
echo "Configuring Flamenco Worker"
cp $MY_DIR/flamenco-worker.cfg ./flamenco-worker.cfg
cp $MY_DIR/flamenco-worker-startup.sh ./flamenco-worker-startup.sh

# Start services
echo "Starting services"
sudo systemctl enable mongod
sudo systemctl start mongod
sudo systemctl enable flamenco-manager
sudo systemctl restart flamenco-manager
