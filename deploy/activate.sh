#!/bin/bash

#### setup root stuff after install

USER=${1:-vagrant}

cat << EOF > /lib/systemd/system/tendermint.service
[Unit]
Description=tendermint
After=bov.service

[Service]
Type=simple
Restart=always
RestartSec=5s
User=${USER}
Environment=TM_HOME=/home/${USER}/.bov
ExecStart=/usr/local/bin/tendermint node

[Install]
WantedBy=multi-user.target
EOF

cat << EOF > /lib/systemd/system/bov.service
[Unit]
Description=bov

[Service]
Type=simple
Restart=always
RestartSec=5s
User=${USER}
ExecStart=/usr/local/bin/bov start

[Install]
WantedBy=multi-user.target
EOF

mv /home/${USER}/go/bin/bov /usr/local/bin/bov
mv /home/${USER}/go/bin/tendermint /usr/local/bin/tendermint

systemctl enable bov
systemctl enable tendermint
