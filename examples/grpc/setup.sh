#!/bin/sh

###############################################################################
PATH=/bin:/usr/bin:/usr/sbin:/usr/etc:/sbin:/etc
tempdir="/tmp"

umask 022       

censored_ls() {
    ls "$@" | egrep -v '^\.|/\.|CVS|RCS|SCCS|linux\.d|solaris\.d|hp_ux\.d|example|service'
}               
        
compare_or_replace() {
    (cmp $2 $3 >/dev/null 2>&1 && echo Skipping $3...) || { 
        echo Updating $3...
        rm -f $tempdir/junk || exit 1
        cp $2 $tempdir/junk || exit 1
        chmod $1 $tempdir/junk || exit 1
        mv -f $tempdir/junk $3 || exit 1
        chmod $1 $3 || exit 1
    }   
}    
###############################################################################
RPATH=
guess_os() {
    os_name=`uname -s`
    os_type=`uname -p`
    case $os_name in
        Linux)
    case $os_type in
        x86_64)
            RPATH="linux64"
            ;;
        i686)
            RPATH="linux32"
            ;;
        *)
            echo "unknown OS - $os_name $os_type"
            exit 1
            ;;
        esac
        ;;
    SunOS)
        case $os_type in
    	i386)
            RPATH="sunos_x86"
            ;;
        *)
            echo "unknown OS - $os_name $os_type"
            exit 1
            ;;
        esac
        ;;
    FreeBSD)
        RPATH="freebsd"
        ;;
    Darwin)
        RPATH="macos"
        ;;
    *)
        echo "unknown OS - $os_name $os_type"
        exit 1
        ;;
    esac
}

create_path()
{
    test -d $1 || mkdir -p $1 || {
        echo "can't mkdir $1"
        exit 1
    }
}

copy_file()
{
    test -f $2 && {
        compare_or_replace $1 $2 $3 || {
            echo "copy file: $2 error"
            exit 1
        }
    }
}

install_file()
{
    rm -f $tempdir/junk2 || {
        echo "can't remove file: $tempdir/junk2"
        exit 1
    }
    test -f $2 && {
        cat $2 | sed -e 's;{install_path};'$INSTALL_PATH';;' >$tempdir/junk2 || {
        echo "can't create file: $tempdir/junk2"
        exit 1
    }
    compare_or_replace $1 $tempdir/junk2 $3 || {
        echo "can't move to file: $3"
        exit 1
        }
    }
    rm -f $tempdir/junk2 || {
        echo "can't remove file: $tempdir/junk2"
        exit 1
    }
}

###############################################################################
INSTALL_PATH=

if [ $# -lt 1 ]
then
#	echo "parameter not enougth($#)"
    echo "usage:$0 install_path"
    exit 1
fi

if [ $# -eq 2 ]
then
    PREFIX_PATH=$1
    INSTALL_PATH=$2
else
    INSTALL_PATH=$1
    PREFIX_PATH=
fi

case $INSTALL_PATH in
/*) ;;
no) ;;
*) echo Error: $INSTALL_PATH should be an absolute path name. 1>&2; exit 1;;
esac

echo Installing to $INSTALL_PATH...

BIN_PATH=$PREFIX_PATH$INSTALL_PATH/bin
SBIN_PATH=$PREFIX_PATH$INSTALL_PATH/sbin
CONF_PATH=$PREFIX_PATH$INSTALL_PATH/conf
VAR_PATH=$PREFIX_PATH$INSTALL_PATH/var

###############################################################################
create_all_path()
{
    create_path $INSTALL_PATH
    create_path $BIN_PATH
    create_path $SBIN_PATH
    create_path $CONF_PATH
    create_path $VAR_PATH
    create_path $VAR_PATH/log
    create_path $VAR_PATH/pid

    chmod 1777 $VAR_PATH/log
}

copy_all_file()
{
    copy_file a+x,go+rx grpc-server $SBIN_PATH/grpc-server
	install_file a+x,go-wrx grpc-server.cf $CONF_PATH/grpc-server.cf
    install_file a+x,go+rx start.sh $BIN_PATH/start.sh
    install_file a+x,go+rx stop.sh $BIN_PATH/stop.sh
    install_file a+x,go+rx status.sh $BIN_PATH/status.sh
}

guess_os
create_all_path
copy_all_file

###############################################################################
