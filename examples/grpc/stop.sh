#!/bin/sh
MASTER_PATH=/opt/soft/acl-master
$MASTER_PATH/bin/master_ctl -s $MASTER_PATH/var/public/master.sock -f {install_path}/conf/grpc-server.cf -a stop
