# hivefs - Demo fuse client for Elastos Hive Cluster

[![img](https://camo.githubusercontent.com/9ff0f4b787066b705774659143d8b88f485119ff/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f6d61646525323062792d456c6173746f732532306f72672d626c75652e7376673f7374796c653d666c61742d737175617265)](http://elastos.org)
[![img](https://camo.githubusercontent.com/85d19725dcd92c6f77a1d72a2e9b2b49c36489ab/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f70726f6a6563742d486976652d626c75652e7376673f7374796c653d666c61742d737175617265)](http://elastos.org/)
[![standard-readme compliant](https://camo.githubusercontent.com/a7e665f337914171fa0b60a110690af78fc5d943/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f7374616e646172642d2d726561646d652d4f4b2d677265656e2e7376673f7374796c653d666c61742d737175617265)](https://github.com/RichardLitt/standard-readme)
[![Build Status](https://camo.githubusercontent.com/d95d2cf5f0f2c8ebf5697026daaa4cbfaab6521e/68747470733a2f2f7472617669732d63692e6f72672f656c6173746f732f456c6173746f732e4e45542e486976652e495046532e7376673f6272616e63683d6d6173746572)](https://travis-ci.org/elastos/Elastos.NET.Hive.Cluster)

## Introduction

In a nutshell:
**hivefs** implements a fuse client for hive cluster.

**hivefs** is a fuse client that allows users to connect hive cluster servers using typical POSIX API.

for instance:
* You can use it to mount hive.cluster as a local disk driver or directory.
* You can read/write on the mounted like as on local file system.

### Background
- [FUSE](https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/Documentation/filesystems/fuse.txt)
(Filesystem In Userpace) is a Linux kernel filesystem that sends the
incoming requests over a file descriptor to userspace. Historically,
these have been served with a
[C library of the same name](http://fuse.sourceforge.net/), but
ultimately FUSE is just a protocol. Since then, the protocol has been
implemented for other platforms such as OS X, FreeBSD and OpenBSD.

- hivefs is fuse app/client, it uses the library [bazil.org/fuse](http://bazil.org/fuse), which is a reimplementation of that
protocol in pure Go.

## Pre-Requirements

In order to run this app, FUSE installation is needed.

Ubuntu Linux:
```sh
sudo apt-get update
sudo apt-get install fuse
```

macOS:  
Please download the installer package from:
[https://github.com/osxfuse/osxfuse/releases](https://github.com/osxfuse/osxfuse/releases)

## Compiling
hivefs is written in Go, so here’s how to get going:
```sh
git clone https://github.com/elastos/Elastos.NET.Hive.Demo.FuseApp.git  $GOPATH/src/github.com/elastos/Elastos.NET.Hive.Demo.FuseApp

cd $GOPATH/src/github.com/elastos/Elastos.NET.Hive.Demo.FuseApp
go get
make
```
## Usage

**hivefs** is a command line program that need to run in background. The usage is below:

```sh
./hivefs -host <host> -port <port number> -uid <uid> MOUNTPOINT
```
- host - hive cluster address. The default value is 127.0.0.1.
- port - hive cluster port. The default value is 9095.
- uid - uid string. The default value is generated by "uid/new" http API. 
- MOUNTPOINT - mount target directory.

For example:
```sh
./hivefs -host 127.0.0.1 -port 9095 -uid uid-ee978fa7-18b6-43d4-9f3e-3e6562131036 /mnt
```

While the hivefs daemon is running, please open a new terminal window and go to the mounted directory for your testing.

If you want to run hivefs in the background even after you close the terminal, then you can use *nohup* to run it. For example:
```sh
nohup ./hivefs -host 127.0.0.1 -port 9095 -uid uid-ee978fa7-18b6-43d4-9f3e-3e6562131036 /mnt &
```


## Troubleshooting


#### Mount errors

At first, make sure the mount target directory has been created and you have the write permission.

During starting the daemon, you may see mount errors below.

In Linux:
```
mount helper error: fusermount: failed to access mountpoint /mnt: Transport endpoint is not connected
```
In macOS:
```
mount helper error: mount_osxfuse: mount point /mnt is itself on a OSXFUSE volume
```
Please use the command *mount* to check whether the mount point is being used. If used, then umount it or choose another mount point.

If you ran hivefs before but didn't unload it with the *umount* command, then the previous mount point is still used by the system although hivefs is not running. Please umount it manually before using hivefs again.

#### Umount errors

Please use *umount* to unload hivefs if you don't use it. If you directly quit the hivefs daemon, the mount point will not be released automatically, and you may not mount it again with the same mount point.

These are errors for unsuccessful umount operations.

In Linux:
```sh
$ umount /mnt
umount: /mnt: target is busy.
```
In macOS:
```sh
$ umount /mnt
umount(/mnt): Resource busy -- try 'diskutil unmount'
```

Please do the following steps to solve this problem:
* Check whether one of the shell stays at the directory inside the mount point. Maybe your running shells have been left the mount point, but other people still use it.
* Check whether somes files or directories are opened by some applications.
It's recommended to use *lsof* to see which users and which processes still open files or directories inside hivefs volume.

#### Access errors

Sometimes you can't access the mounted directory even the hivefs daemon runs corrrectly. You may see the below errors. 

In Linux:
```sh
$ cd /mnt
bash: cd: /mnt: Permission denied
$ ls /mnt
ls: cannot access '/mnt': Permission denied
```
In macOS:
```sh
$ cd /mnt
cd: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
$ ls /mnt
ls: /mnt: No such file or directory
```
Probably it's related to the access right. Make sure the current user has the permission to access the mounted path. For example, if hivefs is running with *root* user, other users are difficult to access the corresponding directories. If possible, don't use *root* to run hivefs.


## Contribution

We welcome contributions to the Elastos Hive Project.

## Acknowledgments

A sincere thank you to all teams and projects that we rely on directly or indirectly.

## License
This project is licensed under the terms of the [MIT license](https://github.com/elastos/Elastos.Hive.Demo.FuseApp/blob/master/LICENSE).
