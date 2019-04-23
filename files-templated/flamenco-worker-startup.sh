#!/bin/bash

# Abort when an error occurs.
set -e

# Environment variables like these are set during the startup task.
#
# These are *NOT* available when SSH'ing into the machine, so that is why some
# parts of this script check for existence of those variables before using
# them.
#
#     FLAMENCO_AZ_STORAGE_ACCOUNT=saflamenco
#     FLAMENCO_AZ_STORAGE_KEY=afdliGF3ADdsf4f98fvklcvh1/4+1f93FBA==
#     AZ_BATCH_ACCOUNT_NAME=flamenco
#     AZ_BATCH_ACCOUNT_URL=https://flamenco.westeurope.batch.azure.com/
#     AZ_BATCH_CERTIFICATES_DIR=/mnt/batch/tasks/startup/certs
#     AZ_BATCH_NODE_ID=tvm-383584635_1-20190115t092314z
#     AZ_BATCH_NODE_IS_DEDICATED=true
#     AZ_BATCH_NODE_ROOT_DIR=/mnt/batch/tasks
#     AZ_BATCH_NODE_SHARED_DIR=/mnt/batch/tasks/shared
#     AZ_BATCH_NODE_STARTUP_DIR=/mnt/batch/tasks/startup
#     AZ_BATCH_POOL_ID=je-moeder-47
#     AZ_BATCH_TASK_DIR=/mnt/batch/tasks/startup
#     AZ_BATCH_TASK_USER=_azbatchtask_start
#     AZ_BATCH_TASK_USER_IDENTITY=TaskNonAdmin
#     AZ_BATCH_TASK_WORKING_DIR=/mnt/batch/tasks/startup/wd
#     HOME=/mnt/batch/tasks/startup/wd
#     PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/mnt/batch/tasks/shared:/mnt/batch/tasks/startup/wd
#     PWD=/mnt/batch/tasks/startup/wd
#     SHLVL=1
#     USER=_azbatchtask_start
#     _=/usr/bin/env

echo -n DATE: ; date
echo -n ID: ; id
echo -n UMASK: ; umask
echo -n PWD: ; pwd
echo
echo ENV; env | sort
echo
echo

if [ -z "${AZ_BATCH_TASK_USER}" ]; then
    echo +++ SKIPPING Installing Requirements to run Blender +++
else
    echo === Installing Requirements to run Blender ===
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install libgl1-mesa-dev libglu1-mesa-dev libx11-dev libxi6 libxrender1 -y
fi

groupadd --force {{ .UnixGroupName }}  # --force makes sure it doesn't fail when the group already exists
adduser _azbatch {{ .UnixGroupName }}
adduser $USER {{ .UnixGroupName }}

echo === Preparing SMB shares ===
cat > fstab-smb <<EOT
{{ .FSTabForStorage }}
EOT
(
    grep -v 'file.core.windows.net' < /etc/fstab
    echo "# Azure SMB shares from file.core.windows.net:"
    cat fstab-smb
) > fstab-new
sudo cp fstab-new /etc/fstab
sudo mkdir -p $(awk '{ print $2 }' < fstab-smb)
# Mount all SMB mountpoints, except 'flamenco-resources' -- it's already mounted and somehow it can get mounted twice.
awk '{ print $2 }' < fstab-smb | grep -v flamenco-resources | sudo xargs -n1 mount

echo === Installing Azure Preempt Monitor service ===
systemctl stop azure-preempt-monitor.service || true
cp /mnt/flamenco-resources/apps/azure-preempt-monitor/azure-preempt-monitor /usr/local/bin
cp /mnt/flamenco-resources/apps/azure-preempt-monitor/azure-preempt-monitor.service /etc/systemd/system
echo "daemon   ALL = NOPASSWD: /bin/systemctl" > /etc/sudoers.d/50-azure-preempt-monitor
chmod 755 /usr/local/bin/azure-preempt-monitor
systemctl daemon-reload
systemctl enable azure-preempt-monitor.service
systemctl start azure-preempt-monitor.service

if [ -z "$AZ_BATCH_NODE_SHARED_DIR" ]; then
    echo +++ SKIPPING Setting up Flamenco Worker +++
else
    echo === Setting up Flamenco Worker ===
    cp /mnt/flamenco-resources/flamenco-worker.cfg $AZ_BATCH_NODE_SHARED_DIR

    echo === Installing Flamenco Worker service ===
    cat > flamenco-worker.service <<EOT
# systemd service description for Flamenco Worker

[Unit]
Description=Flamenco Worker
Documentation=https://flamenco.io/
After=network-online.target

[Service]
Type=simple

ExecStart=/mnt/flamenco-resources/apps/flamenco-worker/flamenco-worker
WorkingDirectory=$AZ_BATCH_NODE_SHARED_DIR
User=_azbatch
Group=_azbatchgrp

RestartPreventExitStatus=SIGUSR1 SIGUSR2
Restart=always
RestartSec=1s

EnvironmentFile=-/etc/default/locale

[Install]
WantedBy=multi-user.target
EOT
    cp flamenco-worker.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable flamenco-worker
fi

echo === Starting Flamenco Worker service ===
systemctl start flamenco-worker

echo === Startup Task Complete ===
