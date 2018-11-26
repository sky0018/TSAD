#!/bin/bash

CURDIR=$(cd `dirname $0`; pwd)
if [ "X$1" != "X" ]; then
    RUNTIME_ROOT=$1
else
    RUNTIME_ROOT=${CURDIR}
fi

if [ "X$RUNTIME_ROOT" == "X" ]; then
    echo "There is no RUNTIME_ROOT support."
    echo "Usage: ./bootstrap.sh $RUNTIME_ROOT"
    exit -1
fi

if [ ! -f $CURDIR/settings.py ]; then
    echo "there is no settings.py exist."
    exit -1
fi

## If you modified this variable, don't forget to modity logfile in service.yml
LOG_DIR="$RUNTIME_ROOT/log/app"
if [ ! -d $LOG_DIR ]; then
    mkdir -p $LOG_DIR
fi

PRODUCT=$(cd $CURDIR; python -c "import settings; print settings.PRODUCT")
SUBSYS=$(cd $CURDIR; python -c "import settings; print settings.SUBSYS")
MODULE=$(cd $CURDIR; python -c "import settings; print settings.MODULE")
if [ -z "$PRODUCT" ] || [ -z "$SUBSYS" ] || [ -z "$MODULE" ]; then
    echo "Support PRODUCT SUBSYS MODULE PORT in settings.py"
    exit -1
fi

PSM=${PRODUCT}.${SUBSYS}.${MODULE}

GLOG_DIR=$CURDIR/ginex_log
if [ ! -d $GLOG_DIR ]; then
    mkdir -p $GLOG_DIR
fi

GOGC=40

GIN_MODE=release exec $CURDIR/bin/toutiao.microservice.tsad -psm=$PSM -log-dir=$GLOG_DIR -conf-dir=$CURDIR/conf/
