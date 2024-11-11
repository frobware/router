These instructions explain how to build OpenSSL 1.1 and the
traditional libcrypt (specifically libcrypt.so.1 from older versions
of glibc). This enables us to run versions of OpenShift's HAProxy that
were linked against OpenSSL 1.1 and the older libcrypt on RHEL/Fedora
systems where the runtime platform now only provides OpenSSL 3 and the
newer libxcrypt. We need the older libcrypt because the OpenShift
HAProxy build depends on it, and the newer libxcrypt is not fully
backward compatible with the original libcrypt.

## Build OpenSSL 1.1.1 on Fedora 40 (possibly RHEL?)

    dnf groupinstall -y "Development Tools"
    dnf install -y perl-core libffi-devel zlib-devel
    wget https://www.openssl.org/source/openssl-1.1.1v.tar.gz
    tar -xzvf openssl-1.1.1v.tar.gz
    cd openssl-1.1.1v
    ./config --prefix=/usr/local/openssl-1.1.1 --openssldir=/usr/local/openssl-1.1.1 shared zlib
    make
    make install
    echo "/usr/local/openssl-1.1.1/lib" | tee /etc/ld.so.conf.d/openssl-1.1.1.conf
    ldconfig

## Build libcrypt

    dnf groupinstall -y "Development Tools"
    dnf groupinstall -y "Development Libraries"
    dnf install -y autoconf automake libtool make gcc gettext-devel texinfo
    wget https://github.com/besser82/libxcrypt/archive/v4.4.10.tar.gz -O libxcrypt-4.4.10.tar.gz
    tar -xzvf libxcrypt-4.4.10.tar.gz
    cd libxcrypt-4.4.10
    # You will have to run boostrap once which will generate the m4/ directory.
    # Once the m4 directory has been created and ./boostrap fails, come back and run the sed.
    sed -i.bak 's/\$as_echo/AS_ECHO/g' m4/ax_pthread.m4
    ./bootstrap
    # Use -fcommon to allow multiple definitions of global variables.
    # This is necessary for compiling legacy OpenSSL 1.1 and the
    # traditional libcrypt code, which were originally written with
    # the older GCC default (-fcommon). Newer GCC versions use
    # -fno-common by default, which can cause linking errors due to
    # these multiple definitions.
    CFLAGS="-fcommon" ./configure --prefix=/usr/local/libcrypt --disable-xcrypt-compat-files --enable-obsolete-api=yes
    make
    make install
    echo "/usr/local/libcrypt/lib" | tee /etc/ld.so.conf.d/libcrypt.conf
    ldconfig

## Extract pre-built version of haproxy from an RPM

    dnf install rpm2cpio cpio
    rpm2cpio haproxy26-2.6.13-3.rhaos4.14.el8.x86_64.rpm | cpio -idmv './usr/sbin/haproxy'

    [root@master-0 ~]# cat /etc/redhat-release
    Fedora release 40 (Forty)

## Verify shared libraries

    [root@master-0 ~]# ldd ./usr/sbin/haproxy
        linux-vdso.so.1 (0x00007ffee9bf7000)
        libcrypt.so.1 => /usr/local/libcrypt/lib/libcrypt.so.1 (0x00007fc2637b3000)
        libz.so.1 => /lib64/libz.so.1 (0x00007fc263792000)
        libdl.so.2 => /lib64/libdl.so.2 (0x00007fc26378d000)
        librt.so.1 => /lib64/librt.so.1 (0x00007fc263788000)
        libpthread.so.0 => /lib64/libpthread.so.0 (0x00007fc263783000)
        libssl.so.1.1 => /usr/local/openssl-1.1.1/lib/libssl.so.1.1 (0x00007fc26316c000)
        libcrypto.so.1.1 => /usr/local/openssl-1.1.1/lib/libcrypto.so.1.1 (0x00007fc262e83000)
        libpcreposix.so.0 => /lib64/libpcreposix.so.0 (0x00007fc26377c000)
        libpcre.so.1 => /lib64/libpcre.so.1 (0x00007fc262e07000)
        libc.so.6 => /lib64/libc.so.6 (0x00007fc262c16000)
        /lib64/ld-linux-x86-64.so.2 (0x00007fc263815000)

    [root@master-0 ~]# ./usr/sbin/haproxy -v
        HAProxy version 2.6.13-234aa6d 2023/05/02 - https://haproxy.org/
        Status: long-term supported branch - will stop receiving fixes around Q2 2027.
        Known bugs: http://www.haproxy.org/bugs/bugs-2.6.13.html
        Running on: Linux 5.14.0-427.44.1.el9_4.x86_64 #1 SMP PREEMPT_DYNAMIC Fri Nov 1 14:40:56 EDT 2024 x86_64

### Switch between HAProxy versions without compromise

    [root@master-0 ~]# ldd /usr/sbin/haproxy-2.8
        linux-vdso.so.1 (0x00007fffcfbe1000)
        libcrypt.so.2 => /lib64/libcrypt.so.2 (0x00007fa9fc485000)
        libssl.so.3 => /lib64/libssl.so.3 (0x00007fa9fc3ae000)
        libcrypto.so.3 => /lib64/libcrypto.so.3 (0x00007fa9fbefe000)
        libz.so.1 => /lib64/libz.so.1 (0x00007fa9fbedd000)
        libpcreposix.so.0 => /lib64/libpcreposix.so.0 (0x00007fa9fbed8000)
        libpcre.so.1 => /lib64/libpcre.so.1 (0x00007fa9fbe5c000)
        libc.so.6 => /lib64/libc.so.6 (0x00007fa9fbc69000)
        /lib64/ld-linux-x86-64.so.2 (0x00007fa9fca25000)

    [root@master-0 ~]# /usr/sbin/haproxy-2.8 -v
    HAProxy version 2.8.10-f28885f 2024/06/14 - https://haproxy.org/
    Status: long-term supported branch - will stop receiving fixes around Q2 2028.
    Known bugs: http://www.haproxy.org/bugs/bugs-2.8.10.html
    Running on: Linux 5.14.0-427.44.1.el9_4.x86_64 #1 SMP PREEMPT_DYNAMIC Fri Nov 1 14:40:56 EDT 2024 x86_64
