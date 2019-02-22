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
#     AZ_BATCH_APP_PACKAGE_blender=/mnt/batch/tasks/applications/blender2.80-daily-2019-10-142019-01-15-09-19
#     AZ_BATCH_APP_PACKAGE_blender_2_80_daily_2019_10_14=/mnt/batch/tasks/applications/blender2.80-daily-2019-10-142019-01-15-09-19
#     AZ_BATCH_APP_PACKAGE_ffmpeg=/mnt/batch/tasks/applications/ffmpeg4.12019-01-15-09-16
#     AZ_BATCH_APP_PACKAGE_ffmpeg_4_1=/mnt/batch/tasks/applications/ffmpeg4.12019-01-15-09-16
#     AZ_BATCH_APP_PACKAGE_flamenco_worker=/mnt/batch/tasks/applications/flamenco-worker2.2.12019-01-15-08-52
#     AZ_BATCH_APP_PACKAGE_flamenco_worker_2_2_1=/mnt/batch/tasks/applications/flamenco-worker2.2.12019-01-15-08-52
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

groupadd --force flamenco  # --force makes sure it doesn't fail when the group already exists
adduser _azbatch flamenco
adduser $USER flamenco

if [ -z "${FLAMENCO_AZ_STORAGE_ACCOUNT}" ]; then
    echo +++ SKIPPING Preparing Blender Animation Studio infrastructure +++
else
    echo === Preparing Blender Animation Studio infrastructure ===
    mkdir -p /render

    # Remove existing entry for /render from fstab
    sed '/ \/render /d' -i /etc/fstab

    # Create a new entry so we're sure the storage account credentials are ok.
    echo "//${FLAMENCO_AZ_STORAGE_ACCOUNT}.file.core.windows.net/render /render cifs vers=3.0,username=${FLAMENCO_AZ_STORAGE_ACCOUNT},password=${FLAMENCO_AZ_STORAGE_KEY},dir_mode=0775,file_mode=0664,uid=_azbatch,forceuid,gid=_azbatchgrp,forcegid,sec=ntlmssp,mfsymlinks 0 0" >> /etc/fstab

    # If mounted, unmount.
    grep ' /render ' /proc/mounts && umount /render

    mount /render
fi

if [ -z "$AZ_BATCH_APP_PACKAGE_flamenco_worker" ]; then
    echo +++ SKIPPING Symlinking applications +++
else
    echo === Symlinking applications ===
    ln -sf $AZ_BATCH_APP_PACKAGE_flamenco_worker /mnt/batch/tasks/applications/flamenco-worker
    ln -sf $AZ_BATCH_APP_PACKAGE_blender /mnt/batch/tasks/applications/blender
    ln -sf $AZ_BATCH_APP_PACKAGE_ffmpeg /mnt/batch/tasks/applications/ffmpeg
fi

echo === Installing Azure Preempt Monitor service ===
systemctl stop azure-preempt-monitor.service || true
cp /mnt/flamenco-resources/azure-preempt-monitor /usr/local/bin
cp /mnt/flamenco-resources/azure-preempt-monitor.service /etc/systemd/system
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

ExecStart=$AZ_BATCH_APP_PACKAGE_flamenco_worker/flamenco-worker
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
