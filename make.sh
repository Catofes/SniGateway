#!/bin/bash

function try () {
"$@" || exit -1
}

[ -z "$ANDROID_NDK_HOME" ] && ANDROID_NDK_HOME=/opt/android-ndk/

DIR=$(pwd)
MIN_API=$1
DEPS=$(pwd)/.deps
ANDROID_ARM_TOOLCHAIN=$DEPS/toolchains/arm-$1
ANDROID_X86_TOOLCHAIN=$DEPS/toolchains/x86_64-$1

ANDROID_ARM_CC=$ANDROID_ARM_TOOLCHAIN/bin/arm-linux-androideabi-gcc
ANDROID_ARM_STRIP=$ANDROID_ARM_TOOLCHAIN/bin/arm-linux-androideabi-strip

ANDROID_X86_CC=$ANDROID_X86_TOOLCHAIN/bin/i686-linux-android-gcc
ANDROID_X86_STRIP=$ANDROID_X86_TOOLCHAIN/bin/i686-linux-android-strip

if [ ! -d "$ANDROID_ARM_TOOLCHAIN" ]; then
    echo "Make standalone toolchain for ARM arch"
    $ANDROID_NDK_HOME/build/tools/make_standalone_toolchain.py --arch arm \
        --api $MIN_API --install-dir $ANDROID_ARM_TOOLCHAIN
fi

if [ ! -d "$ANDROID_X86_TOOLCHAIN" ]; then
    echo "Make standalone toolchain for X86 arch"
    $ANDROID_NDK_HOME/build/tools/make_standalone_toolchain.py --arch x86 \
        --api $MIN_API --install-dir $ANDROID_X86_TOOLCHAIN
fi

if [ ! -d "$DIR/go/bin" ]; then
    echo "Build the custom go"
	export GOROOT=""
	export GOROOT_BOOTSTRAP=/usr/lib/go
    pushd $DIR/go/src
    try ./make.bash
    popd
fi

export GOROOT=$DIR/go
export GOPATH=/home/herbertqiao/Documents/gocode
export PATH=$GOROOT/bin:$PATH

#mkdir -p build/arm
#mkdir -p build/x86

cd $DIR/client/Android
echo "Cross compile kcptun for arm"
try env CGO_ENABLED=1 CC=$ANDROID_ARM_CC GOOS=android GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o client
try $ANDROID_ARM_STRIP client
try mv client $DIR/build/arm/libkcptun.so

echo "Cross compile kcptun for x86"
try env CGO_ENABLED=1 CC=$ANDROID_X86_CC GOOS=android GOARCH=386 go build -ldflags="-s -w" -o client
try $ANDROID_X86_STRIP client
try mv client $DIR/build/x86/libkcptun.so

echo "Successfully build kcptun"
