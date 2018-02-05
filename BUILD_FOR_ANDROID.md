#Build for android

NDK is needed, make sure you have seted $NDK.


Follow these steps:

```sh
export standalone_toolchains = ~/standalone_toolchains

$NDK/build/tools/make-standalone-toolchain.sh --arch=arm --install-dir=$standalone_toolchains

export PATH=$standalone_toolchains/bin:$PATH

export CC=$standalone_toolchains/bin/arm-linux-androideabi-gcc.exe

export GOOS=android

export GOARCH=arm

export CGO_ENABLED=0

```

Then you can build go programs for android.