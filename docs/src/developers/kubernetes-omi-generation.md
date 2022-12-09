# Kubernetes Omi Generation
 
## Generation

Kubernetes image are created using [image-builder][image-builder] (with packer and ansible to generate kubernetes containerd image) with a cron github ci job run every month to create new kubernetes images.

To launch locally:

```bash
git clone https://github.com/kubernetes-sigs/image-builder
export OSC_ACCESS_KEY=access
export OSC_SECRET_KEY=secret
export OSC_REGION=region
cd images/capi/scripts/ci-outscale-nightly.sh
```

## Image Deprecation

Image will be deprecated after 6 months.

```bash
python3 hack/cleanup/cleanup_oapi.py --days 183 --owner my_owner --imageNameFilterPath ./keep_image --imageNamePattern "^(ubuntu|centos)-[0-9.]+-[0-9.]+-kubernetes-v[0-9]+.[0-9]{2}.[0-9]+-[0-9]{4}-[0-9]{2}-[0-9]{2}$"
```

keep_image is a file to keep image which match imageNamePattern and are older than 6 months.

example:
```
ubuntu-2004-2004-kubernetes-v1.25.2-2022-10-13
```

<!-- References -->
[image-builder]: https://cluster-api.sigs.k8s.io/developer/providers/implementers-guide/building_running_and_testing.html
