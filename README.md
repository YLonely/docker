# Docker-Extra

The docker-extra branch is based on [moby](https://github.com/moby/moby) project, it modifies the code to allow docker pull from a extra image storage when downloading images.

Download the package from [release](https://github.com/kubesys/kubeext-extra/releases) page.

## Introduction

Pulling a big image from remote registry can be very slow, while a container only needs a little data to boot up accroding to some expriments, so it's effcient that docker have the access to some extra image layer storage, e.g. NFS layer storage, which stores uncompressed image data, then a container can be started just upon the layer storage and the data is requested through wire in a so called "Lazy" way.

docker-extra project provides a way to allow docker work with those extra (may be shared) storage, which speed up the creation of container.

## Features

We add some other options to dockerd's config file (/etc/docker/daemon.json):
```json
{
    "extra-storage":{
        "path":"path-to-the-extra-storage-dir-on-host[default:/var/lib/docker/extra]",
        "device":"path-to-the-device-to-be-mount-on-path",
        "type":"mount-type"
    }
}
```
If the extra storage is just a dir on host, then the `path` option is enough. If the extra storage is intalled as a device and should be mount on the dir `path`, users should specify the path of the device (e.g. /dev/sdb0) and also the type of the file system (e.g. ext4)

Once the extra-storage is ready, you can pull the image from extra storage with a "-extra" or "extra" flag at the end of the image tag. For example, pulling centos7 from extra storage:
```
docker pull centos:7-extra
```
or pulling the latest version of ubuntu from extra storage:
```
docker pull ubuntu:extra
```

## Extra Storage Requirment

File structure of extra storage dir should be like the one below

![file-structure](docs/img/file-structure.png)

It's easy to create a required extra storage structure, you can just copy the `image` and `overlay2` from `/var/lib/docker/`.

## Build

Merge all the code changes in `diff` or use fils in `full` to replace the corresponding files in projcet, then follow the steps [here](https://github.com/YLonely/docker/blob/master/docs/contributing/set-up-dev-env.md) to build this project.

## Build rpm packages

1. clone `docker-ce-packaging` project
```
git clone https://github.com/docker/docker-ce-packaging.git
```
2. enter subdir `rpm` of `docker-ce-packaging`
3. follow the steps [here](https://github.com/docker/docker-ce-packaging/tree/master/rpm) to create rpm packages for docker.