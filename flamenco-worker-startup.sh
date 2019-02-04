#!/bin/bash

# Abort when an error occurs.
set -e

# Environment variables like these are set during the startup task:
#
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

echo === Symlinking applications ===
ln -s $AZ_BATCH_APP_PACKAGE_flamenco_worker /mnt/batch/tasks/applications/flamenco-worker
ln -s $AZ_BATCH_APP_PACKAGE_blender /mnt/batch/tasks/applications/blender
ln -s $AZ_BATCH_APP_PACKAGE_ffmpeg /mnt/batch/tasks/applications/ffmpeg

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
sudo cp flamenco-worker.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable flamenco-worker

echo === Starting Flamenco Worker service ===
sudo systemctl start flamenco-worker

echo === Startup Task Complete ===
