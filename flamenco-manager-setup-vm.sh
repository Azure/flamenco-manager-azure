#!/bin/bash

set -e

FMANAGER_VERSION="2.4.2"
FWORKER_VERSION="2.3"
BLENDER_VERSION="2.80-5ac7675f4c9c"
FFMPEG_VERSION="4.1.3"

WORKER_COMPONENTS_DIR="/mnt/flamenco-resources/apps"

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
FM_USER="flamenco-manager"
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
    https://builder.blender.org/download/blender-${BLENDER_VERSION}-linux-glibc224-x86_64.tar.bz2 \
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

BLENDER_DIR=blender-${BLENDER_VERSION}-linux-glibc224-x86_64
if [ ! -e $BLENDER_DIR ]; then
    echo "  - Blender ${BLENDER_VERSION} -> $WORKER_COMPONENTS_DIR"
    tar jxf $COMPONENTS_DIR/blender-${BLENDER_VERSION}-linux-glibc224-x86_64.tar.bz2 \
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
