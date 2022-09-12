settings = {
    "trigger_mode": "manual",
    "capi_version": "v1.2.1",
}

settings.update(read_json(
    "tilt-settings.json",
    default = {},
)) 
envsubst_cmd = "./bin/envsubst"
update_settings(k8s_upsert_timeout_secs=60)
if settings.get("trigger_mode") == "manual":
   trigger_mode(TRIGGER_MODE_MANUAL)

if "allowed_contexts" in settings:
    allow_k8s_contexts(settings.get("allowed_contexts"))
else:
    allow_k8s_contexts(os.getenv('K8S_CONTEXT', 'phandalin'))

if "controller_image" in settings:
   docker_build(settings.get("controller_image"), '.')
   capo_cmd = "img=" + settings.get("controller_image") + ":latest make create-capm" 
else:
   docker_build(os.getenv('CONTROLLER_IMAGE', ''), '.')
   capo_cmd = "IMG=" + os.getenv('CONTROLLER_IMAGE','') + ":latest make create-capm"

def deploy_capi():
    version = settings.get("capi_version")
    capi = local('kubectl get deployment -n capi-system capi-controller-manager     --ignore-not-found')
    if not capi:
      capi_cmd = "MINIMUM_CLUSTERCTL_VERSION=${version} make deploy-clusterapi"
      local(capi_cmd)

def create_capm():
  capo_secret = local('kubectl get secret -n cluster-api-provider-outscale-system cluster-api-provider-outscale   --ignore-not-found')
  if not capo_secret:
    if os.getenv('OSC_ACCESS_KEY', "none") == "none":
      print("No AK/SK secret for capo")
      fail("Need to have OSC_SECRET_KEY and OSC_ACCESS_KEY environement variable")
  else: 
    cred_cmd = "make credential"
    local(cred_cmd)
    local(capo_cmd)
deploy_capi()
create_capm()
k8s_yaml('capm.yaml')
