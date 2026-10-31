# Container Image

Every Insyra release publishes a container image with the `insyra` CLI of that release and a complete Go toolchain, so you can analyse data or run your own Go program without installing anything but Docker.

```bash
docker run --rm -it -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6
```

The container starts in `/work`, which the command above maps to the current directory, and opens the `insyra` REPL. Any other command runs in its place:

```bash
docker run --rm -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6 insyra run job.isr
docker run --rm -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6 go run .
```

## Tags

| Tag | Points to |
| --- | --- |
| `0.3.6` | That release. It never changes. |
| `0.3` | The highest stable release of the 0.3 line. |
| `latest` | The highest stable release. |
| `0.4.0-rc1` | A prerelease has only its own tag. |

Name the exact version in anything you want to reproduce later. The floating tags follow version numbers, so a fix released for an older line never moves `latest` back.

Every tag holds `linux/amd64` and `linux/arm64`; Docker picks the one that matches the host.

## What is inside

- The `insyra` CLI built from the release's tag.
- The official Go image (`golang:<version>-trixie`, Debian 13), pinned by digest. `GOTOOLCHAIN` is `local`, so a module whose `go.mod` asks for a newer Go than the image has fails to build instead of downloading one.
- The Vulkan loader, Mesa's Vulkan drivers for AMD and Intel GPUs, and `vulkaninfo`.

Debian packages are installed at build time and not pinned to a version, because Debian removes superseded versions from its mirrors. Each image's SBOM lists the packages it was built with.

## Using the GPU

`accel` reaches a GPU through Vulkan. Which hosts can pass one to a container:

| Host | GPU in the container |
| --- | --- |
| Linux with an NVIDIA GPU and the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) | Yes, with `--gpus all` |
| Linux with an AMD or Intel GPU | Yes, with `--device /dev/dri` |
| Docker Desktop on Windows (WSL 2) | No |
| Docker Desktop on macOS | No |

```bash
# NVIDIA
docker run --rm -it --gpus all -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6

# AMD or Intel
docker run --rm -it --device /dev/dri -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6
```

Check what the container sees:

```bash
insyra accel devices
vulkaninfo --summary
```

Without a GPU, `insyra accel devices` reports `no accel devices detected` and every operation runs on the CPU with the same results. `accel` never counts a software Vulkan driver such as llvmpipe as a GPU.

- **NVIDIA.** The image sets `NVIDIA_DRIVER_CAPABILITIES=compute,utility,graphics`. The toolkit mounts the host's Vulkan driver only with `graphics`, which its default (`compute,utility`) leaves out. If you pass `--gpus 'all,capabilities=…'` yourself, include `graphics`. The image also sets `NVIDIA_VISIBLE_DEVICES=all`, as NVIDIA's own CUDA images do, so on a host whose default Docker runtime is `nvidia` the container gets every GPU even without `--gpus`.
- **Windows.** Docker Desktop runs containers in WSL 2, where CUDA works but Vulkan does not: NVIDIA's WSL driver has no Vulkan driver, and Mesa's Vulkan-on-Direct3D 12 driver (Dozen) is not in Debian's packages.
- **macOS.** Docker Desktop passes no GPU to its virtual machine.

## Running as your own user

The container runs as root by default, so files it writes to `/work` belong to root on the host. Run it as your own user instead:

```bash
docker run --rm -it --user "$(id -u):$(id -g)" -v "$PWD":/work ghcr.io/hazelnutparadise/insyra:0.3.6
```

With `--device /dev/dri`, also add the group that owns the render device, which differs between hosts:

```bash
--group-add "$(stat -c %g /dev/dri/renderD128)"
```

## Keeping state between runs

`HOME` is `/home/insyra`, where the CLI keeps its environments (`~/.insyra`), and `GOCACHE` is `/go/cache`. Both are writable by any user. Mount volumes to keep them, and Go's module cache, between containers:

```bash
docker run --rm -it \
  -v insyra-home:/home/insyra \
  -v insyra-gomod:/go/pkg/mod \
  -v "$PWD":/work \
  ghcr.io/hazelnutparadise/insyra:0.3.6
```
